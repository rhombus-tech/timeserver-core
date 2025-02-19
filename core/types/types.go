package types

import (
	"crypto/ed25519"
	"encoding/binary"
	"time"
)

// SignedTimestamp represents a timestamp signed by a timeserver
type SignedTimestamp struct {
	ServerID   string            `json:"server_id"`
	Time       time.Time         `json:"time"`
	Signature  []byte           `json:"signature"`
	Nonce      []byte           `json:"nonce"`
	RegionID   string           `json:"region_id"`
}

// VerificationRequest represents a request for a timestamp
type VerificationRequest struct {
	Nonce     []byte `json:"nonce"`
	RegionID  string `json:"region_id"`
}

// VerificationResponse represents a response containing a signed timestamp
type VerificationResponse struct {
	Timestamp   *SignedTimestamp `json:"timestamp"`
	ServerProof []byte          `json:"server_proof"`
	Delay       time.Duration   `json:"delay"`
}

// TimeServer represents a single timeserver node
type TimeServer interface {
	// GetID returns the server's unique identifier
	GetID() string

	// GetPublicKey returns the server's public key
	GetPublicKey() ed25519.PublicKey

	// GetRegion returns the server's assigned region
	GetRegion() string

	// SetRegion sets the server's region
	SetRegion(regionID string) error

	// Sign signs a timestamp with the server's private key
	Sign(ts time.Time, nonce []byte) (*SignedTimestamp, error)

	// Verify verifies a signed timestamp
	Verify(ts *SignedTimestamp) error

	// Start starts the server
	Start() error

	// Stop stops the server
	Stop() error
}

// TimeNetwork represents a network of timeservers
type TimeNetwork interface {
	// AddServer adds a new timeserver to the network
	AddServer(server TimeServer) error

	// GetVerifiedTimestamp gets a timestamp verified by multiple servers
	GetVerifiedTimestamp(regionID string) (*SignedTimestamp, error)

	// GetRegionServers gets all servers in a region
	GetRegionServers(regionID string) ([]TimeServer, error)

	// Start starts the network
	Start() error

	// Stop stops the network
	Stop() error
}

// TimestampToBytes converts a timestamp to bytes for signing
func TimestampToBytes(t time.Time) []byte {
	nanos := t.UnixNano()
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(nanos))
	return buf
}

// BytesToTimestamp converts bytes back to a timestamp
func BytesToTimestamp(b []byte) time.Time {
	nanos := binary.BigEndian.Uint64(b)
	return time.Unix(0, int64(nanos))
}
