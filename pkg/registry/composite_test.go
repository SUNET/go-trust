package registry

import (
	"context"
	"errors"
	"testing"
)

// TestCompositeAND tests the LogicAND operator
func TestCompositeAND(t *testing.T) {
	tests := []struct {
		name              string
		registryDecisions []bool
		expectedDecision  bool
		expectedAgreed    int
	}{
		{
			name:              "all agree",
			registryDecisions: []bool{true, true, true},
			expectedDecision:  true,
			expectedAgreed:    3,
		},
		{
			name:              "one disagrees",
			registryDecisions: []bool{true, false, true},
			expectedDecision:  false,
			expectedAgreed:    2,
		},
		{
			name:              "all disagree",
			registryDecisions: []bool{false, false, false},
			expectedDecision:  false,
			expectedAgreed:    0,
		},
		{
			name:              "single registry agrees",
			registryDecisions: []bool{true},
			expectedDecision:  true,
			expectedAgreed:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock registries
			var registries []TrustRegistry
			for i, decision := range tt.registryDecisions {
				mockReg := &MockRegistry{
					name:     mockRegistryName(i),
					decision: decision,
					types:    []string{"x5c"},
				}
				registries = append(registries, mockReg)
			}

			// Create composite registry with AND logic
			composite := NewCompositeRegistry("test-and", LogicAND, registries...)

			// Evaluate
			req := createTestRequest()
			resp, err := composite.Evaluate(context.Background(), req)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}

			// Check decision
			if resp.Decision != tt.expectedDecision {
				t.Errorf("Decision = %v, want %v", resp.Decision, tt.expectedDecision)
			}

			// Check context
			reason := resp.Context.Reason
			if agreed, ok := reason["agreed_count"].(int); !ok || agreed != tt.expectedAgreed {
				t.Errorf("Agreed count = %v, want %v", agreed, tt.expectedAgreed)
			}

			if operator, ok := reason["operator"].(string); !ok || operator != string(LogicAND) {
				t.Errorf("Operator = %v, want %v", operator, LogicAND)
			}
		})
	}
}

// TestCompositeOR tests the LogicOR operator
func TestCompositeOR(t *testing.T) {
	tests := []struct {
		name              string
		registryDecisions []bool
		expectedDecision  bool
	}{
		{
			name:              "one agrees",
			registryDecisions: []bool{true, false, false},
			expectedDecision:  true,
		},
		{
			name:              "all agree",
			registryDecisions: []bool{true, true, true},
			expectedDecision:  true,
		},
		{
			name:              "all disagree",
			registryDecisions: []bool{false, false, false},
			expectedDecision:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var registries []TrustRegistry
			for i, decision := range tt.registryDecisions {
				registries = append(registries, &MockRegistry{
					name:     mockRegistryName(i),
					decision: decision,
					types:    []string{"x5c"},
				})
			}

			composite := NewCompositeRegistry("test-or", LogicOR, registries...)

			resp, err := composite.Evaluate(context.Background(), createTestRequest())
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}

			if resp.Decision != tt.expectedDecision {
				t.Errorf("Decision = %v, want %v", resp.Decision, tt.expectedDecision)
			}
		})
	}
}

// TestCompositeMAJORITY tests the LogicMAJORITY operator
func TestCompositeMAJORITY(t *testing.T) {
	tests := []struct {
		name              string
		registryDecisions []bool
		expectedDecision  bool
		expectedAgreed    int
	}{
		{
			name:              "clear majority (3 of 5)",
			registryDecisions: []bool{true, true, true, false, false},
			expectedDecision:  true,
			expectedAgreed:    3,
		},
		{
			name:              "bare majority (2 of 3)",
			registryDecisions: []bool{true, true, false},
			expectedDecision:  true,
			expectedAgreed:    2,
		},
		{
			name:              "no majority (2 of 4)",
			registryDecisions: []bool{true, true, false, false},
			expectedDecision:  false,
			expectedAgreed:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var registries []TrustRegistry
			for i, decision := range tt.registryDecisions {
				registries = append(registries, &MockRegistry{
					name:     mockRegistryName(i),
					decision: decision,
					types:    []string{"x5c"},
				})
			}

			composite := NewCompositeRegistry("test-majority", LogicMAJORITY, registries...)

			resp, err := composite.Evaluate(context.Background(), createTestRequest())
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}

			if resp.Decision != tt.expectedDecision {
				t.Errorf("Decision = %v, want %v", resp.Decision, tt.expectedDecision)
			}

			reason := resp.Context.Reason
			if agreed, ok := reason["agreed_count"].(int); !ok || agreed != tt.expectedAgreed {
				t.Errorf("Agreed count = %v, want %v", agreed, tt.expectedAgreed)
			}

			hasMajority, _ := reason["has_majority"].(bool)
			if hasMajority != tt.expectedDecision {
				t.Errorf("Has majority = %v, want %v", hasMajority, tt.expectedDecision)
			}
		})
	}
}

// TestCompositeQUORUM tests the LogicQUORUM operator
func TestCompositeQUORUM(t *testing.T) {
	tests := []struct {
		name              string
		threshold         int
		registryDecisions []bool
		expectedDecision  bool
	}{
		{
			name:              "meets quorum (2 of 3, threshold=2)",
			threshold:         2,
			registryDecisions: []bool{true, true, false},
			expectedDecision:  true,
		},
		{
			name:              "misses quorum (1 of 3, threshold=2)",
			threshold:         2,
			registryDecisions: []bool{true, false, false},
			expectedDecision:  false,
		},
		{
			name:              "exactly meets quorum (3 of 5, threshold=3)",
			threshold:         3,
			registryDecisions: []bool{true, true, true, false, false},
			expectedDecision:  true,
		},
		{
			name:              "exceeds quorum (4 of 5, threshold=2)",
			threshold:         2,
			registryDecisions: []bool{true, true, true, true, false},
			expectedDecision:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var registries []TrustRegistry
			for i, decision := range tt.registryDecisions {
				registries = append(registries, &MockRegistry{
					name:     mockRegistryName(i),
					decision: decision,
					types:    []string{"x5c"},
				})
			}

			composite := NewCompositeRegistry("test-quorum", LogicQUORUM, registries...)
			composite.threshold = tt.threshold

			resp, err := composite.Evaluate(context.Background(), createTestRequest())
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}

			if resp.Decision != tt.expectedDecision {
				t.Errorf("Decision = %v, want %v", resp.Decision, tt.expectedDecision)
			}

			reason := resp.Context.Reason
			if threshold, ok := reason["quorum_threshold"].(int); !ok || threshold != tt.threshold {
				t.Errorf("Quorum threshold = %v, want %v", threshold, tt.threshold)
			}

			meetsQuorum, _ := reason["meets_quorum"].(bool)
			if meetsQuorum != tt.expectedDecision {
				t.Errorf("Meets quorum = %v, want %v", meetsQuorum, tt.expectedDecision)
			}
		})
	}
}

// TestCompositeNesting tests nested composite registries
func TestCompositeNesting(t *testing.T) {
	t.Run("(A OR B) AND C", func(t *testing.T) {
		regA := &MockRegistry{name: "regA", decision: true, types: []string{"x5c"}}
		regB := &MockRegistry{name: "regB", decision: false, types: []string{"x5c"}}
		regC := &MockRegistry{name: "regC", decision: true, types: []string{"x5c"}}

		// Create OR group (A OR B)
		orGroup := NewCompositeRegistry("or-group", LogicOR, regA, regB)

		// Create AND with OR group and C
		composite := NewCompositeRegistry("main", LogicAND, orGroup, regC)

		resp, err := composite.Evaluate(context.Background(), createTestRequest())
		if err != nil {
			t.Fatalf("Evaluate() error = %v", err)
		}

		// Should be true: (true OR false) AND true = true AND true = true
		if !resp.Decision {
			t.Error("Decision should be true: (A OR B) AND C where A=true, B=false, C=true")
		}
	})

	t.Run("(A AND B) OR C", func(t *testing.T) {
		regA := &MockRegistry{name: "regA", decision: true, types: []string{"x5c"}}
		regB := &MockRegistry{name: "regB", decision: false, types: []string{"x5c"}}
		regC := &MockRegistry{name: "regC", decision: true, types: []string{"x5c"}}

		// Create AND group (A AND B)
		andGroup := NewCompositeRegistry("and-group", LogicAND, regA, regB)

		// Create OR with AND group and C
		composite := NewCompositeRegistry("main", LogicOR, andGroup, regC)

		resp, err := composite.Evaluate(context.Background(), createTestRequest())
		if err != nil {
			t.Fatalf("Evaluate() error = %v", err)
		}

		// Should be true: (true AND false) OR true = false OR true = true
		if !resp.Decision {
			t.Error("Decision should be true: (A AND B) OR C where A=true, B=false, C=true")
		}
	})
}

// TestCompositeErrorHandling tests error handling in composite registries
func TestCompositeErrorHandling(t *testing.T) {
	t.Run("registry error counts as disagreement in AND", func(t *testing.T) {
		reg1 := &MockRegistry{name: "reg1", decision: true, types: []string{"x5c"}}
		reg2 := &MockRegistry{name: "reg2", decision: true, types: []string{"x5c"}, err: errors.New("test error")}

		composite := NewCompositeRegistry("test-error", LogicAND, reg1, reg2)

		resp, err := composite.Evaluate(context.Background(), createTestRequest())
		if err != nil {
			t.Fatalf("Evaluate() error = %v", err)
		}

		// Should be false because reg2 errored (counted as disagreement)
		if resp.Decision {
			t.Error("Decision should be false when one registry errors in AND logic")
		}

		reason := resp.Context.Reason
		if errorCount, ok := reason["error_count"].(int); !ok || errorCount != 1 {
			t.Errorf("Error count = %v, want 1", errorCount)
		}
	})

	t.Run("registry error doesn't prevent OR success", func(t *testing.T) {
		reg1 := &MockRegistry{name: "reg1", decision: true, types: []string{"x5c"}}
		reg2 := &MockRegistry{name: "reg2", decision: false, types: []string{"x5c"}, err: errors.New("test error")}

		composite := NewCompositeRegistry("test-error-or", LogicOR, reg1, reg2)

		resp, err := composite.Evaluate(context.Background(), createTestRequest())
		if err != nil {
			t.Fatalf("Evaluate() error = %v", err)
		}

		// Should be true because reg1 succeeded
		if !resp.Decision {
			t.Error("Decision should be true when one registry succeeds in OR logic")
		}
	})
}

// TestCompositeHealthy tests the Healthy method
func TestCompositeHealthy(t *testing.T) {
	t.Run("all healthy", func(t *testing.T) {
		reg1 := &MockRegistry{name: "reg1", decision: true, types: []string{"x5c"}}
		reg2 := &MockRegistry{name: "reg2", decision: true, types: []string{"x5c"}}

		composite := NewCompositeRegistry("test", LogicAND, reg1, reg2)

		if !composite.Healthy() {
			t.Error("Composite should be healthy when all children are healthy")
		}
	})

	t.Run("one unhealthy", func(t *testing.T) {
		reg1 := &MockRegistry{name: "reg1", decision: true, types: []string{"x5c"}}
		reg2 := &MockRegistry{name: "reg2", decision: true, types: []string{"x5c"}, err: errors.New("unhealthy")}

		composite := NewCompositeRegistry("test", LogicAND, reg1, reg2)

		if composite.Healthy() {
			t.Error("Composite should be unhealthy when any child is unhealthy")
		}
	})
}
