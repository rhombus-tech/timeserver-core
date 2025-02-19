package consensus

import (
    "encoding/json"
)

// BatchVote represents a vote for a batch of timestamps
type BatchVote struct {
    BatchID    uint64   // ID of the batch being voted on
    Signatures [][]byte // signatures for each timestamp in the batch
}

// BatchCommit represents a commit for a batch of timestamps
type BatchCommit struct {
    BatchID          uint64 // ID of the batch being committed
    AggregatedVotes []byte // aggregated threshold signatures
}

// Encode encodes a BatchVote to bytes
func (v *BatchVote) Encode() ([]byte, error) {
    return json.Marshal(v)
}

// Decode decodes a BatchVote from bytes
func (v *BatchVote) Decode(data []byte) error {
    return json.Unmarshal(data, v)
}

// Encode encodes a BatchCommit to bytes
func (c *BatchCommit) Encode() ([]byte, error) {
    return json.Marshal(c)
}

// Decode decodes a BatchCommit from bytes
func (c *BatchCommit) Decode(data []byte) error {
    return json.Unmarshal(data, c)
}
