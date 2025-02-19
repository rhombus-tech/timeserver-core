package consensus

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// Network defines the network interface required by consensus
type Network interface {
	// RegisterProtocol registers a protocol handler
	RegisterProtocol(protocol.ID, func(context.Context, peer.ID, []byte) error)

	// Broadcast sends a message to all peers
	Broadcast(context.Context, protocol.ID, []byte) error

	// Send sends a message to a specific peer
	Send(context.Context, peer.ID, protocol.ID, []byte) error

	// Start starts the network
	Start(context.Context) error

	// Stop stops the network
	Stop() error
}
