package consensus

import (
    "context"
    "fmt"
    "sort"
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

    now := time.Now()
    
    // Create timestamp
    ts := &types.SignedTimestamp{
        Time: now,
    }
    
    // Create channels for collecting signatures and timestamps
    sigChan := make(chan struct{
        serverID string
        sig []byte
        ts time.Time
    }, fp.config.MinSignatures)
    errChan := make(chan error, len(fp.nearestServers))
    
    // Set timeout for fast path
    ctx, cancel := context.WithTimeout(ctx, fp.config.MaxLatency)
    defer cancel()
    
    // Request signatures in parallel from nearest servers
    validSignatures := 0
    
    // Start all requests first
    for _, serverID := range fp.nearestServers {
        go func(id string) {
            sig, err := fp.requestSignature(ctx, id, ts)
            if err != nil {
                errChan <- fmt.Errorf("server %s failed: %v", id, err)
                return
            }
            sigChan <- struct{
                serverID string
                sig []byte
                ts time.Time
            }{id, sig, time.Now()}
        }(serverID)
    }
    
    // Collect signatures
    signatures := make(map[string][]byte)
    timestamps := make(map[string]time.Time)
    remainingServers := len(fp.nearestServers)
    
    for {
        select {
        case resp := <-sigChan:
            // Validate timestamp is not from future
            if resp.ts.After(now.Add(fp.config.MaxDrift)) {
                errChan <- fmt.Errorf("server %s timestamp is in the future: %v", resp.serverID, resp.ts)
                continue
            }
            
            // Check drift against other timestamps
            for sid, t := range timestamps {
                drift := resp.ts.Sub(t)
                if drift > fp.config.MaxDrift || drift < -fp.config.MaxDrift {
                    remainingServers--
                    return nil, fmt.Errorf("timestamp drift between servers %s and %s exceeds MaxDrift (%v > %v)", resp.serverID, sid, drift, fp.config.MaxDrift)
                }
            }
            
            signatures[resp.serverID] = resp.sig
            timestamps[resp.serverID] = resp.ts
            validSignatures++
            remainingServers--
            
            if validSignatures >= fp.config.MinSignatures {
                // Use median timestamp
                var allTimestamps []time.Time
                for _, t := range timestamps {
                    allTimestamps = append(allTimestamps, t)
                }
                sort.Slice(allTimestamps, func(i, j int) bool {
                    return allTimestamps[i].Before(allTimestamps[j])
                })
                ts.Time = allTimestamps[len(allTimestamps)/2]
                
                // Aggregate signatures
                aggregatedSig, err := fp.thresholdSigs.Aggregate(signatures)
                if err != nil {
                    return nil, fmt.Errorf("fast path: failed to aggregate signatures: %v", err)
                }
                ts.Signature = aggregatedSig
                return ts, nil
            }
            
            // Check if we can still get enough signatures
            if validSignatures + remainingServers < fp.config.MinSignatures {
                return nil, fmt.Errorf("fast path: not enough valid signatures available")
            }
            
        case <-errChan:
            remainingServers--
            // Check if we can still get enough signatures
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
