package consensus

import (
	"time"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// Protocol ID for consensus messages
var ConsensusProtocolID = protocol.ID("/timeserver/consensus/1.0.0")

// Protocol constants
const (
	// MaxClockDrift is the maximum allowed clock drift
	MaxClockDrift = 100 * time.Millisecond
)
