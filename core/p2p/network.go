package p2p

import (
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

// P2PNetwork represents a P2P network node
type P2PNetwork struct {
	host         host.Host
	id           string
	handlers     map[protocol.ID]MessageHandler
	fullHandlers map[protocol.ID]FullMessageHandler
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
		host:         h,
		id:           id,
		handlers:     make(map[protocol.ID]MessageHandler),
		fullHandlers: make(map[protocol.ID]FullMessageHandler),
	}, nil
}

// ID returns the node's ID
func (n *P2PNetwork) ID() string {
	return n.id
}

// RegisterProtocol registers a protocol handler
func (n *P2PNetwork) RegisterProtocol(proto protocol.ID, handler interface{}) {
	n.mu.Lock()
	defer n.mu.Unlock()

	switch h := handler.(type) {
	case MessageHandler:
		n.handlers[proto] = h
	case FullMessageHandler:
		n.fullHandlers[proto] = h
	default:
		panic("invalid handler type")
	}
}

// Broadcast broadcasts a message to all peers
func (n *P2PNetwork) Broadcast(ctx context.Context, proto protocol.ID, data []byte) error {
	peers := n.host.Network().Peers()
	for _, p := range peers {
		if err := n.Send(ctx, p, proto, data); err != nil {
			return fmt.Errorf("failed to send to peer %s: %w", p, err)
		}
	}
	return nil
}

// Send sends a message to a specific peer
func (n *P2PNetwork) Send(ctx context.Context, p peer.ID, proto protocol.ID, data []byte) error {
	s, err := n.host.NewStream(ctx, p, proto)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}
	defer s.Close()

	_, err = s.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to stream: %w", err)
	}

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

// MessageHandler is a function that handles incoming messages
type MessageHandler = func([]byte) error

// FullMessageHandler is a function that handles incoming messages with context and peer information
type FullMessageHandler = func(context.Context, peer.ID, []byte) error

// handleStream handles incoming streams
func (n *P2PNetwork) handleStream(s network.Stream) {
	defer s.Close()

	proto := s.Protocol()
	from := s.Conn().RemotePeer()

	// Read message data
	buf := make([]byte, 1024)
	bytesRead, err := s.Read(buf)
	if err != nil {
		fmt.Printf("Error reading from stream: %v\n", err)
		return
	}

	data := buf[:bytesRead]

	// Handle message based on protocol
	n.mu.RLock()
	handler := n.handlers[proto]
	fullHandler := n.fullHandlers[proto]
	n.mu.RUnlock()

	if handler != nil {
		if err := handler(data); err != nil {
			fmt.Printf("Error handling message: %v\n", err)
		}
	}

	if fullHandler != nil {
		if err := fullHandler(context.Background(), from, data); err != nil {
			fmt.Printf("Error handling message: %v\n", err)
		}
	}
}
