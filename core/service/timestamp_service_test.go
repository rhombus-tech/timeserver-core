package service

import (
	"context"
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/rhombus-tech/timeserver-core/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTimestampService_GetTimestamp(t *testing.T) {
	// Create mock server
	mockServer := newMockTimestampServer("test", "test", nil)
	net := newMockTimeNetwork()
	mockServer.On("GetID").Return("test").Maybe()
	mockServer.On("GetRegion").Return("test").Maybe()
	net.On("AddServer", mockServer).Return(nil)
	require.NoError(t, net.AddServer(mockServer))
	ts := NewTimestampService(mockServer, net, "test")

	// Setup expectations
	now := time.Now()
	expectedTS := &types.SignedTimestamp{
		Time:      now,
		ServerID:  "test",
		RegionID:  "test",
		Signature: []byte("test-signature"),
	}
	mockServer.On("GetTimestamp", mock.Anything).Return(expectedTS, nil)

	// Test GetTimestamp
	result, err := ts.GetTimestamp(context.Background())
	require.NoError(t, err)
	assert.Equal(t, expectedTS, result)

	mockServer.AssertExpectations(t)
}

func TestTimestampService_VerifyTimestamp(t *testing.T) {
	// Create mock server
	mockServer := newMockTimestampServer("test", "test", nil)
	net := newMockTimeNetwork()
	mockServer.On("GetID").Return("test")
	mockServer.On("GetRegion").Return("test")
	net.On("AddServer", mockServer).Return(nil)
	require.NoError(t, net.AddServer(mockServer))
	ts := NewTimestampService(mockServer, net, "test")

	// Create test timestamp
	testTimestamp := &types.SignedTimestamp{
		Time:      time.Now(),
		ServerID:  "test",
		RegionID:  "test",
		Signature: []byte("test"),
	}

	// Setup expectations
	net.On("GetServers").Return([]types.TimeServer{mockServer})
	mockServer.On("Verify", mock.Anything, mock.Anything).Return(nil)

	// Test VerifyTimestamp
	err := ts.VerifyTimestamp(context.Background(), testTimestamp)
	require.NoError(t, err)
}

func TestTimestampService_GetValidators(t *testing.T) {
	// Create mock server
	mockServer := newMockTimestampServer("test", "test", nil)
	net := newMockTimeNetwork()
	mockServer.On("GetID").Return("test").Once()
	mockServer.On("GetRegion").Return("test").Once()
	net.On("AddServer", mockServer).Return(nil).Once()
	net.On("GetServersByRegion", "test").Return([]types.TimeServer{mockServer}).Once()
	net.On("Start").Return(nil).Once()
	require.NoError(t, net.AddServer(mockServer))
	ts := NewTimestampService(mockServer, net, "test")

	// Setup expectations
	expectedValidators := []*types.ValidatorInfo{
		{
			ID:     "test",
			Region: "test",
			Status: "healthy",
		},
	}

	// Test GetValidators
	validators, err := ts.GetValidators()
	require.NoError(t, err)
	assert.Equal(t, expectedValidators, validators)
}

func TestTimestampService_GetStatus(t *testing.T) {
	// Create mock server
	mockServer := newMockTimestampServer("test", "test", nil)
	net := newMockTimeNetwork()
	mockServer.On("GetID").Return("test").Maybe()
	mockServer.On("GetRegion").Return("test").Maybe()
	net.On("AddServer", mockServer).Return(nil)
	require.NoError(t, net.AddServer(mockServer))
	ts := NewTimestampService(mockServer, net, "test")

	// Setup expectations
	expectedStatus := &types.NetworkStatus{
		Status:      "healthy",
		Region:      "test",
		PeerCount:   1,
		LatestRound: 1,
	}
	net.On("GetStatus").Return(expectedStatus, nil)

	// Test GetStatus
	status, err := ts.GetStatus()
	require.NoError(t, err)
	assert.Equal(t, expectedStatus, status)
}

func TestTimestampService_GetRegion(t *testing.T) {
	// Create mock server
	mockServer := newMockTimestampServer("test", "test", nil)
	net := newMockTimeNetwork()
	mockServer.On("GetID").Return("test").Maybe()
	mockServer.On("GetRegion").Return("test").Maybe()
	net.On("AddServer", mockServer).Return(nil)
	require.NoError(t, net.AddServer(mockServer))
	ts := NewTimestampService(mockServer, net, "test")

	// Test GetRegion
	assert.Equal(t, "test", ts.GetRegion())
}

func TestTimestampService_SetRegion(t *testing.T) {
	// Create mock server
	mockServer := newMockTimestampServer("test", "test", nil)
	net := newMockTimeNetwork()
	mockServer.On("GetID").Return("test").Maybe()
	mockServer.On("GetRegion").Return("test").Maybe()
	net.On("AddServer", mockServer).Return(nil)
	require.NoError(t, net.AddServer(mockServer))
	ts := NewTimestampService(mockServer, net, "test")

	// Test SetRegion
	ts.SetRegion("us-west")
	assert.Equal(t, "us-west", ts.GetRegion())
}

func TestTimestampService_MultiRegion(t *testing.T) {
	// Generate test key pairs
	pub1, priv1, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	pub2, priv2, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	// Create mock servers
	mockServer1 := newMockTimestampServer("server1", "us-east", priv1)
	mockServer2 := newMockTimestampServer("server2", "us-west", priv2)

	net := newMockTimeNetwork()
	mockServer1.On("GetID").Return("server1").Maybe()
	mockServer1.On("GetRegion").Return("us-east").Maybe()
	mockServer1.On("GetPublicKey").Return(pub1).Maybe()
	mockServer2.On("GetID").Return("server2").Maybe()
	mockServer2.On("GetRegion").Return("us-west").Maybe()
	mockServer2.On("GetPublicKey").Return(pub2).Maybe()
	net.On("AddServer", mockServer1).Return(nil)
	net.On("AddServer", mockServer2).Return(nil)
	require.NoError(t, net.AddServer(mockServer1))
	require.NoError(t, net.AddServer(mockServer2))

	ts1 := NewTimestampService(mockServer1, net, "us-east")
	ts2 := NewTimestampService(mockServer2, net, "us-west")

	// Setup expectations
	now := time.Now()

	// Create signed timestamps
	ts1Data := &types.SignedTimestamp{
		Time:      now,
		ServerID:  "server1",
		RegionID:  "us-east",
		Signature: nil, // Will be set after signing
	}
	ts2Data := &types.SignedTimestamp{
		Time:      now,
		ServerID:  "server2",
		RegionID:  "us-west",
		Signature: nil, // Will be set after signing
	}

	// Sign the timestamps
	ts1Bytes := types.TimestampToBytes(now)
	ts1Data.Signature = ed25519.Sign(priv1, ts1Bytes)

	ts2Bytes := types.TimestampToBytes(now)
	ts2Data.Signature = ed25519.Sign(priv2, ts2Bytes)

	// Setup mock expectations
	mockServer1.On("GetTimestamp", mock.Anything).Return(ts1Data, nil)
	mockServer2.On("GetTimestamp", mock.Anything).Return(ts2Data, nil)
	mockServer1.On("Verify", mock.Anything, mock.Anything).Return(nil)
	mockServer2.On("Verify", mock.Anything, mock.Anything).Return(nil)

	// Setup mock expectations for GetServers
	net.On("GetServers").Return([]types.TimeServer{mockServer1, mockServer2}).Maybe()

	// Test GetTimestamp
	ts1Result, err := ts1.GetTimestamp(context.Background())
	require.NoError(t, err)
	assert.Equal(t, ts1Data, ts1Result)

	ts2Result, err := ts2.GetTimestamp(context.Background())
	require.NoError(t, err)
	assert.Equal(t, ts2Data, ts2Result)

	// Test cross-region verification
	err = ts1.VerifyTimestamp(context.Background(), ts2Data)
	require.NoError(t, err)

	err = ts2.VerifyTimestamp(context.Background(), ts1Data)
	require.NoError(t, err)
}
