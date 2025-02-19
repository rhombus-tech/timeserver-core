package consensus

import (
    "context"
    "fmt"
    "sync"
    "time"
    "github.com/rhombus-tech/timeserver-core/core/types"
)

// MockNetwork simulates network delays and failures
type MockNetwork struct {
    Delays map[string]time.Duration
    Failures map[string]bool
    Mu sync.RWMutex
}

func NewMockNetwork() *MockNetwork {
    return &MockNetwork{
        Delays: make(map[string]time.Duration),
        Failures: make(map[string]bool),
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
    
    // Start all requests first
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
    remainingServers := len(fp.nearestServers)
    
    for {
        select {
        case sig := <-sigChan:
            for id, s := range sig {
                signatures[id] = s
                validSignatures++
            }
            remainingServers--
            if validSignatures >= fp.config.MinSignatures {
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

// requestSignature simulates network delays and failures
func (fp *MockFastPath) requestSignature(ctx context.Context, serverID string, ts *types.SignedTimestamp) ([]byte, error) {
    fp.network.Mu.RLock()
    delay := fp.network.Delays[serverID]
    fails := fp.network.Failures[serverID]
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

    return []byte(fmt.Sprintf("sig-from-%s", serverID)), nil
}
