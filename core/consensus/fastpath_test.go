package consensus

import (
    "context"
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "github.com/rhombus-tech/timeserver-core/core/types"
)

func TestFastPath(t *testing.T) {
    // Setup test config
    config := FastPathConfig{
        MinSignatures: 5,
        MaxLatency:   50 * time.Millisecond,
        MaxDrift:     10 * time.Millisecond,
    }

    // Create mock threshold signature
    mockThreshold := &ThresholdSignature{}

    // Create test servers
    testServers := []string{
        "server1", "server2", "server3", 
        "server4", "server5", "server6", "server7",
    }

    t.Run("Multiple Servers Success", func(t *testing.T) {
        network := NewMockNetwork()
        // Set different delays for servers
        network.SetDelay("server1", 10*time.Millisecond)
        network.SetDelay("server2", 20*time.Millisecond)
        network.SetDelay("server3", 15*time.Millisecond)
        network.SetDelay("server4", 5*time.Millisecond)
        network.SetDelay("server5", 25*time.Millisecond)

        fp := NewMockFastPath(config, mockThreshold, testServers, network)
        
        ctx := context.Background()
        ts, err := fp.GetTimestamp(ctx)
        
        assert.NoError(t, err)
        assert.NotNil(t, ts)
        assert.NotNil(t, ts.Signature)
    })

    t.Run("Handle Server Failures", func(t *testing.T) {
        network := NewMockNetwork()
        // Make some servers fail
        network.SetFailure("server1", true)
        network.SetFailure("server2", true)
        // But ensure we still have enough working servers
        network.SetDelay("server3", 10*time.Millisecond)
        network.SetDelay("server4", 15*time.Millisecond)
        network.SetDelay("server5", 20*time.Millisecond)
        network.SetDelay("server6", 25*time.Millisecond)
        network.SetDelay("server7", 30*time.Millisecond)

        fp := NewMockFastPath(config, mockThreshold, testServers, network)
        
        ctx := context.Background()
        ts, err := fp.GetTimestamp(ctx)
        
        assert.NoError(t, err)
        assert.NotNil(t, ts)
        assert.NotNil(t, ts.Signature)
    })

    t.Run("Not Enough Servers", func(t *testing.T) {
        network := NewMockNetwork()
        // Make too many servers fail
        network.SetFailure("server1", true)
        network.SetFailure("server2", true)
        network.SetFailure("server3", true)
        network.SetFailure("server4", true)

        fp := NewMockFastPath(config, mockThreshold, testServers, network)
        
        ctx := context.Background()
        ts, err := fp.GetTimestamp(ctx)
        
        assert.Error(t, err)
        assert.Nil(t, ts)
        assert.Contains(t, err.Error(), "not enough valid signatures")
    })

    t.Run("Mixed Delays and Failures", func(t *testing.T) {
        network := NewMockNetwork()
        // Mix of delays and failures
        network.SetFailure("server1", true)
        network.SetDelay("server2", 40*time.Millisecond)
        network.SetDelay("server3", 10*time.Millisecond)
        network.SetFailure("server4", true)
        network.SetDelay("server5", 20*time.Millisecond)
        network.SetDelay("server6", 30*time.Millisecond)
        network.SetDelay("server7", 35*time.Millisecond)

        fp := NewMockFastPath(config, mockThreshold, testServers, network)
        
        ctx := context.Background()
        ts, err := fp.GetTimestamp(ctx)
        
        assert.NoError(t, err)
        assert.NotNil(t, ts)
        assert.NotNil(t, ts.Signature)
    })

    t.Run("Future Timestamp Rejection", func(t *testing.T) {
        network := NewMockNetwork()
        // Set one server to return future timestamp
        network.SetTimestampOffset("server1", 20*time.Millisecond) // Beyond MaxDrift
        network.SetDelay("server2", 5*time.Millisecond)
        network.SetDelay("server3", 10*time.Millisecond)
        network.SetDelay("server4", 15*time.Millisecond)
        network.SetDelay("server5", 20*time.Millisecond)

        fp := NewMockFastPath(config, mockThreshold, testServers, network)
        
        ctx := context.Background()
        _, err := fp.GetTimestamp(ctx)
        
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "timestamp is in the future")
    })

    t.Run("Timestamp Drift Check", func(t *testing.T) {
        network := NewMockNetwork()
        // Set timestamps with too much drift
        network.SetTimestampOffset("server1", -15*time.Millisecond) // Beyond MaxDrift
        network.SetTimestampOffset("server2", 15*time.Millisecond)  // Beyond MaxDrift
        network.SetDelay("server3", 5*time.Millisecond)
        network.SetDelay("server4", 10*time.Millisecond)
        network.SetDelay("server5", 15*time.Millisecond)

        fp := NewMockFastPath(config, mockThreshold, testServers, network)
        
        ctx := context.Background()
        _, err := fp.GetTimestamp(ctx)
        
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "exceeds MaxDrift")
    })

    t.Run("Median Timestamp Selection", func(t *testing.T) {
        network := NewMockNetwork()
        now := time.Now()
        
        // Set timestamps in a specific order to test median selection
        network.SetTimestamp("server1", now.Add(-5*time.Millisecond))
        network.SetTimestamp("server2", now.Add(5*time.Millisecond))
        network.SetTimestamp("server3", now) // This should be selected as median
        network.SetTimestamp("server4", now.Add(-3*time.Millisecond))
        network.SetTimestamp("server5", now.Add(3*time.Millisecond))

        fp := NewMockFastPath(config, mockThreshold, testServers, network)
        
        ctx := context.Background()
        ts, err := fp.GetTimestamp(ctx)
        
        assert.NoError(t, err)
        assert.NotNil(t, ts)
        assert.Equal(t, now.UnixNano(), ts.Time.UnixNano(), "Should select median timestamp")
    })

    t.Run("GetTimestamp Success", func(t *testing.T) {
        fp := NewFastPath(config, mockThreshold, testServers)
        
        ctx := context.Background()
        ts, err := fp.GetTimestamp(ctx)
        
        assert.NoError(t, err)
        assert.NotNil(t, ts)
        assert.NotNil(t, ts.Signature)
        assert.False(t, ts.Time.IsZero())
    })

    t.Run("GetTimestamp Timeout", func(t *testing.T) {
        timeoutConfig := FastPathConfig{
            MinSignatures: 5,
            MaxLatency:   1 * time.Nanosecond,
            MaxDrift:     10 * time.Millisecond,
        }
        
        fp := NewFastPath(timeoutConfig, mockThreshold, testServers)
        
        ctx := context.Background()
        ts, err := fp.GetTimestamp(ctx)
        
        assert.Error(t, err)
        assert.Nil(t, ts)
        assert.Contains(t, err.Error(), "timeout")
    })

    t.Run("Integration with PBFT", func(t *testing.T) {
        state := &State{
            NodeID: "test-node",
            Validators: map[string]bool{
                "server1": true,
                "server2": true,
                "server3": true,
                "server4": true,
                "server5": true,
                "server6": true,
                "server7": true,
            },
        }

        pbft := NewBigBFTConsensus(state, nil)
        
        ctx := context.Background()
        ts := &types.SignedTimestamp{
            Time: time.Now(),
        }
        
        err := pbft.ProposeTimestamp(ctx, ts)
        assert.NoError(t, err)
        
        assert.NotNil(t, pbft.currentBatch)
        assert.True(t, len(pbft.currentBatch.Timestamps) > 0)
    })
}
