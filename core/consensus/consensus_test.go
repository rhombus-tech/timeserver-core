package consensus

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/stretchr/testify/assert"
)

type mockNetwork struct {
	protocolHandlers map[protocol.ID]func(context.Context, peer.ID, []byte) error
	started         bool
}

func newMockNetwork() *mockNetwork {
	return &mockNetwork{
		protocolHandlers: make(map[protocol.ID]func(context.Context, peer.ID, []byte) error),
	}
}

func (m *mockNetwork) RegisterProtocol(id protocol.ID, handler func(context.Context, peer.ID, []byte) error) {
	m.protocolHandlers[id] = handler
}

func (m *mockNetwork) Broadcast(ctx context.Context, proto protocol.ID, data []byte) error {
	return nil
}

func (m *mockNetwork) Send(ctx context.Context, p peer.ID, proto protocol.ID, data []byte) error {
	return nil
}

func (m *mockNetwork) Start(ctx context.Context) error {
	m.started = true
	return nil
}

func (m *mockNetwork) Stop() error {
	m.started = false
	return nil
}

func TestConsensus(t *testing.T) {
	nodeID := "node1"
	validators := map[string]bool{
		"node1": true,
		"node2": true,
		"node3": true,
	}
	network := newMockNetwork()
	state := NewState(nodeID, validators)
	consensus := NewConsensus(state, network)

	// Test Start
	ctx := context.Background()
	err := consensus.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, network.started)

	// Test Stop
	err = consensus.Stop()
	assert.NoError(t, err)
	assert.False(t, network.started)
}

func TestConsensusProposal(t *testing.T) {
	nodeID := "node1"
	validators := map[string]bool{
		"node1": true,
		"node2": true,
		"node3": true,
	}
	network := newMockNetwork()
	state := NewState(nodeID, validators)
	consensus := NewConsensus(state, network)

	// Test proposal
	ctx := context.Background()
	err := consensus.Start(ctx)
	assert.NoError(t, err)

	// TODO: Add more specific proposal tests
}
