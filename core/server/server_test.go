package server

import (
	"context"
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_Basic(t *testing.T) {
	// Generate test key pair
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	// Create server
	s, err := NewServer("test", "us-east", priv, nil)
	require.NoError(t, err)

	// Check basic properties
	assert.Equal(t, "test", s.GetID())
	assert.Equal(t, "us-east", s.GetRegion())
	assert.Equal(t, pub, s.GetPublicKey())

	// Test signing
	msg := []byte("test message")
	sig, err := s.Sign(msg)
	require.NoError(t, err)

	// Test verification
	err = s.Verify(msg, sig)
	require.NoError(t, err)
}

func TestServer_Network(t *testing.T) {
	// Generate test key pairs
	_, priv1, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	_, priv2, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	// Create servers with empty bootstrap peers
	s1, err := NewServer("server1", "us-east", priv1, nil)
	require.NoError(t, err)

	s2, err := NewServer("server2", "us-west", priv2, nil)
	require.NoError(t, err)

	// Start servers
	ctx := context.Background()
	err = s1.Start(ctx)
	require.NoError(t, err)
	defer s1.Stop()

	err = s2.Start(ctx)
	require.NoError(t, err)
	defer s2.Stop()

	// Add peers one at a time
	err = s1.AddPeer(s2)
	require.NoError(t, err)

	// Check peers on s1
	peers1 := s1.GetPeers()
	assert.Len(t, peers1, 1)
	assert.Equal(t, "server2", peers1[0].GetID())
}

func TestServer_FastPath(t *testing.T) {
	// Generate test key pair
	_, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	// Create server
	s, err := NewServer("test", "us-east", priv, nil)
	require.NoError(t, err)

	// Start server
	ctx := context.Background()
	err = s.Start(ctx)
	require.NoError(t, err)
	defer s.Stop()

	// Get timestamp
	ts, err := s.GetFastPath().GetTimestamp(ctx)
	require.NoError(t, err)
	assert.Equal(t, "test", ts.ServerID)
	assert.Equal(t, "us-east", ts.RegionID)
	assert.WithinDuration(t, time.Now(), ts.Time, time.Second)

	// Verify timestamp
	err = s.GetFastPath().VerifyTimestamp(ctx, ts)
	require.NoError(t, err)
}
