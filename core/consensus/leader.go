package consensus

import (
	"crypto/sha256"
	"encoding/binary"
	"sort"
	"sync"
)

// LeaderElection manages the leader election process
type LeaderElection struct {
	mu        sync.RWMutex
	state     *State
	threshold int
}

// NewLeaderElection creates a new leader election instance
func NewLeaderElection(state *State, threshold int) *LeaderElection {
	return &LeaderElection{
		state:     state,
		threshold: threshold,
	}
}

// ElectLeader elects a leader for the given view
func (le *LeaderElection) ElectLeader(view uint64) string {
	le.mu.RLock()
	defer le.mu.RUnlock()

	// Get sorted list of validators
	validators := le.getValidators()
	if len(validators) == 0 {
		return ""
	}

	// Create deterministic hash from view number
	hash := sha256.New()
	viewBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(viewBytes, view)
	hash.Write(viewBytes)
	digest := hash.Sum(nil)

	// Use hash to select leader
	index := binary.BigEndian.Uint64(digest) % uint64(len(validators))
	return validators[index]
}

// getValidators returns a sorted list of validator IDs
func (le *LeaderElection) getValidators() []string {
	validators := make([]string, 0, len(le.state.Validators))
	for id := range le.state.Validators {
		validators = append(validators, id)
	}
	sort.Strings(validators)
	return validators
}

// IsLeader checks if the given node is the leader for the current view
func (le *LeaderElection) IsLeader(nodeID string) bool {
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
