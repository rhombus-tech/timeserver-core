package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rhombus-tech/timeserver-core/core/metrics"
)

// Server represents the REST API server
type Server struct {
	router *mux.Router
	ts     TimestampService
}

// TimestampService defines the interface for timestamp operations
type TimestampService interface {
	GetTimestamp() (int64, []SignatureInfo, string, error)
	VerifyTimestamp(timestamp int64, signatures []SignatureInfo) (bool, string, error)
	GetValidators() ([]ValidatorInfo, error)
	GetStatus() (*StatusInfo, error)
}

// SignatureInfo contains validator signature information
type SignatureInfo struct {
	ValidatorID string `json:"validator_id"`
	Signature   []byte `json:"signature"`
	RegionID    string `json:"region_id"`
}

// ValidatorInfo contains information about a validator
type ValidatorInfo struct {
	ID     string `json:"id"`
	Region string `json:"region"`
	Status string `json:"status"`
}

// StatusInfo contains node status information
type StatusInfo struct {
	Status      string `json:"status"`
	Region      string `json:"region"`
	PeerCount   int32  `json:"peer_count"`
	LatestRound int64  `json:"latest_round"`
}

// NewServer creates a new REST API server
func NewServer(ts TimestampService) *Server {
	s := &Server{
		router: mux.NewRouter(),
		ts:     ts,
	}
	s.routes()
	return s
}

// metricsMiddleware wraps an http.Handler and records metrics about the request
func metricsMiddleware(path string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Call the next handler
		next.ServeHTTP(w, r)
		
		// Record metrics after the handler returns
		duration := time.Since(start)
		metrics.RecordAPILatency(duration.Seconds(), path)
		metrics.RecordAPIRequest(path)
	})
}

// routes sets up the HTTP routes
func (s *Server) routes() {
	// Add metrics middleware to each route
	s.router.Handle("/v1/timestamp", metricsMiddleware("/v1/timestamp",
		http.HandlerFunc(s.handleGetTimestamp()))).Methods("POST")
	s.router.Handle("/v1/timestamp/verify", metricsMiddleware("/v1/timestamp/verify",
		http.HandlerFunc(s.handleVerifyTimestamp()))).Methods("POST")
	s.router.Handle("/v1/validators", metricsMiddleware("/v1/validators",
		http.HandlerFunc(s.handleGetValidators()))).Methods("GET")
	s.router.Handle("/v1/status", metricsMiddleware("/v1/status",
		http.HandlerFunc(s.handleGetStatus()))).Methods("GET")
}

func (s *Server) handleGetTimestamp() http.HandlerFunc {
	type response struct {
		Timestamp  int64          `json:"timestamp"`
		Signatures []SignatureInfo `json:"signatures"`
		Region     string         `json:"region"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		timestamp, sigs, region, err := s.ts.GetTimestamp()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := response{
			Timestamp:  timestamp,
			Signatures: sigs,
			Region:     region,
		}

		json.NewEncoder(w).Encode(resp)
	}
}

func (s *Server) handleVerifyTimestamp() http.HandlerFunc {
	type request struct {
		Timestamp  int64          `json:"timestamp"`
		Signatures []SignatureInfo `json:"signatures"`
	}

	type response struct {
		Valid bool   `json:"valid"`
		Error string `json:"error,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		valid, errMsg, err := s.ts.VerifyTimestamp(req.Timestamp, req.Signatures)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := response{
			Valid: valid,
			Error: errMsg,
		}

		json.NewEncoder(w).Encode(resp)
	}
}

func (s *Server) handleGetValidators() http.HandlerFunc {
	type response struct {
		Validators []ValidatorInfo `json:"validators"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		validators, err := s.ts.GetValidators()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := response{
			Validators: validators,
		}

		json.NewEncoder(w).Encode(resp)
	}
}

func (s *Server) handleGetStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := s.ts.GetStatus()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(status)
	}
}

// ServeHTTP implements the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
