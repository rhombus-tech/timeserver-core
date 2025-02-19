package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const (
	gossipProtocol    = protocol.ID("/timeserver/gossip/1.0.0")
	gossipInterval    = 1 * time.Second
	messageExpiration = 5 * time.Minute
)

// GossipMessage represents a gossip protocol message
type GossipMessage struct {
	Type      string    // Message type
	From      peer.ID   // Sender ID
	Data      []byte    // Message data
	Timestamp time.Time // Message timestamp
}

// Gossip manages the gossip protocol
type Gossip struct {
	network     *P2PNetwork
	handlers    map[string]func(context.Context, peer.ID, []byte) error
	seenMsgs    map[string]bool
	mu          sync.RWMutex
	done        chan struct{}
}

// NewGossip creates a new gossip instance
func NewGossip(network *P2PNetwork) *Gossip {
	return &Gossip{
		network:  network,
		handlers: make(map[string]func(context.Context, peer.ID, []byte) error),
		seenMsgs: make(map[string]bool),
		done:     make(chan struct{}),
	}
}

// Start starts the gossip protocol
func (g *Gossip) Start(ctx context.Context) error {
	// Register gossip protocol handler
	g.network.RegisterProtocol(gossipProtocol, g.handleMessage)
	go g.cleanupSeenMessages()
	return nil
}

// Stop stops the gossip protocol
func (g *Gossip) Stop() error {
	close(g.done)
	return nil
}

// RegisterHandler registers a message handler
func (g *Gossip) RegisterHandler(msgType string, handler func(context.Context, peer.ID, []byte) error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.handlers[msgType] = handler
}

// Broadcast broadcasts a message to all peers
func (g *Gossip) Broadcast(ctx context.Context, msgType string, data []byte) error {
	msg := &GossipMessage{
		Type:      msgType,
		From:      g.network.host.ID(),
		Data:      data,
		Timestamp: time.Now(),
	}

	// Encode message
	msgData, err := EncodeMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Broadcast to all peers
	return g.network.Broadcast(ctx, gossipProtocol, msgData)
}

// handleMessage handles incoming gossip messages
func (g *Gossip) handleMessage(ctx context.Context, from peer.ID, data []byte) error {
	var msg GossipMessage
	if err := DecodeMessage(data, &msg); err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}

	// Check if we've seen this message before
	msgID := fmt.Sprintf("%s-%s-%d", msg.Type, msg.From, msg.Timestamp.UnixNano())
	g.mu.Lock()
	if g.seenMsgs[msgID] {
		g.mu.Unlock()
		return nil
	}
	g.seenMsgs[msgID] = true
	g.mu.Unlock()

	// Find handler for message type
	g.mu.RLock()
	handler := g.handlers[msg.Type]
	g.mu.RUnlock()

	if handler != nil {
		if err := handler(ctx, from, msg.Data); err != nil {
			return fmt.Errorf("failed to handle message: %w", err)
		}
	}

	// Forward message to other peers
	msgData, err := EncodeMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	peers := g.network.host.Network().Peers()
	for _, p := range peers {
		if p != from && p != g.network.host.ID() {
			if err := g.network.Send(ctx, p, gossipProtocol, msgData); err != nil {
				// Log error but continue forwarding to other peers
				fmt.Printf("failed to forward message to peer %s: %v\n", p, err)
			}
		}
	}

	return nil
}

// cleanupSeenMessages periodically removes old seen messages
func (g *Gossip) cleanupSeenMessages() {
	ticker := time.NewTicker(messageExpiration)
	defer ticker.Stop()

	for {
		select {
		case <-g.done:
			return
		case <-ticker.C:
			g.mu.Lock()
			g.seenMsgs = make(map[string]bool)
			g.mu.Unlock()
		}
	}
}
