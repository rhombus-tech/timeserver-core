package server

import (
	"context"
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	// Generate test key pair
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	require.NotNil(t, pub)
	require.NotNil(t, priv)

	// Create server
	s, err := NewServer("test", priv, nil)
	require.NoError(t, err)
	require.NotNil(t, s)

	// Check server fields
	assert.Equal(t, "test", s.GetID())
	assert.Equal(t, pub, s.GetPublicKey())
}

func TestServerSignVerify(t *testing.T) {
	// Generate test key pair
	_, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	// Create server
	s, err := NewServer("test", priv, nil)
	require.NoError(t, err)

	// Test signing
	data := []byte("test data")
	sig := s.Sign(data)
	require.NotNil(t, sig)

	// Test verification
	valid := s.Verify(data, sig)
	assert.True(t, valid)

	// Test invalid signature
	invalid := s.Verify([]byte("wrong data"), sig)
	assert.False(t, invalid)
}

func TestServerStartStop(t *testing.T) {
	// Generate test key pair
	_, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	// Create server
	s, err := NewServer("test", priv, nil)
	require.NoError(t, err)

	// Test start
	ctx := context.Background()
	err = s.Start(ctx)
	require.NoError(t, err)

	// Test double start
	err = s.Start(ctx)
	assert.Error(t, err)

	// Test stop
	err = s.Stop()
	require.NoError(t, err)

	// Test double stop
	err = s.Stop()
	assert.Error(t, err)
}

func TestGetTimestamp(t *testing.T) {
	// Generate test key pair
	_, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	// Create server
	s, err := NewServer("test", priv, nil)
	require.NoError(t, err)

	// Start server
	ctx := context.Background()
	err = s.Start(ctx)
	require.NoError(t, err)
	defer s.Stop()

	// Get timestamp
	ts, err := s.GetTimestamp(ctx)
	require.NoError(t, err)
	require.NotNil(t, ts)

	// Verify timestamp
	assert.Equal(t, s.GetID(), ts.ServerID)
	assert.True(t, ts.Time.Before(time.Now()))
	assert.True(t, ts.Time.After(time.Now().Add(-time.Second)))
}
