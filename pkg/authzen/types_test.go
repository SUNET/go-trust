package authzen

import (
	"encoding/json"
	"testing"
)

// TestEvaluationRequestValidation tests the Validate() method
func TestEvaluationRequestValidation(t *testing.T) {
	tests := []struct {
		name      string
		request   EvaluationRequest
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid x5c request",
			request: EvaluationRequest{
				Subject:  Subject{Type: "key", ID: "did:example:123"},
				Resource: Resource{Type: "x5c", ID: "did:example:123", Key: []interface{}{"certbase64"}},
			},
			wantError: false,
		},
		{
			name: "invalid subject type",
			request: EvaluationRequest{
				Subject:  Subject{Type: "user", ID: "alice"},
				Resource: Resource{Type: "x5c", ID: "alice", Key: []interface{}{"cert"}},
			},
			wantError: true,
			errorMsg:  "subject.type must be 'key'",
		},
		{
			name: "resource.id does not match subject.id",
			request: EvaluationRequest{
				Subject:  Subject{Type: "key", ID: "alice"},
				Resource: Resource{Type: "x5c", ID: "bob", Key: []interface{}{"cert"}},
			},
			wantError: true,
			errorMsg:  "resource.id (bob) must match subject.id (alice)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantError && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestEvaluationRequestSerialization tests JSON marshaling
func TestEvaluationRequestSerialization(t *testing.T) {
	request := EvaluationRequest{
		Subject:  Subject{Type: "key", ID: "did:example:test"},
		Resource: Resource{Type: "x5c", ID: "did:example:test", Key: []interface{}{"certdata"}},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded EvaluationRequest
	err = json.Unmarshal(jsonData, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
}
