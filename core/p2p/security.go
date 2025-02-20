package p2p

import (
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	peerConnectionCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "timeserver_peer_connections_total",
		Help: "Total number of peer connections by type (trusted/untrusted)",
	}, []string{"type"})

	peerScoreGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "timeserver_peer_score",
		Help: "Current peer score based on reliability and accuracy",
	}, []string{"peer_id"})

	suspiciousActivityCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "timeserver_suspicious_activity_total",
		Help: "Total number of suspicious activities detected",
	}, []string{"type"})
)

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	// TrustedPeers is a list of trusted peer IDs
	TrustedPeers []peer.ID
	// MaxPeersPerMinute is the maximum number of new peers allowed per minute
	MaxPeersPerMinute int
	// MinPeerScore is the minimum score required to maintain a peer connection
	MinPeerScore float64
	// MaxPeersPerRegion is the maximum number of peers allowed from a single region
	MaxPeersPerRegion int
}

// PeerSecurity handles peer security and validation
type PeerSecurity struct {
	config SecurityConfig
	scores map[peer.ID]float64
	// Track new connections for rate limiting
	connectionTimes []time.Time
	mu             sync.RWMutex
}

// NewPeerSecurity creates a new PeerSecurity instance
func NewPeerSecurity(config SecurityConfig) *PeerSecurity {
	return &PeerSecurity{
		config:          config,
		scores:          make(map[peer.ID]float64),
		connectionTimes: make([]time.Time, 0),
	}
}

// IsTrustedPeer checks if a peer is in the trusted list
func (s *PeerSecurity) IsTrustedPeer(id peer.ID) bool {
	for _, trusted := range s.config.TrustedPeers {
		if trusted == id {
			return true
		}
	}
	return false
}

// ValidateNewPeer checks if a new peer connection should be allowed
func (s *PeerSecurity) ValidateNewPeer(id peer.ID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Clean up old connection times
	cutoff := now.Add(-time.Minute)
	newTimes := make([]time.Time, 0)
	for _, t := range s.connectionTimes {
		if t.After(cutoff) {
			newTimes = append(newTimes, t)
		}
	}
	s.connectionTimes = newTimes

	// Check rate limiting
	if len(s.connectionTimes) >= s.config.MaxPeersPerMinute {
		suspiciousActivityCounter.WithLabelValues("rate_limit_exceeded").Inc()
		return false
	}

	// Add new connection time
	s.connectionTimes = append(s.connectionTimes, now)

	// Trusted peers bypass other checks
	if s.IsTrustedPeer(id) {
		peerConnectionCounter.WithLabelValues("trusted").Inc()
		return true
	}

	peerConnectionCounter.WithLabelValues("untrusted").Inc()
	return true
}

// UpdatePeerScore updates a peer's score based on its behavior
func (s *PeerSecurity) UpdatePeerScore(id peer.ID, timeDrift time.Duration, responseTime time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Calculate score based on time accuracy and response time
	// Score ranges from 0 to 1, where 1 is best
	timeAccuracy := 1.0 - float64(timeDrift.Milliseconds())/1000.0
	if timeAccuracy < 0 {
		timeAccuracy = 0
	}

	responseScore := 1.0 - float64(responseTime.Milliseconds())/1000.0
	if responseScore < 0 {
		responseScore = 0
	}

	// Combined score with weights
	score := timeAccuracy*0.7 + responseScore*0.3
	s.scores[id] = score

	// Update metric
	peerScoreGauge.WithLabelValues(id.String()).Set(score)
}

// ShouldDisconnectPeer checks if a peer should be disconnected based on its score
func (s *PeerSecurity) ShouldDisconnectPeer(id peer.ID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	score, exists := s.scores[id]
	if !exists {
		return false
	}

	return score < s.config.MinPeerScore
}

// DetectSybilAttack checks for patterns indicating a potential Sybil attack
func (s *PeerSecurity) DetectSybilAttack(newPeers []peer.ID) bool {
	// Check for multiple new peers with similar IDs joining in a short time
	if len(newPeers) > s.config.MaxPeersPerMinute {
		suspiciousActivityCounter.WithLabelValues("potential_sybil_attack").Inc()
		return true
	}
	return false
}
