package consensus

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// Network defines the interface for network operations
type Network interface {
	Broadcast(ctx context.Context, proto protocol.ID, data []byte) error
	Send(ctx context.Context, p peer.ID, proto protocol.ID, data []byte) error
	RegisterProtocol(id protocol.ID, handler func(context.Context, peer.ID, []byte) error)
	Start(ctx context.Context) error
	Stop() error
}
