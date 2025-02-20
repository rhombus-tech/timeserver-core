package service

import (
	"context"
	"crypto/ed25519"
	"errors"
	"net"
	"testing"
	"time"

	pb "github.com/rhombus-tech/timeserver-core/api/grpc"
	"github.com/rhombus-tech/timeserver-core/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

var errInvalidSignature = errors.New("invalid signature")

// mockTimestampServer implements both TimestampServiceInterface and types.TimeServer
type mockTimestampServer struct {
	mock.Mock
	id      string
	region  string
	privKey ed25519.PrivateKey
	pubKey  ed25519.PublicKey
}

// serviceWrapper wraps mockTimestampServer to implement TimestampServiceInterface
type serviceWrapper struct {
	*mockTimestampServer
}

func (s *serviceWrapper) SetRegion(region string) {
	_ = s.mockTimestampServer.SetRegion(region)
}

func newMockTimestampServer(id, region string, privKey ed25519.PrivateKey) *mockTimestampServer {
	var pubKey ed25519.PublicKey
	if privKey != nil {
		pubKey = privKey.Public().(ed25519.PublicKey)
	}
	s := &mockTimestampServer{
		id:      id,
		region:  region,
		privKey: privKey,
		pubKey:  pubKey,
	}
	return s
}

func (s *mockTimestampServer) GetID() string {
	args := s.Called()
	if args.Get(0) != nil {
		return args.Get(0).(string)
	}
	return s.id
}

func (s *mockTimestampServer) GetRegion() string {
	args := s.Called()
	if args.Get(0) != nil {
		return args.Get(0).(string)
	}
	return s.region
}

// SetRegion implements types.TimeServer
func (s *mockTimestampServer) SetRegion(region string) error {
	args := s.Called(region)
	if args.Error(0) != nil {
		return args.Error(0)
	}
	s.region = region
	return nil
}

func (s *mockTimestampServer) GetPublicKey() ed25519.PublicKey {
	args := s.Called()
	if args.Get(0) != nil {
		return args.Get(0).(ed25519.PublicKey)
	}
	return s.pubKey
}

func (s *mockTimestampServer) Sign(message []byte) ([]byte, error) {
	args := s.Called(message)
	if args.Get(0) != nil {
		return args.Get(0).([]byte), args.Error(1)
	}
	if s.privKey == nil {
		return []byte("test-signature"), nil
	}
	return ed25519.Sign(s.privKey, message), nil
}

func (s *mockTimestampServer) Verify(message, signature []byte) error {
	args := s.Called(message, signature)
	if args.Error(0) != nil {
		return args.Error(0)
	}
	if s.privKey == nil {
		return nil
	}
	if !ed25519.Verify(s.pubKey, message, signature) {
		return types.ErrInvalidSignature
	}
	return nil
}

func (s *mockTimestampServer) GetTimestamp(ctx context.Context) (*types.SignedTimestamp, error) {
	args := s.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.SignedTimestamp), args.Error(1)
}

func (s *mockTimestampServer) VerifyTimestamp(ctx context.Context, ts *types.SignedTimestamp) error {
	args := s.Called(ctx, ts)
	return args.Error(0)
}

func (s *mockTimestampServer) GetValidators() ([]*types.ValidatorInfo, error) {
	args := s.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*types.ValidatorInfo), args.Error(1)
}

func (s *mockTimestampServer) GetStatus() (*types.NetworkStatus, error) {
	args := s.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.NetworkStatus), args.Error(1)
}

func TestGRPCAdapter_GetTimestamp(t *testing.T) {
	svc := newMockTimestampServer("test-server", "test-region", nil)
	adapter := NewGRPCAdapter(&serviceWrapper{svc})

	// Test successful case
	now := time.Now()
	expectedTS := &types.SignedTimestamp{
		Time:      now,
		ServerID:  "test-server",
		RegionID:  "test-region",
		Signature: []byte("test-signature"),
	}

	svc.On("GetTimestamp", mock.Anything).Return(expectedTS, nil)

	resp, err := adapter.GetTimestamp(context.Background(), &pb.GetTimestampRequest{})
	require.NoError(t, err)
	assert.Equal(t, now.UnixNano(), resp.Timestamp)
	assert.Equal(t, expectedTS.ServerID, resp.Signatures[0].ValidatorId)
	assert.Equal(t, expectedTS.RegionID, resp.Signatures[0].Region)
	assert.Equal(t, expectedTS.Signature, resp.Signatures[0].Signature)

	svc.AssertExpectations(t)
}

func TestGRPCAdapter_VerifyTimestamp(t *testing.T) {
	svc := newMockTimestampServer("test-server", "test-region", nil)
	adapter := NewGRPCAdapter(&serviceWrapper{svc})

	// Test successful case
	now := time.Now()
	req := &pb.VerifyTimestampRequest{
		Timestamp: now.UnixNano(),
		Signatures: []*pb.SignatureInfo{{
			ValidatorId: "test-server",
			Region:      "test-region",
			Signature:   []byte("test-signature"),
		}},
	}

	svc.On("VerifyTimestamp", mock.Anything, mock.AnythingOfType("*types.SignedTimestamp")).Return(nil)

	resp, err := adapter.VerifyTimestamp(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Valid)

	svc.AssertExpectations(t)
}

func TestGRPCAdapter_GetStatus(t *testing.T) {
	svc := newMockTimestampServer("test-server", "test-region", nil)
	adapter := NewGRPCAdapter(&serviceWrapper{svc})

	// Test successful case
	expectedStatus := &types.NetworkStatus{
		Status:      "healthy",
		Region:      "test-region",
		PeerCount:   0,
		LatestRound: 0,
	}

	svc.On("GetStatus").Return(expectedStatus, nil)

	resp, err := adapter.GetStatus(context.Background(), &pb.GetStatusRequest{})
	require.NoError(t, err)
	assert.Equal(t, expectedStatus.Status, resp.Status)
	assert.Equal(t, expectedStatus.Region, resp.Region)
	assert.Equal(t, int32(expectedStatus.PeerCount), resp.PeerCount)
	assert.Equal(t, int64(expectedStatus.LatestRound), resp.LatestRound)

	svc.AssertExpectations(t)
}

func TestGRPCAdapterFull(t *testing.T) {
	// Generate test key pair
	_, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	// Create mock implementations
	svc := newMockTimestampServer("test", "test", priv)
	adapter := NewGRPCAdapter(&serviceWrapper{svc})

	// Setup mock expectations
	now := time.Now()
	tsBytes := types.TimestampToBytes(now)
	sig := ed25519.Sign(priv, tsBytes)
	expectedTS := &types.SignedTimestamp{
		Time:      now,
		ServerID:  "test",
		RegionID:  "test",
		Signature: sig,
	}

	svc.On("GetTimestamp", mock.Anything).Return(expectedTS, nil)
	svc.On("GetStatus").Return(&types.NetworkStatus{
		Status:      "healthy",
		Region:      "test",
		PeerCount:   0,
		LatestRound: 0,
	}, nil)
	svc.On("GetValidators").Return([]*types.ValidatorInfo{
		{
			ID:     "test",
			Region: "test",
			Status: "active",
		},
	}, nil)

	// Start gRPC server
	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	pb.RegisterTimestampServiceServer(grpcServer, adapter)

	serverErrCh := make(chan error, 1)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			serverErrCh <- err
		}
	}()
	defer grpcServer.Stop()

	// Create gRPC client with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, "", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}))
	require.NoError(t, err)
	defer conn.Close()

	client := pb.NewTimestampServiceClient(conn)

	// Test GetTimestamp
	t.Run("GetTimestamp", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		resp, err := client.GetTimestamp(ctx, &pb.GetTimestampRequest{})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.Signatures)
		assert.Equal(t, now.UnixNano(), resp.Timestamp)
		assert.Equal(t, "test", resp.Signatures[0].ValidatorId)
		assert.Equal(t, "test", resp.Signatures[0].Region)
		assert.Equal(t, sig, resp.Signatures[0].Signature)
	})

	// Test VerifyTimestamp
	t.Run("VerifyTimestamp", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		// First get a valid timestamp
		getResp, err := client.GetTimestamp(ctx, &pb.GetTimestampRequest{})
		require.NoError(t, err)

		// Set up mock for valid signature
		svc.On("VerifyTimestamp", mock.Anything, &types.SignedTimestamp{
			Time:      time.Unix(0, getResp.Timestamp),
			ServerID:  "test",
			RegionID:  "test",
			Signature: getResp.Signatures[0].Signature,
		}).Return(nil)

		// Verify the valid timestamp
		verifyResp, err := client.VerifyTimestamp(ctx, &pb.VerifyTimestampRequest{
			Timestamp:  getResp.Timestamp,
			Signatures: getResp.Signatures,
		})
		require.NoError(t, err)
		assert.True(t, verifyResp.Valid)

		// Test invalid signature
		invalidSig := make([]byte, len(sig))
		copy(invalidSig, sig)
		invalidSig[0] ^= 0xFF // Flip some bits

		// Set up mock for invalid signature
		svc.On("VerifyTimestamp", mock.Anything, &types.SignedTimestamp{
			Time:      time.Unix(0, getResp.Timestamp),
			ServerID:  "test",
			RegionID:  "test",
			Signature: invalidSig,
		}).Return(types.ErrInvalidSignature)

		// Verify with invalid signature should fail
		verifyResp, err = client.VerifyTimestamp(ctx, &pb.VerifyTimestampRequest{
			Timestamp: getResp.Timestamp,
			Signatures: []*pb.SignatureInfo{{
				ValidatorId: "test",
				Region:     "test",
				Signature:  invalidSig,
			}},
		})
		require.NoError(t, err)
		assert.False(t, verifyResp.Valid)
	})

	// Test GetValidators
	t.Run("GetValidators", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		resp, err := client.GetValidators(ctx, &pb.GetValidatorsRequest{})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotEmpty(t, resp.Validators)
		for _, v := range resp.Validators {
			assert.NotEmpty(t, v.Id)
			assert.NotEmpty(t, v.Region)
		}
	})

	// Test GetStatus
	t.Run("GetStatus", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		resp, err := client.GetStatus(ctx, &pb.GetStatusRequest{})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "test", resp.Region)
		assert.Equal(t, "healthy", resp.Status)
		assert.Equal(t, int32(0), resp.PeerCount)
		assert.Equal(t, int64(0), resp.LatestRound)
	})

	select {
	case err := <-serverErrCh:
		t.Fatalf("gRPC server error: %v", err)
	default:
	}

	svc.AssertExpectations(t)
}
