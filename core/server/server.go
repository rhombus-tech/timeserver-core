package server

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rhombus-tech/timeserver-core/core/consensus"
	"github.com/rhombus-tech/timeserver-core/core/p2p"
	"github.com/rhombus-tech/timeserver-core/core/types"
)

// Server represents a timeserver node
type Server struct {
	id        string
	privKey   ed25519.PrivateKey
	pubKey    ed25519.PublicKey
	network   *p2p.P2PNetwork
	consensus *consensus.Consensus
	fastPath  *FastPath
	mu        sync.RWMutex
	started   bool
}

// NewServer creates a new server instance
func NewServer(id string, privKey ed25519.PrivateKey, bootstrapPeers []string) (*Server, error) {
	// Convert ed25519 private key to libp2p private key
	libp2pPrivKey, _, err := crypto.KeyPairFromStdKey(&privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert private key: %w", err)
	}

	// Create P2P network
	network, err := p2p.NewP2PNetwork(id, libp2pPrivKey, bootstrapPeers)
	if err != nil {
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	// Create server
	s := &Server{
		id:      id,
		privKey: privKey,
		pubKey:  privKey.Public().(ed25519.PublicKey),
		network: network,
	}

	// Create fast path handler
	s.fastPath = NewFastPath(network, privKey)

	return s, nil
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("server already started")
	}

	// Start network
	if err := s.network.Start(ctx); err != nil {
		return fmt.Errorf("failed to start network: %w", err)
	}

	// Start consensus
	if s.consensus != nil {
		if err := s.consensus.Start(ctx); err != nil {
			return fmt.Errorf("failed to start consensus: %w", err)
		}
	}

	s.started = true
	return nil
}

// Stop stops the server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return fmt.Errorf("server not started")
	}

	// Stop consensus
	if s.consensus != nil {
		if err := s.consensus.Stop(); err != nil {
			return fmt.Errorf("failed to stop consensus: %w", err)
		}
	}

	// Stop network
	if err := s.network.Stop(); err != nil {
		return fmt.Errorf("failed to stop network: %w", err)
	}

	s.started = false
	return nil
}

// GetID returns the server ID
func (s *Server) GetID() string {
	return s.id
}

// GetPublicKey returns the server's public key
func (s *Server) GetPublicKey() ed25519.PublicKey {
	return s.pubKey
}

// Sign signs data with the server's private key
func (s *Server) Sign(data []byte) []byte {
	return ed25519.Sign(s.privKey, data)
}

// Verify verifies a signature using the server's public key
func (s *Server) Verify(data, sig []byte) bool {
	return ed25519.Verify(s.pubKey, data, sig)
}

// GetTimestamp gets a signed timestamp
func (s *Server) GetTimestamp(ctx context.Context) (*types.SignedTimestamp, error) {
	return s.fastPath.GetTimestamp(ctx)
}
