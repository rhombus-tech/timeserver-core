package service

import (
	"context"
	"fmt"
	"time"

	pb "github.com/rhombus-tech/timeserver-core/api/grpc"
	"github.com/rhombus-tech/timeserver-core/core/types"
)

// TimestampServiceInterface defines the interface that GRPCAdapter expects
type TimestampServiceInterface interface {
	GetTimestamp(ctx context.Context) (*types.SignedTimestamp, error)
	VerifyTimestamp(ctx context.Context, ts *types.SignedTimestamp) error
	GetValidators() ([]*types.ValidatorInfo, error)
	GetStatus() (*types.NetworkStatus, error)
	GetRegion() string
	SetRegion(region string)
}

// GRPCAdapter adapts the TimestampService to gRPC
type GRPCAdapter struct {
	pb.UnimplementedTimestampServiceServer
	service TimestampServiceInterface
}

// NewGRPCAdapter creates a new GRPCAdapter
func NewGRPCAdapter(service TimestampServiceInterface) *GRPCAdapter {
	return &GRPCAdapter{
		service: service,
	}
}

// GetTimestamp gets a timestamp
func (a *GRPCAdapter) GetTimestamp(ctx context.Context, req *pb.GetTimestampRequest) (*pb.GetTimestampResponse, error) {
	ts, err := a.service.GetTimestamp(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get timestamp: %w", err)
	}

	return &pb.GetTimestampResponse{
		Timestamp: ts.Time.UnixNano(),
		Signatures: []*pb.SignatureInfo{{
			ValidatorId: ts.ServerID,
			Signature:   ts.Signature,
			Region:      ts.RegionID,
		}},
		Region: ts.RegionID,
	}, nil
}

// VerifyTimestamp verifies a timestamp
func (a *GRPCAdapter) VerifyTimestamp(ctx context.Context, req *pb.VerifyTimestampRequest) (*pb.VerifyTimestampResponse, error) {
	// Convert to our internal type
	ts := &types.SignedTimestamp{
		Time:      time.Unix(0, req.Timestamp),
		ServerID:  req.Signatures[0].ValidatorId,
		RegionID:  req.Signatures[0].Region,
		Signature: req.Signatures[0].Signature,
	}

	err := a.service.VerifyTimestamp(ctx, ts)
	if err == types.ErrInvalidSignature {
		return &pb.VerifyTimestampResponse{
			Valid: false,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to verify timestamp: %w", err)
	}
	return &pb.VerifyTimestampResponse{
		Valid: true,
	}, nil
}

// GetValidators gets the list of validators
func (a *GRPCAdapter) GetValidators(ctx context.Context, req *pb.GetValidatorsRequest) (*pb.GetValidatorsResponse, error) {
	validators, err := a.service.GetValidators()
	if err != nil {
		return nil, fmt.Errorf("failed to get validators: %w", err)
	}

	pbValidators := make([]*pb.ValidatorInfo, len(validators))
	for i, v := range validators {
		pbValidators[i] = &pb.ValidatorInfo{
			Id:     v.ID,
			Region: v.Region,
			Status: v.Status,
		}
	}

	return &pb.GetValidatorsResponse{
		Validators: pbValidators,
	}, nil
}

// GetStatus gets the network status
func (a *GRPCAdapter) GetStatus(ctx context.Context, req *pb.GetStatusRequest) (*pb.GetStatusResponse, error) {
	status, err := a.service.GetStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	return &pb.GetStatusResponse{
		Status:      status.Status,
		Region:      status.Region,
		PeerCount:   int32(status.PeerCount),
		LatestRound: int64(status.LatestRound),
	}, nil
}

// MustEmbedUnimplementedTimestampServiceServer implements the gRPC interface
func (a *GRPCAdapter) MustEmbedUnimplementedTimestampServiceServer() {
	// This is required by the gRPC interface
}
