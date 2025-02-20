package server

import (
	"context"
	"crypto/ed25519"
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rhombus-tech/timeserver-core/core/consensus"
	"github.com/rhombus-tech/timeserver-core/core/network"
	"github.com/rhombus-tech/timeserver-core/core/p2p"
	"github.com/rhombus-tech/timeserver-core/core/types"
)

// Server represents a timeserver instance
type Server struct {
	id       string
	region   string
	privKey  ed25519.PrivateKey
	pubKey   ed25519.PublicKey
	network  *network.NetworkAdapter
	fastPath *FastPath
}

// NewServer creates a new Server instance
func NewServer(id string, region string, privKey ed25519.PrivateKey, bootstrapPeers []string) (*Server, error) {
	pubKey := privKey.Public().(ed25519.PublicKey)

	// Convert ed25519 private key to libp2p private key
	libp2pPrivKey, _, err := crypto.KeyPairFromStdKey(&privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert private key: %w", err)
	}

	// Create P2P network
	p2pNetwork, err := p2p.NewP2PNetwork(id, libp2pPrivKey, bootstrapPeers)
	if err != nil {
		return nil, fmt.Errorf("failed to create P2P network: %w", err)
	}

	// Create network adapter
	networkAdapter := network.NewNetworkAdapter(p2pNetwork, region)

	// Create consensus state
	validators := make(map[string]bool)
	for _, peer := range bootstrapPeers {
		validators[peer] = true
	}
	state := consensus.NewState(id, validators)

	// Create consensus
	cons := consensus.NewConsensus(state, networkAdapter)
	_ = cons // TODO: Use consensus when implementing consensus-related features

	// Create server
	server := &Server{
		id:      id,
		region:  region,
		privKey: privKey,
		pubKey:  pubKey,
		network: networkAdapter,
	}

	// Create fast path handler
	server.fastPath = NewFastPath(server)

	return server, nil
}

// GetID returns the server ID
func (s *Server) GetID() string {
	return s.id
}

// GetRegion returns the server region
func (s *Server) GetRegion() string {
	return s.region
}

// GetPublicKey returns the server's public key
func (s *Server) GetPublicKey() ed25519.PublicKey {
	return s.pubKey
}

// Sign signs a message with the server's private key
func (s *Server) Sign(message []byte) ([]byte, error) {
	return ed25519.Sign(s.privKey, message), nil
}

// Verify verifies a signature with the server's public key
func (s *Server) Verify(message, signature []byte) error {
	if !ed25519.Verify(s.pubKey, message, signature) {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	return s.network.Start(ctx)
}

// Stop stops the server
func (s *Server) Stop() error {
	return s.network.Stop()
}

// GetFastPath returns the server's fast path handler
func (s *Server) GetFastPath() *FastPath {
	return s.fastPath
}

// AddPeer adds a peer to the network
func (s *Server) AddPeer(peer types.TimeServer) error {
	return s.network.AddServer(peer)
}

// GetPeers returns all peers in the network
func (s *Server) GetPeers() []types.TimeServer {
	return s.network.GetServers()
}

// SetRegion sets the server's region
func (s *Server) SetRegion(region string) error {
	s.region = region
	return nil
}
