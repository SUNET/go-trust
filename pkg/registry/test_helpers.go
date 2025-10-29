package registry

import (
	"context"

	"github.com/SUNET/go-trust/pkg/authzen"
)

// MockRegistry is a test helper that implements TrustRegistry
type MockRegistry struct {
	name     string
	decision bool
	types    []string
	err      error
}

// Name returns the mock registry name
func (m *MockRegistry) Name() string {
	return m.name
}

// Info returns mock registry information
func (m *MockRegistry) Info() RegistryInfo {
	return RegistryInfo{
		Name:        m.name,
		Type:        "mock",
		Description: "Mock registry for testing",
		Version:     "1.0.0",
	}
}

// SupportedResourceTypes returns the resource types this mock supports
func (m *MockRegistry) SupportedResourceTypes() []string {
	return m.types
}

// Evaluate returns the configured decision or error
func (m *MockRegistry) Evaluate(ctx context.Context, req *authzen.EvaluationRequest) (*authzen.EvaluationResponse, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &authzen.EvaluationResponse{
		Decision: m.decision,
		Context: &authzen.EvaluationResponseContext{
			Reason: map[string]interface{}{
				"registry": m.name,
			},
		},
	}, nil
}

// Healthy always returns true for mock registries
func (m *MockRegistry) Healthy() bool {
	return m.err == nil
}

// Refresh is a no-op for mock registries
func (m *MockRegistry) Refresh(ctx context.Context) error {
	return nil
}

// mockRegistryName returns a consistent name for test registries
func mockRegistryName(i int) string {
	names := []string{"registry-0", "registry-1", "registry-2", "registry-3", "registry-4", "registry-5"}
	if i < len(names) {
		return names[i]
	}
	return "registry-unknown"
}

// createTestRequest creates a standard test request
func createTestRequest() *authzen.EvaluationRequest {
	return &authzen.EvaluationRequest{
		Subject: authzen.Subject{
			Type: "key",
			ID:   "test-subject",
		},
		Resource: authzen.Resource{
			Type: "x5c",
			ID:   "test-subject",
			Key:  []interface{}{"dummy-cert"},
		},
	}
}
