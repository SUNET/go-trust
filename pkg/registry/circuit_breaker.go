package registry

import (
	"sync"
	"time"
)

// CircuitState represents the current state of a circuit breaker
type CircuitState string

const (
	// CircuitClosed means normal operation - requests are allowed
	CircuitClosed CircuitState = "closed"

	// CircuitOpen means failures exceeded threshold - requests are rejected
	CircuitOpen CircuitState = "open"

	// CircuitHalfOpen means testing if service recovered - limited requests allowed
	CircuitHalfOpen CircuitState = "half_open"
)

// CircuitBreaker implements the circuit breaker pattern to prevent cascading failures.
//
// When a registry consistently fails, the circuit breaker "opens" and stops sending
// requests to it for a period of time. After the reset timeout, it enters "half-open"
// state to test if the registry has recovered.
//
// This prevents wasting time on known-failing registries and allows graceful degradation.
type CircuitBreaker struct {
	maxFailures  int           // Number of failures before opening circuit
	resetTimeout time.Duration // How long to wait before trying again
	failures     int           // Current failure count
	lastFailure  time.Time     // Time of last failure
	state        CircuitState  // Current circuit state
	mu           sync.RWMutex  // Protects mutable fields
}

// NewCircuitBreaker creates a new CircuitBreaker with the specified parameters.
//
// maxFailures: Number of consecutive failures before opening the circuit
// resetTimeout: Duration to wait in open state before entering half-open
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        CircuitClosed,
		failures:     0,
	}
}

// CanAttempt returns true if a request should be attempted to the registry.
// Returns false if the circuit is open and the reset timeout hasn't expired.
func (cb *CircuitBreaker) CanAttempt() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			return true // Will transition to half-open on next attempt
		}
		return false

	case CircuitHalfOpen:
		return true

	default:
		return false
	}
}

// RecordSuccess records a successful request. Resets the failure count
// and closes the circuit if it was open or half-open.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = CircuitClosed
}

// RecordFailure records a failed request. Increments the failure count
// and opens the circuit if the threshold is exceeded.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	// Transition from half-open to open on any failure
	if cb.state == CircuitHalfOpen {
		cb.state = CircuitOpen
		return
	}

	// Open circuit if failures exceed threshold
	if cb.failures >= cb.maxFailures {
		cb.state = CircuitOpen
	}
}

// GetState returns the current circuit state (for monitoring/debugging)
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetFailureCount returns the current failure count (for monitoring/debugging)
func (cb *CircuitBreaker) GetFailureCount() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}

// Reset manually resets the circuit breaker to closed state with zero failures
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = CircuitClosed
}
