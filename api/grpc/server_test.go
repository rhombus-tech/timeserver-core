package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/rhombus-tech/timeserver-core/api/rest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MockTimestampService mocks the TimestampService interface
type MockTimestampService struct {
	mock.Mock
}

func (m *MockTimestampService) GetTimestamp() (int64, []rest.SignatureInfo, string, error) {
	args := m.Called()
	if args.Get(3) != nil {
		return 0, nil, "", args.Error(3)
	}
	return args.Get(0).(int64), args.Get(1).([]rest.SignatureInfo), args.String(2), nil
}

func (m *MockTimestampService) VerifyTimestamp(timestamp int64, signatures []rest.SignatureInfo) (bool, string, error) {
	args := m.Called(timestamp, signatures)
	return args.Bool(0), args.String(1), args.Error(2)
}

func (m *MockTimestampService) GetValidators() ([]rest.ValidatorInfo, error) {
	args := m.Called()
	if args.Get(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]rest.ValidatorInfo), nil
}

func (m *MockTimestampService) GetStatus() (*rest.StatusInfo, error) {
	args := m.Called()
	if args.Get(1) != nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*rest.StatusInfo), nil
}

func TestGetTimestamp(t *testing.T) {
	mockService := new(MockTimestampService)
	server := NewServer(mockService)

	t.Run("successful get timestamp", func(t *testing.T) {
		now := time.Now().UnixNano()
		signatures := []rest.SignatureInfo{
			{ValidatorID: "server1", Signature: []byte("sig1")},
		}
		region := "us-east"

		mockService.On("GetTimestamp").Return(now, signatures, region, nil).Once()

		resp, err := server.GetTimestamp(context.Background(), &GetTimestampRequest{})
		assert.NoError(t, err)
		assert.Equal(t, now, resp.Timestamp)
		assert.Equal(t, region, resp.Region)
		assert.Len(t, resp.Signatures, 1)
		assert.Equal(t, "server1", resp.Signatures[0].ValidatorId)
		assert.Equal(t, []byte("sig1"), resp.Signatures[0].Signature)
	})

	t.Run("service error", func(t *testing.T) {
		mockService.On("GetTimestamp").Return(int64(0), nil, "", assert.AnError).Once()

		resp, err := server.GetTimestamp(context.Background(), &GetTimestampRequest{})
		assert.Nil(t, resp)
		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})
}

func TestVerifyTimestamp(t *testing.T) {
	mockService := new(MockTimestampService)
	server := NewServer(mockService)

	t.Run("successful verification", func(t *testing.T) {
		timestamp := time.Now().UnixNano()
		signatures := []rest.SignatureInfo{
			{ValidatorID: "server1", Signature: []byte("sig1")},
		}

		mockService.On("VerifyTimestamp", timestamp, signatures).Return(true, "", nil).Once()

		resp, err := server.VerifyTimestamp(context.Background(), &VerifyTimestampRequest{
			Timestamp: timestamp,
			Signatures: []*SignatureInfo{
				{ValidatorId: "server1", Signature: []byte("sig1")},
			},
		})
		assert.NoError(t, err)
		assert.True(t, resp.Valid)
		assert.Empty(t, resp.Error)
	})

	t.Run("invalid signature", func(t *testing.T) {
		timestamp := time.Now().UnixNano()
		signatures := []rest.SignatureInfo{
			{ValidatorID: "server1", Signature: []byte("invalid")},
		}

		mockService.On("VerifyTimestamp", timestamp, signatures).Return(false, "invalid signature", nil).Once()

		resp, err := server.VerifyTimestamp(context.Background(), &VerifyTimestampRequest{
			Timestamp: timestamp,
			Signatures: []*SignatureInfo{
				{ValidatorId: "server1", Signature: []byte("invalid")},
			},
		})
		assert.NoError(t, err)
		assert.False(t, resp.Valid)
		assert.Equal(t, "invalid signature", resp.Error)
	})
}

func TestGetValidators(t *testing.T) {
	mockService := new(MockTimestampService)
	server := NewServer(mockService)

	t.Run("successful get validators", func(t *testing.T) {
		validators := []rest.ValidatorInfo{
			{ID: "server1", Region: "us-east", Status: "active"},
		}

		mockService.On("GetValidators").Return(validators, nil).Once()

		resp, err := server.GetValidators(context.Background(), &GetValidatorsRequest{})
		assert.NoError(t, err)
		assert.Len(t, resp.Validators, 1)
		assert.Equal(t, "server1", resp.Validators[0].Id)
		assert.Equal(t, "us-east", resp.Validators[0].Region)
		assert.Equal(t, "active", resp.Validators[0].Status)
	})

	t.Run("service error", func(t *testing.T) {
		mockService.On("GetValidators").Return(nil, assert.AnError).Once()

		resp, err := server.GetValidators(context.Background(), &GetValidatorsRequest{})
		assert.Nil(t, resp)
		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})
}

func TestGetStatus(t *testing.T) {
	mockService := new(MockTimestampService)
	server := NewServer(mockService)

	t.Run("successful get status", func(t *testing.T) {
		status := &rest.StatusInfo{
			Status:      "healthy",
			Region:      "us-east",
			PeerCount:   5,
			LatestRound: 100,
		}

		mockService.On("GetStatus").Return(status, nil).Once()

		resp, err := server.GetStatus(context.Background(), &GetStatusRequest{})
		assert.NoError(t, err)
		assert.Equal(t, "healthy", resp.Status)
		assert.Equal(t, "us-east", resp.Region)
		assert.Equal(t, int32(5), resp.PeerCount)
		assert.Equal(t, int64(100), resp.LatestRound)
	})

	t.Run("service error", func(t *testing.T) {
		mockService.On("GetStatus").Return(nil, assert.AnError).Once()

		resp, err := server.GetStatus(context.Background(), &GetStatusRequest{})
		assert.Nil(t, resp)
		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
	})
}
