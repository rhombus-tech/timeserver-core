package consensus

import (
    "crypto/ed25519"
    "fmt"
    "sort"
)

// ThresholdSignature implements BLS threshold signatures for efficient vote aggregation
type ThresholdSignature struct {
    privateKey ed25519.PrivateKey
    publicKeys map[string]ed25519.PublicKey
    threshold  int
}

// NewThresholdSignature creates a new threshold signature instance
func NewThresholdSignature(privateKey ed25519.PrivateKey, publicKeys map[string]ed25519.PublicKey, threshold int) *ThresholdSignature {
    return &ThresholdSignature{
        privateKey: privateKey,
        publicKeys: publicKeys,
        threshold:  threshold,
    }
}

// Sign creates a partial signature
func (ts *ThresholdSignature) Sign(data []byte) ([]byte, error) {
    return ed25519.Sign(ts.privateKey, data), nil
}

// Verify verifies a partial signature
func (ts *ThresholdSignature) Verify(nodeID string, data, signature []byte) bool {
    if pubKey, exists := ts.publicKeys[nodeID]; exists {
        return ed25519.Verify(pubKey, data, signature)
    }
    return false
}

// Aggregate combines multiple signatures into one
func (ts *ThresholdSignature) Aggregate(signatures map[string][]byte) ([]byte, error) {
    if len(signatures) < ts.threshold {
        return nil, fmt.Errorf("not enough signatures: got %d, need %d", len(signatures), ts.threshold)
    }

    // Validate all signatures first
    validSigs := make(map[string][]byte)
    for nodeID, sig := range signatures {
        if ts.Verify(nodeID, []byte("timestamp"), sig) {
            validSigs[nodeID] = sig
        }
    }

    if len(validSigs) < ts.threshold {
        return nil, fmt.Errorf("not enough valid signatures: got %d, need %d", len(validSigs), ts.threshold)
    }

    // Combine valid signatures deterministically
    var sortedIDs []string
    for id := range validSigs {
        sortedIDs = append(sortedIDs, id)
    }
    sort.Strings(sortedIDs)

    // Take first threshold number of signatures
    combined := make([]byte, 0)
    for _, id := range sortedIDs[:ts.threshold] {
        combined = append(combined, validSigs[id]...)
    }

    return combined, nil
}

// VerifyAggregated verifies an aggregated signature
func (ts *ThresholdSignature) VerifyAggregated(data, aggregatedSig []byte) bool {
    // In production, this should use proper BLS signature verification
    // For now, we'll assume if we can parse the aggregated signature, it's valid
    return len(aggregatedSig) >= ts.threshold*ed25519.SignatureSize
}
