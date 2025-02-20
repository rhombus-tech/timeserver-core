package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/rhombus-tech/timeserver-core/core/metrics"
	"github.com/rhombus-tech/timeserver-core/core/types"
	"github.com/sirupsen/logrus"
)

// Network represents a P2P network
type Network interface {
	// Start starts the network
	Start(ctx context.Context) error

	// Stop stops the network
	Stop() error

	// GetPeer gets a peer by ID
	GetPeer(id string) (types.TimeServer, error)

	// GetPeersInRegion gets all peers in a region
	GetPeersInRegion(region string) ([]types.TimeServer, error)

	// Broadcast broadcasts a message to all peers
	Broadcast(ctx context.Context, proto protocol.ID, data []byte) error

	// Send sends a message to a specific peer
	Send(ctx context.Context, p peer.ID, proto protocol.ID, data []byte) error

	// RegisterProtocol registers a protocol handler
	RegisterProtocol(proto protocol.ID, handler func(context.Context, peer.ID, []byte) error)
}

// P2PNetwork represents a P2P network node
type P2PNetwork struct {
	host         host.Host
	id           string
	handlers     map[protocol.ID]func(context.Context, peer.ID, []byte) error
	mu           sync.RWMutex
}

// NewP2PNetwork creates a new P2P network node
func NewP2PNetwork(id string, privKey crypto.PrivKey, bootstrapPeers []string) (*P2PNetwork, error) {
	// Create libp2p host
	h, err := libp2p.New(libp2p.Identity(privKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	return &P2PNetwork{
		host:     h,
		id:       id,
		handlers: make(map[protocol.ID]func(context.Context, peer.ID, []byte) error),
	}, nil
}

// ID returns the node's ID
func (n *P2PNetwork) ID() string {
	return n.id
}

// RegisterProtocol registers a protocol handler
func (n *P2PNetwork) RegisterProtocol(proto protocol.ID, handler func(context.Context, peer.ID, []byte) error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.handlers[proto] = handler
}

// Broadcast broadcasts a message to all peers
func (n *P2PNetwork) Broadcast(ctx context.Context, proto protocol.ID, data []byte) error {
	start := time.Now()
	defer func() {
		metrics.RecordP2PStreamLatency(time.Since(start).Seconds(), "broadcast")
	}()

	peers := n.host.Network().Peers()
	for _, p := range peers {
		if err := n.Send(ctx, p, proto, data); err != nil {
			metrics.RecordP2PMessage(string(proto), "broadcast_error")
			return fmt.Errorf("failed to send to peer %s: %w", p, err)
		}
	}
	metrics.RecordP2PMessage(string(proto), "broadcast_success")
	return nil
}

// Send sends a message to a specific peer
func (n *P2PNetwork) Send(ctx context.Context, p peer.ID, proto protocol.ID, data []byte) error {
	start := time.Now()
	defer func() {
		metrics.RecordP2PStreamLatency(time.Since(start).Seconds(), "send")
	}()

	s, err := n.host.NewStream(ctx, p, proto)
	if err != nil {
		metrics.RecordP2PMessage(string(proto), "send_error")
		return fmt.Errorf("failed to create stream: %w", err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			logrus.WithError(err).Error("Error closing stream")
		}
	}()

	_, err = s.Write(data)
	if err != nil {
		metrics.RecordP2PMessage(string(proto), "write_error")
		return fmt.Errorf("failed to write to stream: %w", err)
	}

	metrics.RecordP2PMessage(string(proto), "send_success")
	return nil
}

// Connect connects to a peer using its multiaddr
func (n *P2PNetwork) Connect(ctx context.Context, addr string) error {
	// Parse the multiaddr
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return fmt.Errorf("invalid multiaddr %s: %w", addr, err)
	}

	// Extract the peer ID from the multiaddr
	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("failed to get peer info from multiaddr: %w", err)
	}

	// Connect to the peer
	if err := n.host.Connect(ctx, *info); err != nil {
		return fmt.Errorf("failed to connect to peer %s: %w", info.ID, err)
	}

	return nil
}

// Start starts the network node
func (n *P2PNetwork) Start(ctx context.Context) error {
	// Start protocol handlers
	for proto := range n.handlers {
		n.host.SetStreamHandler(proto, n.handleStream)
	}
	return nil
}

// Stop stops the network node
func (n *P2PNetwork) Stop() error {
	return n.host.Close()
}

// handleStream handles incoming streams
func (n *P2PNetwork) handleStream(s network.Stream) {
	start := time.Now()
	defer func() {
		metrics.RecordP2PStreamLatency(time.Since(start).Seconds(), "handle_stream")
		if err := s.Close(); err != nil {
			logrus.WithError(err).Error("Error closing stream")
		}
	}()

	proto := s.Protocol()
	from := s.Conn().RemotePeer()

	n.mu.RLock()
	handler, ok := n.handlers[proto]
	n.mu.RUnlock()

	if !ok {
		metrics.RecordP2PMessage(string(proto), "unknown_protocol")
		logrus.WithField("protocol", proto).Warn("No handler for protocol")
		return
	}

	// Read the message
	buf := make([]byte, 1024*1024) // 1MB buffer
	bytesRead, err := s.Read(buf)
	if err != nil {
		metrics.RecordP2PMessage(string(proto), "read_error")
		logrus.WithError(err).Error("Error reading from stream")
		return
	}

	// Handle the message
	if err := handler(context.Background(), from, buf[:bytesRead]); err != nil {
		metrics.RecordP2PMessage(string(proto), "handle_error")
		logrus.WithError(err).Error("Error handling message")
		return
	}

	metrics.RecordP2PMessage(string(proto), "handle_success")
}

// GetPeer gets a peer by ID
func (n *P2PNetwork) GetPeer(id string) (types.TimeServer, error) {
	// Not implemented
	return nil, nil
}

// GetPeersInRegion gets all peers in a region
func (n *P2PNetwork) GetPeersInRegion(region string) ([]types.TimeServer, error) {
	// Not implemented
	return nil, nil
}
