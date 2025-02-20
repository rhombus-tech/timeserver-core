package integration

import (
	"context"
	"crypto/ed25519"
	"testing"

	"github.com/rhombus-tech/timeserver-core/core/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiRegionTimestamps(t *testing.T) {
	// Generate keys for test servers
	_, privKey1, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	_, privKey2, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	// Create servers with empty bootstrap peers
	srv1, err := server.NewServer("server1", "region1", privKey1, nil)
	require.NoError(t, err)
	srv2, err := server.NewServer("server2", "region2", privKey2, nil)
	require.NoError(t, err)

	// Start servers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = srv1.Start(ctx)
	require.NoError(t, err)
	defer srv1.Stop()

	err = srv2.Start(ctx)
	require.NoError(t, err)
	defer srv2.Stop()

	// Add peers one at a time
	err = srv1.AddPeer(srv2)
	require.NoError(t, err)
	
	err = srv2.AddPeer(srv1)
	require.NoError(t, err)

	// Test getting timestamps from different regions
	ts1, err := srv1.GetFastPath().GetTimestamp(ctx)
	require.NoError(t, err)
	assert.Equal(t, "region1", ts1.RegionID)

	ts2, err := srv2.GetFastPath().GetTimestamp(ctx)
	require.NoError(t, err)
	assert.Equal(t, "region2", ts2.RegionID)

	// Verify timestamps across regions
	err = srv1.GetFastPath().VerifyTimestamp(ctx, ts2)
	require.NoError(t, err)

	err = srv2.GetFastPath().VerifyTimestamp(ctx, ts1)
	require.NoError(t, err)

	// Test getting network status
	status1, err := srv1.GetFastPath().GetStatus()
	require.NoError(t, err)
	assert.Equal(t, "region1", status1.Region)
	assert.Equal(t, 1, status1.PeerCount)
}
