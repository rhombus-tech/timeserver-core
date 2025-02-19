package server

import (
	"encoding/binary"
	"encoding/json"
	"time"

	"github.com/libp2p/go-libp2p/core/protocol"
)

const (
	FastPathProtocol = protocol.ID("/timeserver/fastpath/1.0.0")
)

// FastPathRequest represents a fast path timestamp request
type FastPathRequest struct {
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
	From      string    `json:"from"`
}

// FastPathResponse represents a fast path timestamp response
type FastPathResponse struct {
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
	Signature []byte    `json:"signature"`
}

// TimestampToBytes converts a timestamp to bytes for signing
func TimestampToBytes(t time.Time) []byte {
	nanos := t.UnixNano()
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(nanos))
	return buf
}

// Encode encodes a FastPathRequest to bytes
func (r *FastPathRequest) Encode() []byte {
	data, _ := json.Marshal(r)
	return data
}

// Decode decodes a FastPathRequest from bytes
func (r *FastPathRequest) Decode(data []byte) error {
	return json.Unmarshal(data, r)
}

// Encode encodes a FastPathResponse to bytes
func (r *FastPathResponse) Encode() []byte {
	data, _ := json.Marshal(r)
	return data
}

// Decode decodes a FastPathResponse from bytes
func (r *FastPathResponse) Decode(data []byte) error {
	return json.Unmarshal(data, r)
}

// generateRequestID generates a unique request ID
func generateRequestID(t time.Time) string {
	return t.Format("20060102150405.000000000")
}
