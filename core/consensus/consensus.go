package consensus

import (
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
)

// Consensus implements the consensus protocol
type Consensus struct {
	mu       sync.RWMutex
	state    *State
	network  NetworkInterface
	election *ValidatorElection
}

// NewConsensus creates a new consensus instance
func NewConsensus(nodeID string, validators map[string]bool, network NetworkInterface) *Consensus {
	state := NewState(nodeID, validators)
	return &Consensus{
		state:    state,
		network:  network,
		election: NewValidatorElection(state),
	}
}

// Start starts the consensus protocol
func (c *Consensus) Start(ctx context.Context) error {
	c.network.RegisterProtocol(ConsensusProtocolID, c.handleMessage)
	return nil
}

// Stop stops the consensus protocol
func (c *Consensus) Stop() error {
	return nil
}

// handleMessage handles incoming consensus messages
func (c *Consensus) handleMessage(ctx context.Context, from peer.ID, data []byte) error {
	var msg ConsensusMessage
	// TODO: Implement message handling
	_ = msg // Silence unused variable warning
	return fmt.Errorf("not implemented")
}

// GetState returns the current consensus state
func (c *Consensus) GetState() *State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// ProcessViewChange processes a view change
func (c *Consensus) ProcessViewChange(ctx context.Context, msg *ConsensusMessage) error {
	// TODO: Implement view change
	return fmt.Errorf("not implemented")
}
