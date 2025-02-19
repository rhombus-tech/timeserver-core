package consensus

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/stretchr/testify/assert"
)

type mockNetwork struct {
	handlers map[protocol.ID]func(context.Context, peer.ID, []byte) error
}

func newMockNetwork() *mockNetwork {
	return &mockNetwork{
		handlers: make(map[protocol.ID]func(context.Context, peer.ID, []byte) error),
	}
}

func (m *mockNetwork) Broadcast(ctx context.Context, proto protocol.ID, data []byte) error {
	return nil
}

func (m *mockNetwork) Send(ctx context.Context, p peer.ID, proto protocol.ID, data []byte) error {
	return nil
}

func (m *mockNetwork) RegisterProtocol(id protocol.ID, handler func(context.Context, peer.ID, []byte) error) {
	m.handlers[id] = handler
}

func (m *mockNetwork) Start(ctx context.Context) error {
	return nil
}

func (m *mockNetwork) Stop() error {
	return nil
}

func TestConsensus(t *testing.T) {
	// Create mock network
	network := newMockNetwork()

	// Create consensus instance
	nodeID := "node1"
	validators := map[string]bool{
		"node1": true,
		"node2": true,
		"node3": true,
	}
	c := NewConsensus(nodeID, validators, network)

	// Start consensus
	ctx := context.Background()
	err := c.Start(ctx)
	assert.NoError(t, err)

	// Get state
	state := c.GetState()
	assert.Equal(t, nodeID, state.GetNodeID())
	assert.Equal(t, uint64(0), state.GetView())

	// Stop consensus
	err = c.Stop()
	assert.NoError(t, err)
}

func TestViewChange(t *testing.T) {
	// Create mock network
	network := newMockNetwork()

	// Create consensus instance
	nodeID := "node1"
	validators := map[string]bool{
		"node1": true,
		"node2": true,
		"node3": true,
	}
	c := NewConsensus(nodeID, validators, network)

	// Start consensus
	ctx := context.Background()
	err := c.Start(ctx)
	assert.NoError(t, err)

	// Process view change
	msg := &ConsensusMessage{
		Type:      "view_change",
		View:      1,
		From:      "node2",
		Timestamp: &SignedTimestamp{Time: time.Now()},
	}
	err = c.ProcessViewChange(ctx, msg)
	assert.Error(t, err) // Should return not implemented error

	// Stop consensus
	err = c.Stop()
	assert.NoError(t, err)
}
