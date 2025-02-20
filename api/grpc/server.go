package grpc

import (
	"context"

	"github.com/rhombus-tech/timeserver-core/api/rest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements the gRPC TimestampService
type Server struct {
	UnimplementedTimestampServiceServer
	timestampService TimestampService
}

// TimestampService defines the interface for timestamp operations
type TimestampService interface {
	GetTimestamp() (int64, []rest.SignatureInfo, string, error)
	VerifyTimestamp(timestamp int64, signatures []rest.SignatureInfo) (bool, string, error)
	GetValidators() ([]rest.ValidatorInfo, error)
	GetStatus() (*rest.StatusInfo, error)
}

// NewServer creates a new gRPC server
func NewServer(ts TimestampService) *Server {
	return &Server{
		timestampService: ts,
	}
}

// GetTimestamp implements the GetTimestamp RPC
func (s *Server) GetTimestamp(ctx context.Context, req *GetTimestampRequest) (*GetTimestampResponse, error) {
	timestamp, signatures, region, err := s.timestampService.GetTimestamp()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Convert signatures to proto format
	protoSigs := make([]*SignatureInfo, len(signatures))
	for i, sig := range signatures {
		protoSigs[i] = &SignatureInfo{
			ValidatorId: sig.ValidatorID,
			Signature:   sig.Signature,
		}
	}

	return &GetTimestampResponse{
		Timestamp:  timestamp,
		Signatures: protoSigs,
		Region:     region,
	}, nil
}

// VerifyTimestamp implements the VerifyTimestamp RPC
func (s *Server) VerifyTimestamp(ctx context.Context, req *VerifyTimestampRequest) (*VerifyTimestampResponse, error) {
	// Convert proto signatures to service format
	signatures := make([]rest.SignatureInfo, len(req.Signatures))
	for i, sig := range req.Signatures {
		signatures[i] = rest.SignatureInfo{
			ValidatorID: sig.ValidatorId,
			Signature:   sig.Signature,
		}
	}

	valid, errMsg, err := s.timestampService.VerifyTimestamp(req.Timestamp, signatures)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &VerifyTimestampResponse{
		Valid: valid,
		Error: errMsg,
	}, nil
}

// GetValidators implements the GetValidators RPC
func (s *Server) GetValidators(ctx context.Context, req *GetValidatorsRequest) (*GetValidatorsResponse, error) {
	validators, err := s.timestampService.GetValidators()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Convert validators to proto format
	protoValidators := make([]*ValidatorInfo, len(validators))
	for i, v := range validators {
		protoValidators[i] = &ValidatorInfo{
			Id:     v.ID,
			Region: v.Region,
			Status: v.Status,
		}
	}

	return &GetValidatorsResponse{
		Validators: protoValidators,
	}, nil
}

// GetStatus implements the GetStatus RPC
func (s *Server) GetStatus(ctx context.Context, req *GetStatusRequest) (*GetStatusResponse, error) {
	nodeStatus, err := s.timestampService.GetStatus()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &GetStatusResponse{
		Status:      nodeStatus.Status,
		Region:      nodeStatus.Region,
		PeerCount:   nodeStatus.PeerCount,
		LatestRound: nodeStatus.LatestRound,
	}, nil
}

// RegisterServer registers the gRPC server on the provided grpc.Server
func RegisterServer(s *grpc.Server, ts TimestampService) {
	RegisterTimestampServiceServer(s, NewServer(ts))
}
