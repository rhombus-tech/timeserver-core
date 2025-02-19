package consensus

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// NetworkInterface defines the interface for network operations
type NetworkInterface interface {
	Broadcast(ctx context.Context, proto protocol.ID, data []byte) error
	Send(ctx context.Context, p peer.ID, proto protocol.ID, data []byte) error
	RegisterProtocol(id protocol.ID, handler func(context.Context, peer.ID, []byte) error)
	Start(ctx context.Context) error
	Stop() error
}

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

// SignedTimestamp represents a timestamp signed by a server
type SignedTimestamp struct {
	Time      time.Time // Timestamp
	ServerID  string    // ID of the server that created the timestamp
	Signature []byte    // Signature of the timestamp
}

// ConsensusMessage represents a consensus protocol message
type ConsensusMessage struct {
	Type      string           // Message type
	View      uint64           // View number
	From      string           // Sender ID
	To        string           // Recipient ID
	Timestamp *SignedTimestamp // Message timestamp
	Data      []byte          // Message data
	Signature []byte          // Message signature
}
