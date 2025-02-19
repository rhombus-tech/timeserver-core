package consensus

import (
	"encoding/json"
	"sync"

	"github.com/rhombus-tech/timeserver-core/core/types"
)

// ConsensusState represents the current state of PBFT consensus
type ConsensusState struct {
	View        uint64
	Leader      string
	Validators  []string
	LastBlock   []byte
	Timestamps  map[string]*types.SignedTimestamp
}

// Message types for PBFT consensus
type PrepareMessage struct {
	ValidatorID string
	Timestamp   *types.SignedTimestamp
	Digest      []byte
}

type CommitMessage struct {
	ValidatorID string
	Timestamp   *types.SignedTimestamp
	Digest      []byte
}

type ViewChangeMessage struct {
	ValidatorID string
	NewView     uint64
	LastDigest  []byte
}

// Encode/Decode methods for messages
func (m *PrepareMessage) Encode() ([]byte, error) {
	return json.Marshal(m)
}

func (m *PrepareMessage) Decode(data []byte) error {
	return json.Unmarshal(data, m)
}

func (m *CommitMessage) Encode() ([]byte, error) {
	return json.Marshal(m)
}

func (m *CommitMessage) Decode(data []byte) error {
	return json.Unmarshal(data, m)
}

func (m *ViewChangeMessage) Encode() ([]byte, error) {
	return json.Marshal(m)
}

func (m *ViewChangeMessage) Decode(data []byte) error {
	return json.Unmarshal(data, m)
}

// PBFTConsensus implements PBFT consensus for timeserver network
type PBFTConsensus struct {
	state       *ConsensusState
	prepares    map[string][]string  // msg -> validators
	commits     map[string][]string
	threshold   int
	mu          sync.RWMutex
}

// NewPBFTConsensus creates a new PBFT consensus instance
func NewPBFTConsensus(threshold int) *PBFTConsensus {
	return &PBFTConsensus{
		state: &ConsensusState{
			View:       0,
			Timestamps: make(map[string]*types.SignedTimestamp),
		},
		prepares:   make(map[string][]string),
		commits:    make(map[string][]string),
		threshold:  threshold,
	}
}

// ProposeTimestamp proposes a new timestamp for consensus
func (p *PBFTConsensus) ProposeTimestamp(ts *types.SignedTimestamp) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// TODO: Implement PBFT consensus logic
	// 1. Leader broadcasts PRE-PREPARE
	// 2. Validators send PREPARE
	// 3. On 2f+1 PREPARE, send COMMIT
	// 4. On 2f+1 COMMIT, timestamp is committed

	return nil
}

// OnPrepare handles incoming prepare messages
func (p *PBFTConsensus) OnPrepare(prepare *PrepareMessage) error {
	// TODO: Implement prepare phase
	return nil
}

// OnCommit handles incoming commit messages
func (p *PBFTConsensus) OnCommit(commit *CommitMessage) error {
	// TODO: Implement commit phase
	return nil
}

// ViewChange initiates a view change
func (p *PBFTConsensus) ViewChange() error {
	// TODO: Implement view change protocol
	return nil
}
