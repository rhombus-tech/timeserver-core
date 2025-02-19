package consensus

import (
	"time"
)

// NewState creates a new consensus state
func NewState(nodeID string, validators map[string]bool) *State {
	return &State{
		NodeID:     nodeID,
		Validators: validators,
		View:       0,
	}
}

// GetNodeID returns the node ID
func (s *State) GetNodeID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.NodeID
}

// GetView returns the current view number
func (s *State) GetView() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.View
}

// SetView sets the current view number
func (s *State) SetView(view uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.View = view
}

// GetLeader returns the current leader
func (s *State) GetLeader() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Leader
}

// SetLeader sets the current leader
func (s *State) SetLeader(leader string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Leader = leader
}

// GetLastTime returns the last consensus timestamp
func (s *State) GetLastTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastTime
}

// SetLastTime sets the last consensus timestamp
func (s *State) SetLastTime(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastTime = t
}

// GetLastDigest returns the last consensus digest
func (s *State) GetLastDigest() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastDigest
}

// SetLastDigest sets the last consensus digest
func (s *State) SetLastDigest(digest []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastDigest = digest
}

// GetValidators returns the validators map
func (s *State) GetValidators() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	validators := make(map[string]bool)
	for k, v := range s.Validators {
		validators[k] = v
	}
	return validators
}
