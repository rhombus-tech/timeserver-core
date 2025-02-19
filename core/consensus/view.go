package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rhombus-tech/timeserver-core/core/types"
)

// ViewChange manages the view change process
type ViewChange struct {
	mu       sync.RWMutex
	state    *ConsensusState
	network  Network
	viewID   uint64
	messages map[uint64][]*ConsensusMessage
}

// NewViewChange creates a new view change instance
func NewViewChange(state *ConsensusState, network Network) *ViewChange {
	return &ViewChange{
		state:    state,
		network:  network,
		messages: make(map[uint64][]*ConsensusMessage),
	}
}

// StartViewChange initiates a view change
func (vc *ViewChange) StartViewChange(ctx context.Context) error {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	// Increment view ID
	vc.viewID++

	// Create view change message
	msg := &ConsensusMessage{
		Type: "viewchange",
		View: vc.viewID,
		From: vc.state.NodeID,
		Timestamp: &types.SignedTimestamp{
			Time:     time.Now(),
			ServerID: vc.state.NodeID,
		},
	}

	// Add message to view change messages
	vc.messages[vc.viewID] = append(vc.messages[vc.viewID], msg)

	// Broadcast view change
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to encode view change message: %w", err)
	}

	if err := vc.network.Broadcast(ctx, ConsensusProtocolID, data); err != nil {
		return fmt.Errorf("failed to broadcast view change message: %w", err)
	}

	return nil
}

// ProcessViewChangeMessage processes a view change message
func (vc *ViewChange) ProcessViewChangeMessage(ctx context.Context, data []byte) error {
	var msg ConsensusMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal view change message: %w", err)
	}

	vc.mu.Lock()
	defer vc.mu.Unlock()

	// Add message to view change messages
	vc.messages[msg.View] = append(vc.messages[msg.View], &msg)

	// Check if we have enough messages for this view
	if len(vc.messages[msg.View]) >= (len(vc.state.Validators)/2 + 1) {
		// We have a quorum, update the view
		vc.viewID = msg.View
		return nil
	}

	return nil
}

// GetCurrentView returns the current view ID
func (vc *ViewChange) GetCurrentView() uint64 {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.viewID
}
