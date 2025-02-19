package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rhombus-tech/timeserver-core/core/types"
)

// Consensus represents the consensus engine
type Consensus struct {
	state    *State
	network  Network
	mu       sync.RWMutex
}

// NewConsensus creates a new consensus instance
func NewConsensus(state *State, network Network) *Consensus {
	return &Consensus{
		state:   state,
		network: network,
	}
}

// Start starts the consensus engine
func (c *Consensus) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Register consensus protocol handler
	c.network.RegisterProtocol(ConsensusProtocolID, c.handleNetworkMessage)

	return c.network.Start(ctx)
}

// Stop stops the consensus engine
func (c *Consensus) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.network.Stop()
}

// handleNetworkMessage handles incoming network messages
func (c *Consensus) handleNetworkMessage(ctx context.Context, from peer.ID, data []byte) error {
	var msg ConsensusMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal consensus message: %w", err)
	}

	return c.handleMessage(ctx, msg.From, data)
}

// handleMessage handles incoming consensus messages
func (c *Consensus) handleMessage(ctx context.Context, from string, data []byte) error {
	var msg ConsensusMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal consensus message: %w", err)
	}

	// TODO: Implement message handling
	return nil
}

// ProposeTimestamp proposes a new timestamp for consensus
func (c *Consensus) ProposeTimestamp(ctx context.Context, ts *types.SignedTimestamp) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create proposal message
	msg := &ConsensusMessage{
		Type:      "propose",
		View:      c.state.GetView(),
		From:      c.state.GetNodeID(),
		Timestamp: ts,
	}

	// Broadcast proposal
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal proposal message: %w", err)
	}

	return c.network.Broadcast(ctx, ConsensusProtocolID, data)
}

// GetState returns the current consensus state
func (c *Consensus) GetState() *State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}
