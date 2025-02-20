package metrics

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server represents a metrics server
type Server struct {
	server *http.Server
}

// NewServer creates a new metrics server
func NewServer(port int) *Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	return &Server{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}
}

// Start starts the metrics server
func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

// Stop stops the metrics server
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
