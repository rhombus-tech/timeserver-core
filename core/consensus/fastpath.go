package consensus

import (
    "context"
    "fmt"
    "sync"
    "time"
    "github.com/rhombus-tech/timeserver-core/core/types"
)

// FastPathConfig defines configuration for fast path consensus
type FastPathConfig struct {
    MinSignatures    int           // 5-7 servers required for fast path
    MaxLatency      time.Duration  // Maximum time to wait for signatures
    MaxDrift        time.Duration  // Maximum allowed drift between timestamps
}

// FastPathImpl implements the fast path consensus mechanism
type FastPathImpl struct {
    config         FastPathConfig
    thresholdSigs  *ThresholdSignature
    nearestServers []string        // Pre-selected closest servers for fast path
    mu             sync.RWMutex
}

// NewFastPath creates a new FastPath instance
func NewFastPath(config FastPathConfig, thresholdSigs *ThresholdSignature, servers []string) *FastPathImpl {
    return &FastPathImpl{
        config:        config,
        thresholdSigs: thresholdSigs,
        nearestServers: servers,
    }
}

// requestSignature requests a signature from a single server
func (fp *FastPathImpl) requestSignature(ctx context.Context, serverID string, ts *types.SignedTimestamp) ([]byte, error) {
    // For now, simulate getting signature
    return []byte("signature"), nil
}

// GetTimestamp implements FastPathHandler interface
func (fp *FastPathImpl) GetTimestamp(ctx context.Context) (*types.SignedTimestamp, error) {
    fp.mu.Lock()
    defer fp.mu.Unlock()

    // Create timestamp
    ts := &types.SignedTimestamp{
        Time: time.Now(),
    }
    
    // Create channels for collecting signatures
    sigChan := make(chan map[string][]byte, fp.config.MinSignatures)
    errChan := make(chan error, len(fp.nearestServers))
    
    // Set timeout for fast path
    ctx, cancel := context.WithTimeout(ctx, fp.config.MaxLatency)
    defer cancel()
    
    // Request signatures in parallel from nearest servers
    validSignatures := 0
    failedServers := 0
    
    for _, serverID := range fp.nearestServers {
        go func(id string) {
            sig, err := fp.requestSignature(ctx, id, ts)
            if err != nil {
                errChan <- fmt.Errorf("server %s failed: %v", id, err)
                return
            }
            sigChan <- map[string][]byte{id: sig}
        }(serverID)
    }
    
    // Collect signatures
    signatures := make(map[string][]byte)
    
    for {
        select {
        case sig := <-sigChan:
            for id, s := range sig {
                signatures[id] = s
                validSignatures++
            }
            if validSignatures >= fp.config.MinSignatures {
                // Aggregate signatures
                aggregatedSig, err := fp.thresholdSigs.Aggregate(signatures)
                if err != nil {
                    return nil, fmt.Errorf("fast path: failed to aggregate signatures: %v", err)
                }
                ts.Signature = aggregatedSig
                return ts, nil
            }
            
        case <-errChan:
            failedServers++
            // Check if we can still get enough signatures
            remainingServers := len(fp.nearestServers) - failedServers
            if validSignatures + remainingServers < fp.config.MinSignatures {
                return nil, fmt.Errorf("fast path: not enough valid signatures available")
            }
            
        case <-ctx.Done():
            return nil, fmt.Errorf("fast path: timeout waiting for signatures")
        }
    }
}

// FastPathHandler defines the interface for fast path consensus
type FastPathHandler interface {
    // GetTimestamp attempts to get a timestamp via fast path
    GetTimestamp(ctx context.Context) (*types.SignedTimestamp, error)
}
