package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Consensus metrics
	ConsensusLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "timeserver_consensus_latency_seconds",
		Help: "Time taken to reach consensus",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1},
	}, []string{"result"})

	ConsensusTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "timeserver_consensus_total",
		Help: "Total number of consensus attempts and results",
	}, []string{"result"})

	// Timestamp metrics
	TimestampLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "timeserver_timestamp_latency_seconds",
		Help: "Latency of timestamp operations",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1},
	}, []string{"operation"})

	TimestampErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "timeserver_timestamp_errors_total",
		Help: "Total number of timestamp operation errors",
	}, []string{"type"})

	// Network health metrics
	PeerCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "timeserver_peer_count",
		Help: "Number of connected peers",
	}, []string{"region"})

	NetworkLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "timeserver_network_latency_seconds",
		Help: "Network latency between peers",
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5},
	}, []string{"peer_id"})

	// P2P metrics
	P2PMessageTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "timeserver_p2p_messages_total",
		Help: "Total number of P2P messages by type",
	}, []string{"type", "direction"})

	P2PDiscoveryEvents = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "timeserver_p2p_discovery_events_total",
		Help: "Total number of peer discovery events",
	}, []string{"event"})

	P2PConnectionState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "timeserver_p2p_connection_state",
		Help: "Current state of P2P connections",
	}, []string{"state"})

	P2PStreamLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "timeserver_p2p_stream_latency_seconds",
		Help: "Latency of P2P stream operations",
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5},
	}, []string{"operation"})

	// API metrics
	apiLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "timeserver_api_latency_seconds",
		Help: "Time taken for API requests",
	}, []string{"path"})

	apiRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "timeserver_api_requests_total",
		Help: "Number of API requests",
	}, []string{"path"})
)

// RecordConsensusLatency records the time taken to reach consensus
func RecordConsensusLatency(duration float64, result string) {
	ConsensusLatency.WithLabelValues(result).Observe(duration)
	ConsensusTotal.WithLabelValues(result).Inc()
}

// RecordTimestampLatency records the latency of timestamp operations
func RecordTimestampLatency(duration float64, operation string) {
	TimestampLatency.WithLabelValues(operation).Observe(duration)
}

// RecordTimestampError increments the error counter for timestamp operations
func RecordTimestampError(errorType string) {
	TimestampErrors.WithLabelValues(errorType).Inc()
}

// UpdatePeerCount updates the gauge for connected peers
func UpdatePeerCount(count int, region string) {
	PeerCount.WithLabelValues(region).Set(float64(count))
}

// RecordNetworkLatency records the network latency to a peer
func RecordNetworkLatency(duration float64, peerID string) {
	NetworkLatency.WithLabelValues(peerID).Observe(duration)
}

// RecordP2PMessage records a P2P message event
func RecordP2PMessage(msgType, direction string) {
	P2PMessageTotal.WithLabelValues(msgType, direction).Inc()
}

// RecordP2PDiscoveryEvent records a peer discovery event
func RecordP2PDiscoveryEvent(event string) {
	P2PDiscoveryEvents.WithLabelValues(event).Inc()
}

// UpdateP2PConnectionState updates the P2P connection state gauge
func UpdateP2PConnectionState(state string, value float64) {
	P2PConnectionState.WithLabelValues(state).Set(value)
}

// RecordP2PStreamLatency records the latency of P2P stream operations
func RecordP2PStreamLatency(duration float64, operation string) {
	P2PStreamLatency.WithLabelValues(operation).Observe(duration)
}

// RecordAPILatency records the time taken for an API request
func RecordAPILatency(duration float64, path string) {
	apiLatency.WithLabelValues(path).Observe(duration)
}

// RecordAPIRequest records an API request
func RecordAPIRequest(path string) {
	apiRequests.WithLabelValues(path).Inc()
}
