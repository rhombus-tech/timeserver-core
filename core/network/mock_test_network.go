package network

import (
	"crypto/ed25519"
	"errors"
	"sync"
	"time"

	"github.com/rhombus-tech/timeserver-core/core/types"
)

// MockNetwork implements types.TimeNetwork for testing
type MockNetwork struct {
	mu      sync.RWMutex
	servers map[string]*MockServer
	regions map[string][]*MockServer
}

// NewMockNetwork creates a new mock network for testing
func NewMockNetwork() *MockNetwork {
	return &MockNetwork{
		servers: make(map[string]*MockServer),
		regions: make(map[string][]*MockServer),
	}
}

// MockServer implements types.TimeServer for testing
type MockServer struct {
	ID       string
	Region   string
	PrivKey  ed25519.PrivateKey
	PubKey   ed25519.PublicKey
	Failures int
	RTT      time.Duration
}

// NewMockServer creates a new mock server
func NewMockServer(id, region string, privKey ed25519.PrivateKey, pubKey ed25519.PublicKey) *MockServer {
	return &MockServer{
		ID:      id,
		Region:  region,
		PrivKey: privKey,
		PubKey:  pubKey,
		RTT:     50 * time.Millisecond, // Default RTT value
	}
}

// GetID returns the server ID
func (s *MockServer) GetID() string {
	return s.ID
}

// GetRegion returns the server region
func (s *MockServer) GetRegion() string {
	return s.Region
}

// SetRegion sets the server region
func (s *MockServer) SetRegion(region string) error {
	s.Region = region
	return nil
}

// GetPublicKey returns the server's public key
func (s *MockServer) GetPublicKey() ed25519.PublicKey {
	return s.PubKey
}

// Sign signs a message using the server's private key
func (s *MockServer) Sign(message []byte) ([]byte, error) {
	return ed25519.Sign(s.PrivKey, message), nil
}

// Verify verifies a signature using the server's public key
func (s *MockServer) Verify(message []byte, signature []byte) error {
	if !ed25519.Verify(s.PubKey, message, signature) {
		return types.ErrInvalidSignature
	}
	return nil
}

// SetLatency sets the simulated network latency for the server
func (s *MockServer) SetLatency(latency time.Duration) {
	s.RTT = latency
}

// AddServer adds a server to the mock network
func (n *MockNetwork) AddServer(server types.TimeServer) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	// Check if server already exists
	if _, exists := n.servers[server.GetID()]; exists {
		return types.ErrServerExists
	}
	
	// Convert to MockServer if possible
	mockServer, ok := server.(*MockServer)
	if !ok {
		// If it's not already a MockServer, we'd need to create one
		// But for simplicity in tests, we'll just require MockServer instances
		return errors.New("only MockServer instances are supported")
	}
	
	n.servers[mockServer.ID] = mockServer
	
	// Add to region mapping
	region := mockServer.GetRegion()
	if n.regions[region] == nil {
		n.regions[region] = make([]*MockServer, 0)
	}
	n.regions[region] = append(n.regions[region], mockServer)
	
	return nil
}

// RemoveServer removes a server from the mock network
func (n *MockNetwork) RemoveServer(serverID string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	server, exists := n.servers[serverID]
	if !exists {
		return types.ErrServerNotFound
	}
	
	// Remove from servers map
	delete(n.servers, serverID)
	
	// Remove from region mapping
	region := server.Region
	if servers, ok := n.regions[region]; ok {
		for i, s := range servers {
			if s.ID == serverID {
				// Remove server from slice
				n.regions[region] = append(servers[:i], servers[i+1:]...)
				break
			}
		}
		
		// If region is empty, remove it
		if len(n.regions[region]) == 0 {
			delete(n.regions, region)
		}
	}
	
	return nil
}

// GetServers returns all servers in the network
func (n *MockNetwork) GetServers() []types.TimeServer {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	servers := make([]types.TimeServer, 0, len(n.servers))
	for _, server := range n.servers {
		servers = append(servers, server)
	}
	
	return servers
}

// GetServersByRegion returns servers in a specific region
func (n *MockNetwork) GetServersByRegion(region string) []types.TimeServer {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	if regionServers, ok := n.regions[region]; ok {
		servers := make([]types.TimeServer, len(regionServers))
		for i, server := range regionServers {
			servers[i] = server
		}
		return servers
	}
	
	return []types.TimeServer{}
}

// GetStatus returns the mock network status
func (n *MockNetwork) GetStatus() (*types.NetworkStatus, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	status := &types.NetworkStatus{
		Status:    "healthy",
		Region:    "test",
		PeerCount: len(n.servers),
	}
	
	return status, nil
}

// Start starts the mock network
func (n *MockNetwork) Start() error {
	return nil
}

// Stop stops the mock network
func (n *MockNetwork) Stop() error {
	return nil
}
