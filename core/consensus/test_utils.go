package consensus

import (
    "context"
    "fmt"
    "sort"
    "strings"
    "sync"
    "time"
    "github.com/rhombus-tech/timeserver-core/core/types"
)

// MockNetwork simulates network delays and failures
type MockNetwork struct {
    Delays map[string]time.Duration
    Failures map[string]bool
    TimestampOffsets map[string]time.Duration
    FixedTimestamps map[string]time.Time
    Mu sync.RWMutex
}

func NewMockNetwork() *MockNetwork {
    return &MockNetwork{
        Delays: make(map[string]time.Duration),
        Failures: make(map[string]bool),
        TimestampOffsets: make(map[string]time.Duration),
        FixedTimestamps: make(map[string]time.Time),
    }
}

func (n *MockNetwork) SetDelay(serverID string, delay time.Duration) {
    n.Mu.Lock()
    defer n.Mu.Unlock()
    n.Delays[serverID] = delay
}

func (n *MockNetwork) SetFailure(serverID string, fails bool) {
    n.Mu.Lock()
    defer n.Mu.Unlock()
    n.Failures[serverID] = fails
}

func (n *MockNetwork) SetTimestampOffset(serverID string, offset time.Duration) {
    n.Mu.Lock()
    defer n.Mu.Unlock()
    n.TimestampOffsets[serverID] = offset
}

func (n *MockNetwork) SetTimestamp(serverID string, ts time.Time) {
    n.Mu.Lock()
    defer n.Mu.Unlock()
    n.FixedTimestamps[serverID] = ts
}

// MockFastPath wraps FastPathImpl with mock network
type MockFastPath struct {
    config         FastPathConfig
    thresholdSigs  *ThresholdSignature
    nearestServers []string
    network        *MockNetwork
    mu             sync.RWMutex
}

func NewMockFastPath(config FastPathConfig, threshold *ThresholdSignature, servers []string, network *MockNetwork) *MockFastPath {
    return &MockFastPath{
        config:        config,
        thresholdSigs: threshold,
        nearestServers: servers,
        network:       network,
    }
}

// GetTimestamp implements FastPathHandler interface
func (fp *MockFastPath) GetTimestamp(ctx context.Context) (*types.SignedTimestamp, error) {
    fp.mu.Lock()
    defer fp.mu.Unlock()

    now := time.Now()
    
    // Create timestamp
    ts := &types.SignedTimestamp{
        Time: now,
    }
    
    // Create channels for collecting signatures and timestamps
    type serverResponse struct {
        serverID string
        sig []byte
        ts time.Time
    }
    sigChan := make(chan serverResponse, fp.config.MinSignatures)
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
            
            // Get the server's timestamp
            fp.network.Mu.RLock()
            serverTime := now
            if offset, ok := fp.network.TimestampOffsets[id]; ok {
                serverTime = serverTime.Add(offset)
            }
            if fixedTs, ok := fp.network.FixedTimestamps[id]; ok {
                serverTime = fixedTs
            }
            fp.network.Mu.RUnlock()
            
            sigChan <- serverResponse{id, sig, serverTime}
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
                remainingServers--
                continue
            }
            
            // Check drift against other timestamps
            for sid, t := range timestamps {
                if drift := resp.ts.Sub(t); drift > fp.config.MaxDrift || drift < -fp.config.MaxDrift {
                    errChan <- fmt.Errorf("timestamp drift between %s and %s exceeds MaxDrift: %v", resp.serverID, sid, drift)
                    remainingServers--
                    continue
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
            
        case err := <-errChan:
            if strings.Contains(err.Error(), "timestamp") {
                // For timestamp errors, we want to propagate the specific error
                return nil, err
            }
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

// requestSignature simulates network delays and failures
func (fp *MockFastPath) requestSignature(ctx context.Context, serverID string, ts *types.SignedTimestamp) ([]byte, error) {
    fp.network.Mu.RLock()
    delay := fp.network.Delays[serverID]
    fails := fp.network.Failures[serverID]
    offset := fp.network.TimestampOffsets[serverID]
    fixedTs, hasFixedTs := fp.network.FixedTimestamps[serverID]
    fp.network.Mu.RUnlock()

    // Check for failure first before delay
    if fails {
        return nil, fmt.Errorf("server %s failed", serverID)
    }

    // Simulate network delay
    if delay > 0 {
        select {
        case <-time.After(delay):
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }

    // Apply timestamp modifications
    if hasFixedTs {
        ts.Time = fixedTs
    } else if offset != 0 {
        ts.Time = ts.Time.Add(offset)
    }

    return []byte(fmt.Sprintf("sig-from-%s", serverID)), nil
}
