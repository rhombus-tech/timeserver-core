package network

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/rhombus-tech/timeserver-core/core/p2p"
	"github.com/rhombus-tech/timeserver-core/core/types"
)

// NetworkAdapter adapts a P2P network to the TimeNetwork interface
type NetworkAdapter struct {
	network p2p.Network
	region  string
	servers map[string]types.TimeServer
	mu      sync.RWMutex
}

// NewNetworkAdapter creates a new NetworkAdapter
func NewNetworkAdapter(network p2p.Network, region string) *NetworkAdapter {
	return &NetworkAdapter{
		network: network,
		region:  region,
		servers: make(map[string]types.TimeServer),
		mu:      sync.RWMutex{},
	}
}

// AddServer adds a server to the network
func (a *NetworkAdapter) AddServer(server types.TimeServer) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	a.servers[server.GetID()] = server
	return nil
}

// RemoveServer removes a server from the network
func (a *NetworkAdapter) RemoveServer(serverID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	delete(a.servers, serverID)
	return nil
}

// GetServers returns all servers in the network
func (a *NetworkAdapter) GetServers() []types.TimeServer {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	result := make([]types.TimeServer, 0, len(a.servers))
	for _, server := range a.servers {
		result = append(result, server)
	}
	return result
}

// GetServersByRegion returns all servers in a specific region
func (a *NetworkAdapter) GetServersByRegion(region string) []types.TimeServer {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	var result []types.TimeServer
	for _, server := range a.servers {
		if server.GetRegion() == region {
			result = append(result, server)
		}
	}
	return result
}

// GetStatus returns the current network status
func (a *NetworkAdapter) GetStatus() (*types.NetworkStatus, error) {
	return &types.NetworkStatus{
		Status:      "healthy",
		Region:      a.region,
		PeerCount:   len(a.GetServers()),
		LatestRound: 0,
	}, nil
}

// Start starts the network adapter
func (a *NetworkAdapter) Start(ctx context.Context) error {
	return a.network.Start(ctx)
}

// Stop stops the network adapter
func (a *NetworkAdapter) Stop() error {
	return a.network.Stop()
}

// Broadcast broadcasts a message to the network
func (a *NetworkAdapter) Broadcast(ctx context.Context, protocolID protocol.ID, data []byte) error {
	return a.network.Broadcast(ctx, protocolID, data)
}

// Send sends a message to a specific peer
func (a *NetworkAdapter) Send(ctx context.Context, peerID peer.ID, protocolID protocol.ID, data []byte) error {
	return a.network.Send(ctx, peerID, protocolID, data)
}

// RegisterProtocol registers a protocol handler
func (a *NetworkAdapter) RegisterProtocol(protocolID protocol.ID, handler func(context.Context, peer.ID, []byte) error) {
	a.network.RegisterProtocol(protocolID, handler)
}

// GetTimestamp gets a timestamp from a server
func (a *NetworkAdapter) GetTimestamp(server types.TimeServer) (*types.SignedTimestamp, error) {
	now := time.Now()
	message := types.TimestampToBytes(now)
	signature, err := server.Sign(message)
	if err != nil {
		return nil, fmt.Errorf("failed to sign timestamp: %w", err)
	}

	return &types.SignedTimestamp{
		Time:      now,
		ServerID:  server.GetID(),
		RegionID:  server.GetRegion(),
		Signature: signature,
	}, nil
}

// VerifyTimestamp verifies a timestamp from a server
func (a *NetworkAdapter) VerifyTimestamp(server types.TimeServer, ts *types.SignedTimestamp) error {
	return ts.Verify(server)
}
