package client

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rhombus-tech/timeserver-core/core/network"
	"github.com/rhombus-tech/timeserver-core/core/types"
)

// TestHFTClient_GetVerifiedTimestamp tests the HFT client functionality
func TestHFTClient_GetVerifiedTimestamp(t *testing.T) {
	// Create a mock network
	mockNet := network.NewMockNetwork()
	
	// Create test servers
	numServers := 5
	regions := []string{"us-east", "us-west", "eu-west"}
	
	for i := 0; i < numServers; i++ {
		// Generate a key pair for the server
		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("Failed to generate key pair: %v", err)
		}
		
		// Assign region
		region := regions[i%len(regions)]
		
		// Create a mock server
		mockServer := network.NewMockServer(fmt.Sprintf("server-%d", i), region, privKey, pubKey)
		
		// Add it to the network
		mockNet.AddServer(mockServer)
	}
	
	// Create an HFT client with default options
	client := NewHFTClient(mockNet, nil)
	
	// Get a timestamp
	ctx := context.Background()
	ts, err := client.GetVerifiedTimestamp(ctx, "us-east")
	
	// Verify the timestamp was obtained
	if err != nil {
		t.Fatalf("Failed to get timestamp: %v", err)
	}
	
	if ts == nil {
		t.Fatal("Timestamp is nil")
	}
	
	t.Logf("Got timestamp: %v", ts)
}

// TestHFTClient_RegionalProximity tests regional proximity optimization
func TestHFTClient_RegionalProximity(t *testing.T) {
	// Create a mock network
	mockNet := network.NewMockNetwork()
	
	// Create test servers in different regions
	regions := []string{"us-east", "us-west", "eu-west"}
	for i, region := range regions {
		for j := 0; j < 3; j++ { // 3 servers per region
			// Generate a key pair for the server
			pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				t.Fatalf("Failed to generate key pair: %v", err)
			}
			
			// Create a mock server
			serverID := fmt.Sprintf("server-%s-%d", region, j)
			mockServer := network.NewMockServer(serverID, region, privKey, pubKey)
			
			// Add latency based on region (simulating network distance)
			mockServer.SetLatency(time.Duration(i*50) * time.Millisecond)
			
			// Add it to the network
			mockNet.AddServer(mockServer)
		}
	}
	
	// Create an HFT client
	client := NewHFTClient(mockNet, nil)
	
	// Test getting timestamps from specific regions
	ctx := context.Background()
	
	// First get a timestamp from us-east (should be fastest)
	startTime := time.Now()
	ts1, err := client.GetVerifiedTimestamp(ctx, "us-east")
	us_east_time := time.Since(startTime)
	
	if err != nil {
		t.Fatalf("Failed to get timestamp from us-east: %v", err)
	}
	
	// Now get a timestamp from eu-west (should be slower)
	startTime = time.Now()
	ts2, err := client.GetVerifiedTimestamp(ctx, "eu-west")
	eu_west_time := time.Since(startTime)
	
	if err != nil {
		t.Fatalf("Failed to get timestamp from eu-west: %v", err)
	}
	
	// Verify regional preference - timestamps should be from the requested regions
	if ts1.RegionID != "us-east" {
		t.Errorf("Expected timestamp from us-east, got %s", ts1.RegionID)
	}
	
	if ts2.RegionID != "eu-west" {
		t.Errorf("Expected timestamp from eu-west, got %s", ts2.RegionID)
	}
	
	// Log the timing difference
	t.Logf("us-east time: %v, eu-west time: %v", us_east_time, eu_west_time)
}

// TestHFTClient_ProgressiveResponse tests progressive response handling
func TestHFTClient_ProgressiveResponse(t *testing.T) {
	// Create a mock network
	mockNet := network.NewMockNetwork()
	
	// Create test servers with varying latencies
	numServers := 5
	for i := 0; i < numServers; i++ {
		// Generate a key pair for the server
		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("Failed to generate key pair: %v", err)
		}
		
		// Create a mock server
		mockServer := network.NewMockServer(fmt.Sprintf("server-%d", i), "test-region", privKey, pubKey)
		
		// Set increasing latency for each server
		// First server responds in 10ms, second in 100ms, etc.
		mockServer.SetLatency(time.Duration((i+1)*50) * time.Millisecond)
		
		// Add it to the network
		mockNet.AddServer(mockServer)
	}
	
	// Create HFT client with progressive response enabled
	progOpts := DefaultHFTClientOptions()
	progOpts.enableProgressiveResponse = true
	progOpts.quorum = 3
	progClient := NewHFTClient(mockNet, progOpts)
	
	// Create HFT client with progressive response disabled
	nonProgOpts := DefaultHFTClientOptions()
	nonProgOpts.enableProgressiveResponse = false
	nonProgOpts.quorum = 3
	nonProgClient := NewHFTClient(mockNet, nonProgOpts)
	
	// Compare performance
	ctx := context.Background()
	
	// Progressive response should return as soon as the quorum of 3 fastest servers respond
	startTime := time.Now()
	_, err := progClient.GetVerifiedTimestamp(ctx, "")
	progTime := time.Since(startTime)
	
	if err != nil {
		t.Fatalf("Progressive client failed: %v", err)
	}
	
	// Non-progressive response waits for all servers
	startTime = time.Now()
	_, err = nonProgClient.GetVerifiedTimestamp(ctx, "")
	nonProgTime := time.Since(startTime)
	
	if err != nil {
		t.Fatalf("Non-progressive client failed: %v", err)
	}
	
	// Progressive should be faster
	t.Logf("Progressive: %v, Non-progressive: %v", progTime, nonProgTime)
	if progTime >= nonProgTime {
		t.Errorf("Progressive response should be faster, but wasn't: %v >= %v", progTime, nonProgTime)
	}
}

// TestHFTClient_VerificationCaching tests caching of verification results
func TestHFTClient_VerificationCaching(t *testing.T) {
	// Create a mock network
	mockNet := network.NewMockNetwork()
	
	// Create a test server
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}
	
	mockServer := network.NewMockServer("server-1", "test-region", privKey, pubKey)
	mockNet.AddServer(mockServer)
	
	// Create HFT client with caching enabled
	cacheOpts := DefaultHFTClientOptions()
	cacheOpts.enableCaching = true
	cacheClient := NewHFTClient(mockNet, cacheOpts)
	
	// Create HFT client with caching disabled
	noCacheOpts := DefaultHFTClientOptions()
	noCacheOpts.enableCaching = false
	noCacheClient := NewHFTClient(mockNet, noCacheOpts)
	
	// Prepare a context
	ctx := context.Background()
	
	// Create a synthetic signature for the server to verify
	message := []byte("test message")
	signature := ed25519.Sign(privKey, message)
	
	// Function to benchmark verification
	benchmark := func(client *HFTClient, iterations int) time.Duration {
		start := time.Now()
		
		ts := &types.SignedTimestamp{
			ServerID:  "server-1",
			RegionID:  "test-region",
			Time:      time.Now(),
			Signature: signature,
		}
		
		for i := 0; i < iterations; i++ {
			_ = client.verifyTimestampResponse(ts, mockServer)
		}
		
		return time.Since(start)
	}
	
	// Run benchmark with 100 iterations
	iterations := 100
	
	// First verification should prime the cache
	_, _ = cacheClient.GetVerifiedTimestamp(ctx, "")
	
	// Benchmark with caching
	cachedTime := benchmark(cacheClient, iterations)
	
	// Benchmark without caching
	nonCachedTime := benchmark(noCacheClient, iterations)
	
	// Cached verification should be faster
	t.Logf("Cached: %v, Non-cached: %v", cachedTime, nonCachedTime)
	if cachedTime >= nonCachedTime {
		t.Errorf("Cached verification should be faster, but wasn't: %v >= %v", cachedTime, nonCachedTime)
	}
	
	// Verify cache hit stats
	stats := cacheClient.GetStats()
	t.Logf("Cache stats: %v", stats)
}

// TestHFTClient_BatchedRequests tests batched timestamp requests
func TestHFTClient_BatchedRequests(t *testing.T) {
	// Create a mock network
	mockNet := network.NewMockNetwork()
	
	// Create a test server
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}
	
	mockServer := network.NewMockServer("server-1", "test-region", privKey, pubKey)
	mockServer.SetLatency(50 * time.Millisecond) // Set latency to 50ms
	mockNet.AddServer(mockServer)
	
	// Create HFT client with batching enabled
	batchOpts := DefaultHFTClientOptions()
	batchOpts.enableBatching = true
	batchClient := NewHFTClient(mockNet, batchOpts)
	
	// Create HFT client with batching disabled
	noBatchOpts := DefaultHFTClientOptions()
	noBatchOpts.enableBatching = false
	noBatchClient := NewHFTClient(mockNet, noBatchOpts)
	
	// Prepare a context
	ctx := context.Background()
	
	// Benchmark function for batched requests
	benchmarkBatch := func(client *HFTClient, iterations int, batchSize int) time.Duration {
		var wg sync.WaitGroup
		var totalTime int64
		
		// Create atomic counter to track errors
		var errorCount int32
		
		start := time.Now()
		
		// Run iterations in batch
		for i := 0; i < iterations; i++ {
			batchKey := fmt.Sprintf("batch-%d", i/batchSize)
			
			wg.Add(1)
			go func(key string) {
				defer wg.Done()
				
				batchStart := time.Now()
				_, err := client.GetVerifiedTimestampForBatch(ctx, key, "test-region", 200*time.Millisecond)
				batchTime := time.Since(batchStart)
				
				if err != nil {
					atomic.AddInt32(&errorCount, 1)
				}
				
				atomic.AddInt64(&totalTime, int64(batchTime))
			}(batchKey)
		}
		
		wg.Wait()
		
		if errorCount > 0 {
			t.Errorf("%d errors occurred during batch benchmark", errorCount)
		}
		
		return time.Since(start)
	}
	
	// Run benchmark
	iterations := 50
	batchSize := 10 // 10 operations share the same batch key
	
	// Benchmark with batching
	batchedTime := benchmarkBatch(batchClient, iterations, batchSize)
	
	// Benchmark without batching
	nonBatchedTime := benchmarkBatch(noBatchClient, iterations, batchSize)
	
	// Batched should be faster
	t.Logf("Batched: %v, Non-batched: %v", batchedTime, nonBatchedTime)
	if batchedTime >= nonBatchedTime {
		t.Errorf("Batched requests should be faster, but weren't: %v >= %v", batchedTime, nonBatchedTime)
	}
	
	// Verify cache hits stats
	stats := batchClient.GetStats()
	t.Logf("Batch stats: %v", stats)
	
	// Cache hits should be around the number of iterations minus the batch count
	expectedCacheHits := iterations - (iterations / batchSize)
	actualCacheHits := stats["cache_hits"]
	
	t.Logf("Expected ~%d cache hits, got %d", expectedCacheHits, actualCacheHits)
}

// TestHFTClient_DualFormatParameterHandling tests the dual-format parameter handling
func TestHFTClient_DualFormatParameterHandling(t *testing.T) {
	// Test data for different formats
	testCases := []struct {
		name        string
		input       []byte
		expected    []byte
		expectError bool
	}{
		{
			name:        "Empty data",
			input:       []byte{},
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Direct format (no length prefix)",
			input:       []byte("direct format data"),
			expected:    []byte("direct format data"),
			expectError: false,
		},
		{
			name:        "Length-prefixed format (valid)",
			// Use an explicit test string to ensure length matching
			input:       func() []byte {
				testStr := "length prefixed data"
				lenBytes := make([]byte, 4)
				binary.LittleEndian.PutUint32(lenBytes, uint32(len(testStr)))
				return append(lenBytes, []byte(testStr)...)
			}(),
			expected:    []byte("length prefixed data"),
			expectError: false,
		},
		{
			name:        "Length-prefixed format (invalid - too short)",
			input:       append([]byte{100, 0, 0, 0}, []byte("short data")...),
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Length-prefixed format (invalid - too large)",
			input:       append([]byte{0xFF, 0xFF, 0x10, 0x00}, []byte("data")...),
			expected:    []byte{0xFF, 0xFF, 0x10, 0x00, 'd', 'a', 't', 'a'},
			expectError: false, // Should fall back to direct format
		},
		{
			name:        "Less than 4 bytes (direct format fallback)",
			input:       []byte{1, 2, 3},
			expected:    []byte{1, 2, 3},
			expectError: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseWithDualFormatSupport(tc.input)
			
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				
				if string(result) != string(tc.expected) {
					t.Errorf("Expected %q, got %q", string(tc.expected), string(result))
				}
			}
		})
	}
}

// TestHFTClient_Benchmark is a benchmark test for the HFT client
func BenchmarkHFTClient(b *testing.B) {
	// Create a mock network
	mockNet := network.NewMockNetwork()
	
	// Create test servers with varying latencies
	numServers := 5
	for i := 0; i < numServers; i++ {
		// Generate a key pair for the server
		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			b.Fatalf("Failed to generate key pair: %v", err)
		}
		
		// Create a mock server
		mockServer := network.NewMockServer(fmt.Sprintf("server-%d", i), "test-region", privKey, pubKey)
		
		// Set increasing latency for each server
		mockServer.SetLatency(time.Duration((i+1)*10) * time.Millisecond)
		
		// Add it to the network
		mockNet.AddServer(mockServer)
	}
	
	// Create HFT client with all optimizations enabled
	opts := DefaultHFTClientOptions()
	client := NewHFTClient(mockNet, opts)
	
	// Prepare a context
	ctx := context.Background()
	
	// Reset the timer
	b.ResetTimer()
	
	// Run the benchmark
	for i := 0; i < b.N; i++ {
		_, err := client.GetVerifiedTimestamp(ctx, "")
		if err != nil {
			b.Fatalf("Error getting timestamp: %v", err)
		}
	}
}
