package server

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rhombus-tech/timeserver-core/core/p2p"
	"github.com/rhombus-tech/timeserver-core/core/types"
)

const (
	fastPathThreshold = 3    // Number of quick responses needed
	fastPathTimeout   = 50 * time.Millisecond
)

// FastPath handles fast path timestamp requests
type FastPath struct {
	network *p2p.P2PNetwork
	privKey ed25519.PrivateKey
	mu      sync.RWMutex
	id      string
}

// NewFastPath creates a new FastPath instance
func NewFastPath(network *p2p.P2PNetwork, privKey ed25519.PrivateKey) *FastPath {
	fp := &FastPath{
		network: network,
		privKey: privKey,
		id:      network.ID(),
	}

	// Register fast path message handler
	network.RegisterProtocol(p2p.FastPathProtocol, fp.handleMessage)

	return fp
}

// GetTimestamp gets a signed timestamp via fast path
func (fp *FastPath) GetTimestamp(ctx context.Context) (*types.SignedTimestamp, error) {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	// Generate timestamp
	now := time.Now()
	ts := &types.SignedTimestamp{
		Time:      now,
		ServerID:  fp.id,
		Signature: ed25519.Sign(fp.privKey, types.TimestampToBytes(now)),
	}

	return ts, nil
}

// handleMessage handles incoming fast path messages
func (fp *FastPath) handleMessage(data []byte) error {
	var msg p2p.TimestampMessage
	if err := p2p.DecodeMessage(data, &msg); err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}

	// Generate timestamp
	now := time.Now()
	ts := &types.SignedTimestamp{
		Time:      now,
		ServerID:  fp.id,
		Signature: ed25519.Sign(fp.privKey, types.TimestampToBytes(now)),
	}

	// Create response message
	resp := p2p.NewTimestampMessage(peer.ID(fp.id), fp.id, ts)

	// Send response
	respData, err := p2p.EncodeMessage(resp)
	if err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}

	return fp.network.Send(context.Background(), msg.From, p2p.FastPathProtocol, respData)
}
