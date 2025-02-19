package consensus

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// ValidatorElection manages validator election
type ValidatorElection struct {
	state      *State
	validators []string
}

// NewValidatorElection creates a new validator election instance
func NewValidatorElection(state *State) *ValidatorElection {
	// Convert validator map to sorted slice for deterministic leader selection
	validators := make([]string, 0, len(state.GetValidators()))
	for v := range state.GetValidators() {
		validators = append(validators, v)
	}
	sort.Strings(validators)
	
	return &ValidatorElection{
		state:      state,
		validators: validators,
	}
}

// GetValidatorForView returns the validator for a given view
func (ve *ValidatorElection) GetValidatorForView(view uint64) string {
	if len(ve.validators) == 0 {
		return ""
	}
	return ve.validators[int(view)%len(ve.validators)]
}

// StartViewChange initiates a view change
func (ve *ValidatorElection) StartViewChange(ctx context.Context) error {
	ve.state.mu.Lock()
	defer ve.state.mu.Unlock()

	// Increment view number
	newView := ve.state.View + 1

	// Calculate new leader
	validators := ve.state.GetValidators()
	var validatorIDs []string
	for id := range validators {
		validatorIDs = append(validatorIDs, id)
	}

	// Sort validator IDs to ensure consistent leader selection
	sort.Strings(validatorIDs)

	// Select leader based on view number
	newLeader := validatorIDs[newView%uint64(len(validatorIDs))]

	// Update state
	ve.state.View = newView
	ve.state.Leader = newLeader

	// Create view change message
	msg := &ConsensusMessage{
		Type: "view_change",
		View: newView,
		From: ve.state.NodeID,
		To:   newLeader,
		Timestamp: &SignedTimestamp{
			Time:     time.Now(),
			ServerID: ve.state.NodeID,
		},
	}

	// TODO: Broadcast message to all validators
	_ = msg // Silence unused variable warning
	return fmt.Errorf("not implemented")
}
