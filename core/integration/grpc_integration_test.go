package integration

import (
	"context"
	"crypto/ed25519"
	"net"
	"testing"

	pb "github.com/rhombus-tech/timeserver-core/api/grpc"
	"github.com/rhombus-tech/timeserver-core/core/server"
	"github.com/rhombus-tech/timeserver-core/core/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

type testServer struct {
	pb.UnimplementedTimestampServiceServer
	server *server.Server
	lis    *bufconn.Listener
	srv    *grpc.Server
}

func newTestServer(t *testing.T) *testServer {
	// Generate server key
	_, privKey, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)

	// Create server
	srv, err := server.NewServer("test-server", "test-region", privKey, nil)
	require.NoError(t, err)

	// Create gRPC server
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	adapter := service.NewGRPCAdapter(srv.GetFastPath())
	pb.RegisterTimestampServiceServer(s, adapter)

	ts := &testServer{
		server: srv,
		lis:    lis,
		srv:    s,
	}

	// Start gRPC server
	go func() {
		if err := s.Serve(lis); err != nil {
			t.Errorf("Failed to serve: %v", err)
		}
	}()

	return ts
}

func (s *testServer) stop() {
	s.srv.Stop()
	s.lis.Close()
	s.server.Stop()
}

func bufDialer(lis *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, s string) (net.Conn, error) {
		return lis.Dial()
	}
}

func TestGRPCIntegration(t *testing.T) {
	ctx := context.Background()

	// Start test server
	ts := newTestServer(t)
	defer ts.stop()

	// Create gRPC client
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(bufDialer(ts.lis)),
		grpc.WithInsecure())
	require.NoError(t, err)
	defer conn.Close()

	client := pb.NewTimestampServiceClient(conn)

	t.Run("GetTimestamp", func(t *testing.T) {
		resp, err := client.GetTimestamp(ctx, &pb.GetTimestampRequest{})
		require.NoError(t, err)
		assert.NotZero(t, resp.Timestamp)
		assert.Equal(t, "test-region", resp.Region)
		assert.Len(t, resp.Signatures, 1)
		assert.Equal(t, "test-server", resp.Signatures[0].ValidatorId)
	})

	t.Run("VerifyTimestamp", func(t *testing.T) {
		// First get a timestamp
		tsResp, err := client.GetTimestamp(ctx, &pb.GetTimestampRequest{})
		require.NoError(t, err)

		// Then verify it
		verifyResp, err := client.VerifyTimestamp(ctx, &pb.VerifyTimestampRequest{
			Timestamp:  tsResp.Timestamp,
			Signatures: tsResp.Signatures,
		})
		require.NoError(t, err)
		assert.True(t, verifyResp.Valid)
	})

	t.Run("GetStatus", func(t *testing.T) {
		resp, err := client.GetStatus(ctx, &pb.GetStatusRequest{})
		require.NoError(t, err)
		assert.Equal(t, "test-region", resp.Region)
	})
}
