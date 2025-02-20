package network

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/rhombus-tech/timeserver-core/core/types"
)

// MockTimeNetwork implements the TimeNetwork interface for testing
type MockTimeNetwork struct {
	sync.RWMutex
	servers map[string]types.TimeServer
	regions map[string][]types.TimeServer
}

func NewMockTimeNetwork() *MockTimeNetwork {
	return &MockTimeNetwork{
		servers: make(map[string]types.TimeServer),
		regions: make(map[string][]types.TimeServer),
	}
}

func (n *MockTimeNetwork) AddServer(server types.TimeServer) error {
	n.Lock()
	defer n.Unlock()

	id := server.GetID()
	region := server.GetRegion()
	
	// Check if server already exists
	if _, exists := n.servers[id]; exists {
		return types.ErrServerExists
	}
	
	n.servers[id] = server
	if n.regions[region] == nil {
		n.regions[region] = make([]types.TimeServer, 0)
	}
	n.regions[region] = append(n.regions[region], server)
	return nil
}

func (n *MockTimeNetwork) RemoveServer(serverID string) error {
	n.Lock()
	defer n.Unlock()

	if server, exists := n.servers[serverID]; exists {
		region := server.GetRegion()
		delete(n.servers, serverID)
		
		// Remove from regions
		servers := n.regions[region]
		for i, s := range servers {
			if s.GetID() == serverID {
				n.regions[region] = append(servers[:i], servers[i+1:]...)
				break
			}
		}
	}
	return nil
}

func (n *MockTimeNetwork) GetServers() []types.TimeServer {
	n.RLock()
	defer n.RUnlock()

	servers := make([]types.TimeServer, 0, len(n.servers))
	for _, server := range n.servers {
		servers = append(servers, server)
	}
	return servers
}

func (n *MockTimeNetwork) GetServersByRegion(region string) []types.TimeServer {
	n.RLock()
	defer n.RUnlock()

	if servers, exists := n.regions[region]; exists {
		result := make([]types.TimeServer, len(servers))
		copy(result, servers)
		return result
	}
	return nil
}

func (n *MockTimeNetwork) GetStatus() (*types.NetworkStatus, error) {
	n.RLock()
	defer n.RUnlock()

	return &types.NetworkStatus{
		Status:      "healthy",
		Region:      "mock-region",
		PeerCount:   len(n.servers),
		LatestRound: 1,
	}, nil
}

func (n *MockTimeNetwork) Start() error {
	return nil
}

func (n *MockTimeNetwork) Stop() error {
	return nil
}

// Additional methods for testing
func (n *MockTimeNetwork) Broadcast(_ context.Context, _ protocol.ID, _ []byte) error {
	return nil
}

func (n *MockTimeNetwork) Send(_ context.Context, _ peer.ID, _ protocol.ID, _ []byte) error {
	return nil
}

func (n *MockTimeNetwork) RegisterProtocol(_ protocol.ID, _ func(context.Context, peer.ID, []byte) error) {
	// Do nothing for mock
}

func (n *MockTimeNetwork) GetTimestamp(server types.TimeServer) (*types.SignedTimestamp, error) {
	ts := &types.SignedTimestamp{
		Time:     time.Now(),
		ServerID: server.GetID(),
		RegionID: server.GetRegion(),
	}
	
	data := types.TimestampToBytes(ts.Time)
	sig, err := server.Sign(data)
	if err != nil {
		return nil, err
	}
	ts.Signature = sig
	return ts, nil
}

func (n *MockTimeNetwork) VerifyTimestamp(server types.TimeServer, ts *types.SignedTimestamp) error {
	return ts.Verify(server)
}
