package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rhombus-tech/timeserver-core/core/types"
)

// Message types
const (
	TypeTimestamp  = "timestamp"
	TypeConsensus  = "consensus"
	TypeViewChange = "viewchange"
)

// Protocol IDs
const (
	FastPathProtocol   = "/timeserver/fastpath/1.0.0"
	ConsensusProtocol  = "/timeserver/consensus/1.0.0"
)

// TimestampMessage represents a timestamp request/response
type TimestampMessage struct {
	Type      string
	From      peer.ID
	To        string
	SignedTS  *types.SignedTimestamp
}

// NewTimestampMessage creates a new timestamp message
func NewTimestampMessage(from peer.ID, to string, ts *types.SignedTimestamp) *TimestampMessage {
	return &TimestampMessage{
		Type:      TypeTimestamp,
		From:      from,
		To:        to,
		SignedTS:  ts,
	}
}

// ViewChangeMessage represents a view change message
type ViewChangeMessage struct {
	Type      string
	From      peer.ID
	To        string
	ViewID    uint64
	Timestamp *types.SignedTimestamp
}

// ConsensusMessage represents a consensus protocol message
type ConsensusMessage struct {
	Type      string
	From      peer.ID
	To        string
	ViewID    uint64
	Timestamp *types.SignedTimestamp
	Data      []byte
}

// EncodeMessage encodes a message to bytes
func EncodeMessage(msg interface{}) ([]byte, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}
	return data, nil
}

// DecodeMessage decodes bytes into a message
func DecodeMessage(data []byte, msg interface{}) error {
	if err := json.Unmarshal(data, msg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}
	return nil
}
