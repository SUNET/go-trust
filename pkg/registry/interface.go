// Package registry provides the TrustRegistry interface and RegistryManager
// for coordinating multiple trust resolution backends.
//
// The registry abstraction allows go-trust to support multiple trust frameworks
// simultaneously (ETSI 119 612, OpenID Federation, DID methods, etc.) and query
// them in parallel for optimal performance.
//
// Core components:
//   - interface.go: TrustRegistry interface and type definitions
//   - manager.go: RegistryManager coordinating multiple registries
//   - strategies.go: Resolution strategy implementations (FirstMatch, AllRegistries, etc.)
//   - circuit_breaker.go: Circuit breaker for handling registry failures
package registry

import (
	"context"

	"github.com/SUNET/go-trust/pkg/authzen"
)

// TrustRegistry represents a trust resolution backend that can evaluate
// AuthZEN trust evaluation requests.
//
// Implementations might include:
//   - ETSI 119 612 Trust Status Lists
//   - OpenID Federation entity resolution
//   - DID method resolvers
//   - Custom enterprise trust registries
type TrustRegistry interface {
	// Evaluate performs trust evaluation for the given AuthZEN request.
	// Returns an EvaluationResponse with decision=true if the binding is trusted,
	// decision=false otherwise. Should not return an error for "not found" cases;
	// instead return decision=false with appropriate context.
	Evaluate(ctx context.Context, req *authzen.EvaluationRequest) (*authzen.EvaluationResponse, error)

	// SupportedResourceTypes returns the resource.type values this registry
	// can handle. Use "*" to indicate support for all types.
	// Examples: ["x5c", "jwk"], ["entity_configuration"], ["did:web"]
	SupportedResourceTypes() []string

	// Info returns metadata about this registry instance
	Info() RegistryInfo

	// Healthy returns true if the registry is operational and can serve requests.
	// This is used for health checks and circuit breaker decisions.
	Healthy() bool

	// Refresh triggers an update of cached data (e.g., fetch new TSLs, refresh
	// trust chains). Returns error if refresh fails, but registry may still be
	// operational with stale data.
	Refresh(ctx context.Context) error
}

// RegistryInfo provides metadata about a TrustRegistry instance
type RegistryInfo struct {
	Name         string   // Human-readable name, e.g. "ETSI TSL Registry"
	Type         string   // Registry type identifier, e.g. "etsi_tsl", "openid_federation"
	Description  string   // Description of what this registry provides
	Version      string   // Implementation version
	TrustAnchors []string // List of trust anchor identifiers (TSL URLs, federation roots, etc.)
}

// ResolutionStrategy defines how RegistryManager aggregates results from multiple registries
type ResolutionStrategy string

const (
	// FirstMatch returns as soon as any registry returns decision=true (default, fastest)
	// Semantics: OR with fast exit
	FirstMatch ResolutionStrategy = "first_match"

	// AllRegistries queries all applicable registries and aggregates results (for auditing)
	// Semantics: OR with complete result collection
	AllRegistries ResolutionStrategy = "all"

	// BestMatch queries all registries and returns the one with highest confidence
	// Semantics: OR with quality selection
	BestMatch ResolutionStrategy = "best_match"

	// Sequential tries registries in registration order until one succeeds (for rate-limited APIs)
	// Semantics: OR with ordered evaluation
	Sequential ResolutionStrategy = "sequential"
)
