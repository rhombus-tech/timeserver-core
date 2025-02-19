package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rhombus-tech/timeserver-core/core/types"
)

// ConsensusState represents the current state of consensus
type ConsensusState struct {
	NodeID     string            // Node ID
	View       uint64            // Current view number
	Validators map[string]bool   // Set of validator IDs
	Leader     string           // Current leader ID
	LastDigest []byte          // Last consensus digest
	LastTime   time.Time       // Last consensus timestamp
}

// BigBFTConsensus implements multi-leader consensus for timeserver
type BigBFTConsensus struct {
	state           *State
	fastPath        FastPathHandler
	thresholdSigs   *ThresholdSignature
	batchWindow     time.Duration
	currentBatch    *TimestampBatch
	batches        map[uint64]*TimestampBatch  // BatchID -> Batch
	votes          map[uint64]map[string][]byte // BatchID -> NodeID -> Vote
	commits        map[uint64]map[string][]byte
	mu             sync.RWMutex
	network        Network
}

// NewBigBFTConsensus creates a new BigBFT consensus instance
func NewBigBFTConsensus(state *State, network Network) *BigBFTConsensus {
	// Configure fast path with 5-7 servers
	fastPathConfig := FastPathConfig{
		MinSignatures: 5,              // Require 5 signatures minimum
		MaxLatency:   50 * time.Millisecond,  // 50ms timeout
		MaxDrift:     10 * time.Millisecond,  // 10ms maximum drift
	}

	// Convert validator map to slice for nearest servers
	var validators []string
	for v := range state.Validators {
		validators = append(validators, v)
	}
	
	// Use up to 7 validators for fast path
	var nearestServers []string
	if len(validators) > 7 {
		nearestServers = validators[:7]
	} else {
		nearestServers = validators
	}
	
	fastPath := &FastPathImpl{
		config:        fastPathConfig,
		thresholdSigs: NewThresholdSignature(nil, nil, 5), // Configure properly
		nearestServers: nearestServers,
	}

	return &BigBFTConsensus{
		state:         state,
		fastPath:      fastPath,
		batchWindow:   time.Second,    // 1s batch window for normal consensus
		batches:      make(map[uint64]*TimestampBatch),
		votes:        make(map[uint64]map[string][]byte),
		commits:      make(map[uint64]map[string][]byte),
		network:      network,
	}
}

// ProposeTimestamp proposes a new timestamp for consensus
func (b *BigBFTConsensus) ProposeTimestamp(ctx context.Context, ts *types.SignedTimestamp) error {
	// Try fast path first
	fastTs, err := b.fastPath.GetTimestamp(ctx)
	if err == nil {
		ts = fastTs  // Use the fast path timestamp
		return nil
	}

	// Fall back to normal consensus if fast path fails
	b.mu.Lock()
	defer b.mu.Unlock()

	// Create new batch if needed
	if b.currentBatch == nil {
		b.currentBatch = &TimestampBatch{
			BatchID:    uint64(time.Now().UnixNano()),
			Leader:     b.state.GetNodeID(),
			Timestamps: make([]*types.SignedTimestamp, 0),
		}
	}

	// Add timestamp to current batch
	b.currentBatch.Timestamps = append(b.currentBatch.Timestamps, ts)

	// Start consensus if batch window expired
	go b.processBatchAfterWindow(ctx, b.currentBatch.BatchID)

	return nil
}

// processBatchAfterWindow processes the batch after the window expires
func (b *BigBFTConsensus) processBatchAfterWindow(ctx context.Context, batchID uint64) {
	time.Sleep(b.batchWindow)
	
	b.mu.Lock()
	if b.currentBatch != nil && b.currentBatch.BatchID == batchID {
		batch := b.currentBatch
		b.currentBatch = nil
		b.mu.Unlock()
		
		b.startConsensus(ctx, batch)
	} else {
		b.mu.Unlock()
	}
}

// startConsensus starts the consensus process for a batch
func (b *BigBFTConsensus) startConsensus(ctx context.Context, batch *TimestampBatch) error {
	// Initialize vote maps for this batch
	b.mu.Lock()
	b.batches[batch.BatchID] = batch
	b.votes[batch.BatchID] = make(map[string][]byte)
	b.commits[batch.BatchID] = make(map[string][]byte)
	b.mu.Unlock()

	// Serialize batch for broadcast
	batchData, err := json.Marshal(batch)
	if err != nil {
		return err
	}

	// Broadcast batch to all validators
	msg := &ConsensusMessage{
		Type: "propose",
		View: batch.BatchID,
		From: b.state.GetNodeID(),
		Data: batchData,
	}

	return b.broadcast(ctx, msg)
}

// handleVote handles incoming vote messages with aggregation
func (b *BigBFTConsensus) handleVote(ctx context.Context, msg *ConsensusMessage) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	batchID := msg.View
	if _, exists := b.votes[batchID]; !exists {
		return nil // Ignore votes for unknown batches
	}

	// Add vote
	b.votes[batchID][msg.From] = msg.Data

	// Check if we have enough votes
	if len(b.votes[batchID]) >= b.getQuorumSize() {
		// Aggregate votes using threshold signatures
		aggregatedVote, err := b.thresholdSigs.Aggregate(b.votes[batchID])
		if err != nil {
			return err
		}

		// Move to commit phase
		commitMsg := &ConsensusMessage{
			Type: "commit",
			View: batchID,
			From: b.state.GetNodeID(),
			Data: aggregatedVote,
		}

		return b.broadcast(ctx, commitMsg)
	}

	return nil
}

// getQuorumSize returns the number of votes needed for consensus
func (b *BigBFTConsensus) getQuorumSize() int {
	return (len(b.state.GetValidators()) * 2 / 3) + 1
}

// broadcast sends a message to all validators
func (b *BigBFTConsensus) broadcast(ctx context.Context, msg *ConsensusMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return b.network.Broadcast(ctx, ConsensusProtocolID, data)
}
