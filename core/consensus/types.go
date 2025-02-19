package consensus

import (
    "sync"
    "time"

    "github.com/rhombus-tech/timeserver-core/core/types"
)

// State represents the consensus state
type State struct {
    mu         sync.RWMutex
    NodeID     string            // Node ID
    View       uint64            // Current view number
    Validators map[string]bool   // Set of validator IDs
    Leader     string           // Current leader ID
    LastDigest []byte          // Last consensus digest
    LastTime   time.Time       // Last consensus timestamp
}

// ConsensusMessage represents a consensus protocol message
type ConsensusMessage struct {
    Type      string                 // Message type (propose, vote, commit, view_change)
    View      uint64                 // View number
    From      string                 // Sender ID
    To        string                 // Recipient ID (empty for broadcast)
    Timestamp *types.SignedTimestamp // Message timestamp
    Data      []byte                // Message data (batch, vote, etc)
    Signature []byte                // Message signature
}

// TimestampBatch represents a batch of timestamps to be ordered
type TimestampBatch struct {
    Timestamps []*types.SignedTimestamp
    BatchID    uint64
    Leader     string
    FastPath   bool    // Added for fast path support
}
