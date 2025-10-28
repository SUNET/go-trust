package api

import (
	"sync"
	"time"

	"github.com/SUNET/go-trust/pkg/logging"
	"github.com/SUNET/go-trust/pkg/pipeline"
)

// ServerContext holds the shared state for the API server, including the pipeline context.
// It provides thread-safe access to the pipeline context and tracks when it was last processed.
// This struct is used by API handlers to access the current state of Trust Status Lists (TSLs)
// and certificate pools for making trust decisions.
//
// The ServerContext always has a configured Logger for API operations. If none is provided
// during initialization, a default logger is used.
type ServerContext struct {
	mu              sync.RWMutex      // Mutex for thread-safe access
	PipelineContext *pipeline.Context // The current pipeline context with TSLs and certificate pool
	LastProcessed   time.Time         // Timestamp when the pipeline was last processed
	Logger          logging.Logger    // Logger for API operations (never nil)
	RateLimiter     *RateLimiter      // Rate limiter for API endpoints (optional)
	Metrics         *Metrics          // Prometheus metrics (optional)
	BaseURL         string            // Base URL for the PDP (e.g., "https://pdp.example.com") for .well-known discovery
}

// Lock locks the ServerContext for writing.
func (s *ServerContext) Lock() {
	s.mu.Lock()
}

// Unlock unlocks the ServerContext after writing.
func (s *ServerContext) Unlock() {
	s.mu.Unlock()
}

// RLock locks the ServerContext for reading.
func (s *ServerContext) RLock() {
	s.mu.RLock()
}

// RUnlock unlocks the ServerContext after reading.
func (s *ServerContext) RUnlock() {
	s.mu.RUnlock()
}

// WithLogger returns a copy of the ServerContext with the specified logger.
// This allows for easy reconfiguration of the logger while preserving
// the rest of the ServerContext's state.
//
// Parameters:
//   - logger: The new logger to use for the ServerContext
//
// Returns:
//   - A new ServerContext instance with the same state but using the specified logger
func (s *ServerContext) WithLogger(logger logging.Logger) *ServerContext {
	// Always ensure a valid logger
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	s.RLock()
	defer s.RUnlock()

	return &ServerContext{
		PipelineContext: s.PipelineContext,
		LastProcessed:   s.LastProcessed,
		Logger:          logger,
	}
}
