package network

import (
	"crypto/ed25519"
	"crypto/rand"

	"github.com/rhombus-tech/timeserver-core/core/types"
)

// MockTimeServer implements the TimeServer interface for testing
type MockTimeServer struct {
	id        string
	region    string
	pubKey    ed25519.PublicKey
	privKey   ed25519.PrivateKey
}

func NewMockTimeServer(id string) (*MockTimeServer, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &MockTimeServer{
		id:      id,
		region:  "default",
		pubKey:  pub,
		privKey: priv,
	}, nil
}

func (m *MockTimeServer) GetID() string {
	return m.id
}

func (m *MockTimeServer) GetRegion() string {
	return m.region
}

func (m *MockTimeServer) SetRegion(region string) error {
	m.region = region
	return nil
}

func (m *MockTimeServer) GetPublicKey() ed25519.PublicKey {
	return m.pubKey
}

func (m *MockTimeServer) Sign(message []byte) ([]byte, error) {
	return ed25519.Sign(m.privKey, message), nil
}

func (m *MockTimeServer) Verify(message []byte, signature []byte) error {
	if !ed25519.Verify(m.pubKey, message, signature) {
		return types.ErrInvalidSignature
	}
	return nil
}
