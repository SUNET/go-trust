package api

import (
	"sync"
	"time"

	"github.com/SUNET/go-trust/pkg/pipeline"
)

type ServerContext struct {
	mu              sync.RWMutex
	PipelineContext *pipeline.Context
	LastProcessed   time.Time
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
