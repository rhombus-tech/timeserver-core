package service

import (
	"context"

	"github.com/rhombus-tech/timeserver-core/core/types"
)

// FastPathHandler handles fast path timestamp operations
type FastPathHandler interface {
	GetTimestamp(ctx context.Context) (*types.SignedTimestamp, error)
}

// TimestampService handles timestamp operations
type TimestampService struct {
	fastPath FastPathHandler
	network  types.TimeNetwork
	region   string
}

// NewTimestampService creates a new TimestampService
func NewTimestampService(fastPath FastPathHandler, network types.TimeNetwork, region string) *TimestampService {
	return &TimestampService{
		fastPath: fastPath,
		network:  network,
		region:   region,
	}
}

// GetTimestamp gets a timestamp from the network
func (s *TimestampService) GetTimestamp(ctx context.Context) (*types.SignedTimestamp, error) {
	return s.fastPath.GetTimestamp(ctx)
}

// VerifyTimestamp verifies a timestamp's signature
func (s *TimestampService) VerifyTimestamp(ctx context.Context, ts *types.SignedTimestamp) error {
	servers := s.network.GetServers()
	for _, server := range servers {
		if server.GetID() == ts.ServerID {
			return ts.Verify(server)
		}
	}
	return types.ErrServerNotFound
}

// GetValidators gets the list of validators in the network
func (s *TimestampService) GetValidators() ([]*types.ValidatorInfo, error) {
	servers := s.network.GetServersByRegion(s.region)
	validators := make([]*types.ValidatorInfo, len(servers))
	
	for i, server := range servers {
		validators[i] = &types.ValidatorInfo{
			ID:     server.GetID(),
			Region: server.GetRegion(),
			Status: "healthy",
		}
	}
	
	return validators, nil
}

// GetStatus gets the status of the timestamp service
func (s *TimestampService) GetStatus() (*types.NetworkStatus, error) {
	return s.network.GetStatus()
}

// GetRegion returns the region of this service
func (s *TimestampService) GetRegion() string {
	return s.region
}

// SetRegion sets the region for this service
func (s *TimestampService) SetRegion(region string) {
	s.region = region
}
