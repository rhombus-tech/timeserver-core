package consensus

import (
	"crypto/sha256"
	"encoding/binary"
	"sort"
	"sync"
)

// LeaderElection manages leader election
type LeaderElection struct {
	mu        sync.RWMutex
	state      *State
	threshold int
	validators []string
}

// NewLeaderElection creates a new leader election instance
func NewLeaderElection(state *State, threshold int) *LeaderElection {
	// Convert validator map to sorted slice for deterministic leader selection
	validators := make([]string, 0, len(state.GetValidators()))
	for v := range state.GetValidators() {
		validators = append(validators, v)
	}
	sort.Strings(validators)

	return &LeaderElection{
		state:      state,
		threshold: threshold,
		validators: validators,
	}
}

// ElectLeader elects a leader for the given view
func (le *LeaderElection) ElectLeader(view uint64) string {
	le.mu.RLock()
	defer le.mu.RUnlock()

	if len(le.validators) == 0 {
		return ""
	}

	// Create deterministic hash from view number
	hash := sha256.New()
	viewBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(viewBytes, view)
	hash.Write(viewBytes)
	digest := hash.Sum(nil)

	// Use hash to select leader
	index := binary.BigEndian.Uint64(digest) % uint64(len(le.validators))
	return le.validators[index]
}

// GetLeaderForView returns the leader for a given view
func (le *LeaderElection) GetLeaderForView(view uint64) string {
	le.mu.RLock()
	defer le.mu.RUnlock()

	if len(le.validators) == 0 {
		return ""
	}
	return le.validators[int(view)%len(le.validators)]
}

// IsLeader returns true if the current node is the leader for the given view
func (le *LeaderElection) IsLeader(view uint64) bool {
	le.mu.RLock()
	defer le.mu.RUnlock()
	return le.GetLeaderForView(view) == le.state.GetNodeID()
}

// IsLeader checks if the given node is the leader for the current view
func (le *LeaderElection) IsLeaderNode(nodeID string) bool {
	le.mu.RLock()
	defer le.mu.RUnlock()
	return le.state.Leader == nodeID
}

// GetLeader returns the current leader
func (le *LeaderElection) GetLeader() string {
	le.mu.RLock()
	defer le.mu.RUnlock()
	return le.state.Leader
}
