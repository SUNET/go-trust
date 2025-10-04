package api

import (
	"sync"
	"time"

	"github.com/SUNET/go-trust/pkg/pipeline"
)

// ServerContext holds the shared state for the API server, including the pipeline context.
// It provides thread-safe access to the pipeline context and tracks when it was last processed.
// This struct is used by API handlers to access the current state of Trust Status Lists (TSLs)
// and certificate pools for making trust decisions.
type ServerContext struct {
	mu              sync.RWMutex      // Mutex for thread-safe access
	PipelineContext *pipeline.Context // The current pipeline context with TSLs and certificate pool
	LastProcessed   time.Time         // Timestamp when the pipeline was last processed
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
