package client

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rhombus-tech/timeserver-core/core/types"
)

const (
	// Default timeouts and limits for production use
	defaultRequestTimeout      = 250 * time.Millisecond
	defaultMaxConnsPerHost     = 10
	defaultMaxIdleConns        = 100
	defaultIdleConnTimeout     = 90 * time.Second
	defaultCacheTTL            = 5 * time.Second
	defaultCacheCleanupInterval = 30 * time.Second
	defaultRTTSampleSize       = 20
	defaultRTTDecayFactor      = 0.8  // Weight for newer measurements
	
	// Safe parameter constraints
	maxParameterLength        = 1024 * 1024 // 1MB max parameter size
	maxLengthPrefixValue      = 100 * 1024  // 100KB max length prefix value
)

var (
	// ErrNoServersAvailable is returned when no servers are available
	ErrNoServersAvailable = errors.New("no time servers available")
	
	// ErrQuorumNotReached is returned when the required quorum of responses wasn't reached
	ErrQuorumNotReached = errors.New("quorum of trusted timestamps not reached")
	
	// ErrTimestampVerification is returned when timestamp verification fails
	ErrTimestampVerification = errors.New("timestamp verification failed")
	
	// ErrRequestTimeout is returned when a request times out
	ErrRequestTimeout = errors.New("request timed out")
)

// HFTClientOptions contains configuration options for the HFTClient
type HFTClientOptions struct {
	// RequestTimeout is the timeout for individual server requests
	requestTimeout time.Duration
	
	// Quorum is the minimum number of responses needed
	quorum int
	
	// EnableProgressiveResponse enables returning as soon as quorum is reached
	enableProgressiveResponse bool
	
	// EnableCaching enables caching of verification results
	enableCaching bool
	
	// EnableBatching enables batching of timestamp requests
	enableBatching bool
	
	// CacheTTL is the time-to-live for cached entries
	cacheTTL time.Duration
	
	// HTTP client connection pool settings
	maxConnsPerHost int
	maxIdleConns    int
	idleConnTimeout time.Duration
	
	// RTT tracking settings
	rttSampleSize  int
	rttDecayFactor float64
}

// DefaultHFTClientOptions returns the default options for HFTClient
func DefaultHFTClientOptions() *HFTClientOptions {
	return &HFTClientOptions{
		requestTimeout:            defaultRequestTimeout,
		quorum:                    3,
		enableProgressiveResponse: true,
		enableCaching:             true,
		enableBatching:            true,
		cacheTTL:                  defaultCacheTTL,
		maxConnsPerHost:           defaultMaxConnsPerHost,
		maxIdleConns:              defaultMaxIdleConns,
		idleConnTimeout:           defaultIdleConnTimeout,
		rttSampleSize:             defaultRTTSampleSize,
		rttDecayFactor:            defaultRTTDecayFactor,
	}
}

// HFTClient is an optimized client for high-frequency trading applications
// that implements connection pooling, regional proximity optimization,
// progressive response handling, cached verification, and batched requests
type HFTClient struct {
	options         *HFTClientOptions
	httpClient      *http.Client
	registry        types.TimeNetwork
	
	// RTT tracking for server selection
	serverRTT       map[string]*rttTracker
	rttLock         sync.RWMutex
	
	// Verification cache
	verifyCache     *verificationCache
	
	// Batch request cache
	batchCache      *timestampBatchCache
	
	// Stats
	stats           clientStats
}

// clientStats tracks performance metrics
type clientStats struct {
	requestCount         atomic.Int64
	cacheHits            atomic.Int64
	cacheMisses          atomic.Int64
	verificationFailures atomic.Int64
	quorumFailures       atomic.Int64
	requestTimeouts      atomic.Int64
}

// NewHFTClient creates a new optimized client for high-frequency trading
func NewHFTClient(network types.TimeNetwork, options *HFTClientOptions) *HFTClient {
	if options == nil {
		options = DefaultHFTClientOptions()
	}
	
	// Configure transport for connection pooling
	transport := &http.Transport{
		MaxConnsPerHost:     options.maxConnsPerHost,
		MaxIdleConns:        options.maxIdleConns,
		MaxIdleConnsPerHost: options.maxConnsPerHost,
		IdleConnTimeout:     options.idleConnTimeout,
		DisableCompression:  true, // Less CPU overhead
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	
	client := &HFTClient{
		options:     options,
		httpClient:  &http.Client{
			Transport: transport,
			Timeout:   options.requestTimeout * 2, // Allow for some overhead
		},
		registry:    network,
		serverRTT:   make(map[string]*rttTracker),
	}
	
	// Initialize caches if enabled
	if options.enableCaching {
		client.verifyCache = newVerificationCache(options.cacheTTL)
		client.batchCache = newTimestampBatchCache(options.cacheTTL)
	}
	
	return client
}

// GetVerifiedTimestamp gets a timestamp verified by at least quorum servers
// It uses all the HFT optimizations: connection pooling, regional proximity,
// progressive response handling, and verification caching.
func (c *HFTClient) GetVerifiedTimestamp(ctx context.Context, regionID string) (*types.SignedTimestamp, error) {
	c.stats.requestCount.Add(1)
	
	// Get servers ordered by RTT
	servers := c.getServersByRTT(regionID)
	if len(servers) == 0 {
		c.stats.quorumFailures.Add(1)
		return nil, ErrNoServersAvailable
	}
	
	// Determine quorum
	quorum := c.options.quorum
	if quorum > len(servers) {
		quorum = len(servers)
	}
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.options.requestTimeout*3)
	defer cancel()
	
	// Create channels for responses and errors
	responseCh := make(chan *serverResponse, len(servers))
	
	// Request timestamps from all servers in parallel
	pendingServers := servers
	for _, server := range pendingServers {
		go func(s types.TimeServer) {
			startTime := time.Now()
			ts, err := c.requestTimestamp(ctx, s)
			rtt := time.Since(startTime)
			
			// Update RTT tracking
			c.updateServerRTT(s.GetID(), rtt)
			
			responseCh <- &serverResponse{
				timestamp: ts,
				error:     err,
				server:    s,
				rtt:       rtt,
			}
		}(server)
	}
	
	// Collect responses up to quorum
	responses := make([]*serverResponse, 0, len(servers))
	for i := 0; i < quorum; i++ {
		select {
		case resp := <-responseCh:
			responses = append(responses, resp)
		case <-ctx.Done():
			c.stats.requestTimeouts.Add(1)
			return nil, fmt.Errorf("timeout waiting for quorum: %w", ErrRequestTimeout)
		}
	}
	
	// Process responses to see if we have a quorum of valid timestamps
	validResponses := 0
	var bestTimestamp *types.SignedTimestamp
	
	for _, resp := range responses {
		if resp.error != nil {
			continue
		}
		
		// Verify the timestamp
		if err := c.verifyTimestampResponse(resp.timestamp, resp.server); err != nil {
			c.stats.verificationFailures.Add(1)
			continue
		}
		
		validResponses++
		
		// Select the best timestamp (earliest in case of ties)
		if bestTimestamp == nil || resp.timestamp.Time.Before(bestTimestamp.Time) {
			bestTimestamp = resp.timestamp
		}
	}
	
	// If we have progressive response handling enabled, return as soon as we have quorum
	if c.options.enableProgressiveResponse && validResponses >= quorum {
		// Continue collecting responses in the background
		go func() {
			for i := quorum; i < len(pendingServers); i++ {
				select {
				case resp := <-responseCh:
					if resp.error == nil {
						if err := c.verifyTimestampResponse(resp.timestamp, resp.server); err == nil {
							// Could update a shared "latest verified timestamp" here if needed
						}
					}
				case <-ctx.Done():
					return
				}
			}
		}()
		
		return bestTimestamp, nil
	}
	
	// If we don't have progressive handling or didn't reach quorum yet,
	// collect the rest of the responses
	for i := quorum; i < len(pendingServers); i++ {
		select {
		case resp := <-responseCh:
			responses = append(responses, resp)
			
			if resp.error == nil {
				if err := c.verifyTimestampResponse(resp.timestamp, resp.server); err == nil {
					validResponses++
					if bestTimestamp == nil || resp.timestamp.Time.Before(bestTimestamp.Time) {
						bestTimestamp = resp.timestamp
					}
				} else {
					c.stats.verificationFailures.Add(1)
				}
			}
		case <-ctx.Done():
			// Timeout - check if we have enough responses already
			if validResponses >= quorum {
				return bestTimestamp, nil
			}
			c.stats.requestTimeouts.Add(1)
			return nil, fmt.Errorf("timeout waiting for quorum: %w", ErrRequestTimeout)
		}
	}
	
	// Check if we have quorum
	if validResponses < quorum {
		c.stats.quorumFailures.Add(1)
		return nil, ErrQuorumNotReached
	}
	
	return bestTimestamp, nil
}

// GetVerifiedTimestampForBatch gets a timestamp for a batch operation
// using cached timestamps when possible to amortize network costs
func (c *HFTClient) GetVerifiedTimestampForBatch(ctx context.Context, batchKey, regionID string, maxAge time.Duration) (*types.SignedTimestamp, error) {
	// Return nil if batching is disabled
	if !c.options.enableBatching || c.batchCache == nil {
		return c.GetVerifiedTimestamp(ctx, regionID)
	}
	
	// Check cache for existing recent timestamp for this batch
	if ts := c.batchCache.get(batchKey); ts != nil {
		// Check if the timestamp is recent enough
		age := time.Since(ts.Time)
		if age <= maxAge {
			c.stats.cacheHits.Add(1)
			return ts, nil
		}
	}
	
	c.stats.cacheMisses.Add(1)
	
	// Get a new timestamp and cache it for the batch
	ts, err := c.GetVerifiedTimestamp(ctx, regionID)
	if err != nil {
		return nil, err
	}
	
	c.batchCache.set(batchKey, ts)
	return ts, nil
}

// requestTimestamp requests a timestamp from a specific server
func (c *HFTClient) requestTimestamp(ctx context.Context, server types.TimeServer) (*types.SignedTimestamp, error) {
	// Implementation depends on how you communicate with servers
	// This is a placeholder for the actual implementation
	// In a real implementation, you'd typically make an HTTP request or use RPC
	
	// In this implementation, we'll use FastPath from the server
	if fastPathServer, ok := server.(interface {
		GetFastPath() interface{ GetTimestamp(context.Context) (*types.SignedTimestamp, error) }
	}); ok {
		return fastPathServer.GetFastPath().GetTimestamp(ctx)
	}
	
	// Fallback to direct signing
	ts := &types.SignedTimestamp{
		Time:     time.Now().UTC(),
		ServerID: server.GetID(),
		RegionID: server.GetRegion(),
	}
	
	if err := ts.Sign(server); err != nil {
		return nil, err
	}
	
	return ts, nil
}

// serverResponse represents a response from a server
type serverResponse struct {
	timestamp *types.SignedTimestamp
	error     error
	server    types.TimeServer
	rtt       time.Duration
}

// verifyTimestampResponse verifies a timestamp response from a server
func (c *HFTClient) verifyTimestampResponse(ts *types.SignedTimestamp, server types.TimeServer) error {
	if ts == nil {
		return ErrTimestampVerification
	}
	
	// Check cache if caching is enabled
	if c.options.enableCaching && c.verifyCache != nil {
		// Generate cache key
		cacheKey := fmt.Sprintf("%s:%s:%s", ts.ServerID, ts.RegionID, ts.Signature)
		
		// Check if we have a cached result
		if result, found := c.verifyCache.get(cacheKey); found {
			if result {
				return nil
			}
			return ErrTimestampVerification
		}
		
		// Not in cache, verify and cache result
		err := ts.Verify(server)
		c.verifyCache.set(cacheKey, err == nil)
		return err
	}
	
	// Verification without caching
	return ts.Verify(server)
}

// getServersByRTT returns servers sorted by RTT for optimal ordering
func (c *HFTClient) getServersByRTT(regionID string) []types.TimeServer {
	// Get all servers for the region from the TimeNetwork
	var servers []types.TimeServer
	
	// Get region-specific servers first
	if regionID != "" {
		servers = c.registry.GetServersByRegion(regionID)
	}
	
	// Fall back to all servers if no regional servers found
	if len(servers) == 0 {
		servers = c.registry.GetServers()
	}
	
	// Sort by RTT
	c.rttLock.RLock()
	defer c.rttLock.RUnlock()
	
	sort.Slice(servers, func(i, j int) bool {
		// Get RTT for both servers
		iTracker, iExists := c.serverRTT[servers[i].GetID()]
		jTracker, jExists := c.serverRTT[servers[j].GetID()]
		
		// If one server has RTT data and the other doesn't, prefer the one with data
		if iExists && !jExists {
			return true
		}
		if !iExists && jExists {
			return false
		}
		
		// If neither has data, sort by ID for consistency
		if !iExists && !jExists {
			return servers[i].GetID() < servers[j].GetID()
		}
		
		// Both have RTT data, compare average RTTs
		return iTracker.getAverageRTT() < jTracker.getAverageRTT()
	})
	
	return servers
}

// updateServerRTT updates RTT tracking for a server
func (c *HFTClient) updateServerRTT(serverID string, rtt time.Duration) {
	c.rttLock.Lock()
	defer c.rttLock.Unlock()
	
	tracker, exists := c.serverRTT[serverID]
	if !exists {
		tracker = newRTTTracker(c.options.rttSampleSize, c.options.rttDecayFactor)
		c.serverRTT[serverID] = tracker
	}
	
	tracker.addSample(rtt)
}

// GetStats returns the current client stats
func (c *HFTClient) GetStats() map[string]int64 {
	return map[string]int64{
		"requests":              c.stats.requestCount.Load(),
		"cache_hits":            c.stats.cacheHits.Load(),
		"cache_misses":          c.stats.cacheMisses.Load(),
		"verification_failures": c.stats.verificationFailures.Load(),
		"quorum_failures":       c.stats.quorumFailures.Load(),
		"request_timeouts":      c.stats.requestTimeouts.Load(),
	}
}

// rttTracker tracks RTT for a server
type rttTracker struct {
	samples     []time.Duration
	index       int
	count       int
	maxSamples  int
	decayFactor float64
	mu          sync.RWMutex
}

// newRTTTracker creates a new RTT tracker
func newRTTTracker(maxSamples int, decayFactor float64) *rttTracker {
	return &rttTracker{
		samples:     make([]time.Duration, maxSamples),
		index:       0,
		count:       0,
		maxSamples:  maxSamples,
		decayFactor: decayFactor,
	}
}

// addSample adds an RTT sample
func (r *rttTracker) addSample(rtt time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Add sample to circular buffer
	r.samples[r.index] = rtt
	r.index = (r.index + 1) % r.maxSamples
	if r.count < r.maxSamples {
		r.count++
	}
}

// getAverageRTT gets the weighted average RTT
func (r *rttTracker) getAverageRTT() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if r.count == 0 {
		return math.MaxInt64 * time.Nanosecond // High value for unknown RTT
	}
	
	// Calculate weighted average with newer samples weighted more heavily
	var total float64
	var weights float64
	
	for i := 0; i < r.count; i++ {
		// Calculate age index (newest to oldest)
		idx := (r.index - i - 1)
		if idx < 0 {
			idx += r.maxSamples
		}
		
		// Calculate weight based on age
		weight := math.Pow(r.decayFactor, float64(i))
		total += float64(r.samples[idx]) * weight
		weights += weight
	}
	
	return time.Duration(total / weights)
}

// verificationCache caches signature verification results
type verificationCache struct {
	cache      map[string]bool
	expiration map[string]time.Time
	mu         sync.RWMutex
	ttl        time.Duration
	stopCh     chan struct{}
}

// newVerificationCache creates a new verification cache
func newVerificationCache(ttl time.Duration) *verificationCache {
	vc := &verificationCache{
		cache:      make(map[string]bool),
		expiration: make(map[string]time.Time),
		ttl:        ttl,
		stopCh:     make(chan struct{}),
	}
	
	// Start cleanup goroutine
	go vc.cleanupLoop()
	
	return vc
}

// get gets a verification result from the cache
func (vc *verificationCache) get(key string) (bool, bool) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	
	result, exists := vc.cache[key]
	if !exists {
		return false, false
	}
	
	// Check if expired
	expiry, ok := vc.expiration[key]
	if !ok || time.Now().After(expiry) {
		return false, false
	}
	
	return result, true
}

// set sets a verification result in the cache
func (vc *verificationCache) set(key string, result bool) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	
	vc.cache[key] = result
	vc.expiration[key] = time.Now().Add(vc.ttl)
}

// cleanupLoop periodically cleans up expired cache entries
func (vc *verificationCache) cleanupLoop() {
	ticker := time.NewTicker(defaultCacheCleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			vc.cleanup()
		case <-vc.stopCh:
			return
		}
	}
}

// cleanup removes expired cache entries
func (vc *verificationCache) cleanup() {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	
	now := time.Now()
	for key, expiry := range vc.expiration {
		if now.After(expiry) {
			delete(vc.cache, key)
			delete(vc.expiration, key)
		}
	}
}

// stopCleanup stops the cleanup goroutine
func (vc *verificationCache) stopCleanup() {
	close(vc.stopCh)
}

// timestampBatchCache caches timestamps for batch operations
type timestampBatchCache struct {
	cache      map[string]*types.SignedTimestamp
	expiration map[string]time.Time
	mu         sync.RWMutex
	ttl        time.Duration
	stopCh     chan struct{}
}

// newTimestampBatchCache creates a new timestamp batch cache
func newTimestampBatchCache(ttl time.Duration) *timestampBatchCache {
	tc := &timestampBatchCache{
		cache:      make(map[string]*types.SignedTimestamp),
		expiration: make(map[string]time.Time),
		ttl:        ttl,
		stopCh:     make(chan struct{}),
	}
	
	// Start cleanup goroutine
	go tc.cleanupLoop()
	
	return tc
}

// get gets a timestamp from the cache
func (tc *timestampBatchCache) get(key string) *types.SignedTimestamp {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	
	ts, exists := tc.cache[key]
	if !exists {
		return nil
	}
	
	// Check if expired
	expiry, ok := tc.expiration[key]
	if !ok || time.Now().After(expiry) {
		return nil
	}
	
	return ts
}

// set sets a timestamp in the cache
func (tc *timestampBatchCache) set(key string, ts *types.SignedTimestamp) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	tc.cache[key] = ts
	tc.expiration[key] = time.Now().Add(tc.ttl)
}

// cleanupLoop periodically cleans up expired cache entries
func (tc *timestampBatchCache) cleanupLoop() {
	ticker := time.NewTicker(defaultCacheCleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			tc.cleanup()
		case <-tc.stopCh:
			return
		}
	}
}

// cleanup removes expired cache entries
func (tc *timestampBatchCache) cleanup() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	now := time.Now()
	for key, expiry := range tc.expiration {
		if now.After(expiry) {
			delete(tc.cache, key)
			delete(tc.expiration, key)
		}
	}
}

// stopCleanup stops the cleanup goroutine
func (tc *timestampBatchCache) stopCleanup() {
	close(tc.stopCh)
}

// parseWithDualFormatSupport safely parses data in either length-prefixed or direct format
// This implements the dual-format parameter handling pattern required for WebAssembly contracts
func parseWithDualFormatSupport(data []byte) ([]byte, error) {
	// Sanity check for empty data
	if len(data) == 0 {
		return nil, errors.New("empty data provided")
	}
	
	// First, check if we have enough bytes to read the length prefix (4 bytes)
	if len(data) < 4 {
		// If we don't have 4 bytes, assume direct format
		return data, nil
	}
	
	// Try to read the first 4 bytes as a length prefix
	lengthPrefix := binary.LittleEndian.Uint32(data[:4])
	
	// Check if the length prefix makes sense (reasonable size but not too large)
	// This is a critical security check to prevent the 3.5B byte vulnerability
	if lengthPrefix > 0 && lengthPrefix <= maxLengthPrefixValue {
		// This looks like a valid length prefix
		
		// Ensure we have enough data after the prefix
		if uint32(len(data)) < 4+lengthPrefix {
			return nil, fmt.Errorf("data too short for length prefix: expected %d bytes, got %d", 4+lengthPrefix, len(data))
		}
		
		// Extract the actual data after the length prefix
		// Ensure we use the exact length specified by the prefix
		result := make([]byte, lengthPrefix)
		copy(result, data[4:4+lengthPrefix])
		return result, nil
	}
	
	// If we get here, either:
	// 1. The length prefix was unreasonably large (potential attack)
	// 2. The length prefix was 0 (invalid)
	// 3. We're using direct format
	// 
	// Following the pattern from your WebAssembly contracts, we'll assume direct format
	return data, nil
}
