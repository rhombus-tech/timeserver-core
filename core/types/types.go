package types

import (
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrServerNotFound   = errors.New("server not found")
	ErrRegionNotFound   = errors.New("region not found")
	ErrServerExists     = errors.New("server already exists")
)

// TimeServer represents a server in the time network
type TimeServer interface {
	GetID() string
	GetRegion() string
	SetRegion(region string) error
	GetPublicKey() ed25519.PublicKey
	Sign(message []byte) ([]byte, error)
	Verify(message []byte, signature []byte) error
}

// TimeNetwork represents the network of time servers
type TimeNetwork interface {
	AddServer(server TimeServer) error
	RemoveServer(serverID string) error
	GetServers() []TimeServer
	GetServersByRegion(region string) []TimeServer
	GetStatus() (*NetworkStatus, error)
	Start() error
	Stop() error
}

// SignedTimestamp represents a timestamp that has been signed by a time server
type SignedTimestamp struct {
	Time      time.Time
	ServerID  string
	RegionID  string
	Signature []byte
}

// Verify checks if the signature is valid for this timestamp
func (st *SignedTimestamp) Verify(server TimeServer) error {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, uint64(st.Time.UnixNano()))
	return server.Verify(data, st.Signature)
}

// String returns a string representation of the signed timestamp
func (st *SignedTimestamp) String() string {
	return fmt.Sprintf("SignedTimestamp{Time: %v, ServerID: %s, RegionID: %s}", st.Time, st.ServerID, st.RegionID)
}

// NetworkStatus represents the current status of the time network
type NetworkStatus struct {
	Status      string
	Region      string
	PeerCount   int
	LatestRound uint64
}

// ValidatorInfo represents information about a validator in the network
type ValidatorInfo struct {
	ID     string
	Region string
	Status string
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

// Sign signs a timestamp using the provided server
func (s *SignedTimestamp) Sign(server TimeServer) error {
	message := TimestampToBytes(s.Time)
	sig, err := server.Sign(message)
	if err != nil {
		return fmt.Errorf("failed to sign timestamp: %w", err)
	}
	s.Signature = sig
	s.ServerID = server.GetID()
	s.RegionID = server.GetRegion()
	return nil
}
