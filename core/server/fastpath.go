package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rhombus-tech/timeserver-core/core/types"
)

// FastPath handles fast path timestamp requests
type FastPath struct {
	server *Server
	mu     sync.RWMutex
}

// NewFastPath creates a new FastPath instance
func NewFastPath(server *Server) *FastPath {
	return &FastPath{
		server: server,
		mu:     sync.RWMutex{},
	}
}

// GetTimestamp gets a signed timestamp via fast path
func (fp *FastPath) GetTimestamp(ctx context.Context) (*types.SignedTimestamp, error) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	// Generate timestamp
	now := time.Now()
	ts := &types.SignedTimestamp{
		Time:     now,
		ServerID: fp.server.GetID(),
		RegionID: fp.server.GetRegion(),
	}

	// Sign timestamp
	err := ts.Sign(fp.server)
	if err != nil {
		return nil, fmt.Errorf("failed to sign timestamp: %w", err)
	}

	return ts, nil
}

// VerifyTimestamp verifies a timestamp from another server
func (fp *FastPath) VerifyTimestamp(ctx context.Context, ts *types.SignedTimestamp) error {
	// Check if the timestamp was signed by this server
	if ts.ServerID == fp.server.GetID() {
		return ts.Verify(fp.server)
	}
	
	// Find the server that signed the timestamp
	peers := fp.server.GetPeers()
	for _, peer := range peers {
		if peer.GetID() == ts.ServerID {
			return ts.Verify(peer)
		}
	}
	return fmt.Errorf("server %s not found", ts.ServerID)
}

// GetStatus returns the current fast path status
func (fp *FastPath) GetStatus() (*types.NetworkStatus, error) {
	return &types.NetworkStatus{
		Status:      "healthy",
		Region:      fp.server.GetRegion(),
		PeerCount:   len(fp.server.GetPeers()),
		LatestRound: 0,
	}, nil
}

// GetValidators returns the list of validators
func (fp *FastPath) GetValidators() ([]*types.ValidatorInfo, error) {
	peers := fp.server.GetPeers()
	validators := make([]*types.ValidatorInfo, len(peers))
	for i, peer := range peers {
		validators[i] = &types.ValidatorInfo{
			ID:     peer.GetID(),
			Region: peer.GetRegion(),
			Status: "active",
		}
	}
	return validators, nil
}

// GetRegion returns the server's region
func (fp *FastPath) GetRegion() string {
	return fp.server.GetRegion()
}

// SetRegion sets the server's region
func (fp *FastPath) SetRegion(region string) {
	fp.server.SetRegion(region)
}
