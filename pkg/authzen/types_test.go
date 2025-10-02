package authzen

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEntitySerialization(t *testing.T) {
	tests := []struct {
		name     string
		entity   Entity
		expected string
		wantErr  bool
	}{
		{
			name: "basic entity",
			entity: Entity{
				Type: "user",
				ID:   "12345",
			},
			expected: `{"type":"user","id":"12345"}`,
			wantErr:  false,
		},
		{
			name: "entity with properties",
			entity: Entity{
				Type: "certificate",
				ID:   "cert123",
				Properties: map[string]interface{}{
					"issuer": "CN=Test CA",
					"valid":  true,
				},
			},
			expected: `{"type":"certificate","id":"cert123","properties":{"issuer":"CN=Test CA","valid":true}}`,
			wantErr:  false,
		},
		{
			name: "entity with complex properties",
			entity: Entity{
				Type: "document",
				ID:   "doc456",
				Properties: map[string]interface{}{
					"created":   "2025-10-02T12:00:00Z",
					"signature": map[string]interface{}{"algorithm": "RSA", "valid": true},
				},
			},
			expected: `{"type":"document","id":"doc456","properties":{"created":"2025-10-02T12:00:00Z","signature":{"algorithm":"RSA","valid":true}}}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal the entity
			data, err := json.Marshal(tt.entity)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))

			// Unmarshal back to verify roundtrip
			var decoded Entity
			err = json.Unmarshal(data, &decoded)
			assert.NoError(t, err)
			assert.Equal(t, tt.entity.Type, decoded.Type)
			assert.Equal(t, tt.entity.ID, decoded.ID)

			// For properties, we need to compare based on the specific test
			if tt.entity.Properties != nil {
				assert.Equal(t, len(tt.entity.Properties), len(decoded.Properties))
				for k, v := range tt.entity.Properties {
					assert.Contains(t, decoded.Properties, k)
					assert.Equal(t, v, decoded.Properties[k])
				}
			} else {
				assert.Nil(t, decoded.Properties)
			}
		})
	}
}

func TestEvaluationRequestSerialization(t *testing.T) {
	tests := []struct {
		name     string
		request  EvaluationRequest
		expected string
		wantErr  bool
	}{
		{
			name: "basic request",
			request: EvaluationRequest{
				Subject: Entity{
					Type: "user",
					ID:   "user123",
				},
				Resource: Entity{
					Type: "document",
					ID:   "doc456",
				},
				Action: struct {
					Name       string                 `json:"name"`
					Properties map[string]interface{} `json:"properties,omitempty"`
				}{
					Name: "read",
				},
			},
			expected: `{
				"subject": {"type":"user","id":"user123"},
				"resource": {"type":"document","id":"doc456"},
				"action": {"name":"read"}
			}`,
			wantErr: false,
		},
		{
			name: "request with properties and context",
			request: EvaluationRequest{
				Subject: Entity{
					Type: "user",
					ID:   "user456",
					Properties: map[string]interface{}{
						"roles": []string{"admin", "editor"},
					},
				},
				Resource: Entity{
					Type: "certificate",
					ID:   "cert789",
					Properties: map[string]interface{}{
						"issuer": "CN=Trusted CA",
						"valid":  true,
					},
				},
				Action: struct {
					Name       string                 `json:"name"`
					Properties map[string]interface{} `json:"properties,omitempty"`
				}{
					Name: "verify",
					Properties: map[string]interface{}{
						"timestamp": "2025-10-02T12:00:00Z",
					},
				},
				Context: map[string]interface{}{
					"ip_address": "192.168.1.1",
					"timestamp":  "2025-10-02T12:01:00Z",
				},
			},
			expected: `{
				"subject": {
					"type": "user",
					"id": "user456",
					"properties": {
						"roles": ["admin", "editor"]
					}
				},
				"resource": {
					"type": "certificate",
					"id": "cert789",
					"properties": {
						"issuer": "CN=Trusted CA",
						"valid": true
					}
				},
				"action": {
					"name": "verify",
					"properties": {
						"timestamp": "2025-10-02T12:00:00Z"
					}
				},
				"context": {
					"ip_address": "192.168.1.1",
					"timestamp": "2025-10-02T12:01:00Z"
				}
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal the request
			data, err := json.Marshal(tt.request)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))

			// Unmarshal back to verify roundtrip
			var decoded EvaluationRequest
			err = json.Unmarshal(data, &decoded)
			assert.NoError(t, err)

			// Check subject
			assert.Equal(t, tt.request.Subject.Type, decoded.Subject.Type)
			assert.Equal(t, tt.request.Subject.ID, decoded.Subject.ID)

			// Check resource
			assert.Equal(t, tt.request.Resource.Type, decoded.Resource.Type)
			assert.Equal(t, tt.request.Resource.ID, decoded.Resource.ID)

			// Check action
			assert.Equal(t, tt.request.Action.Name, decoded.Action.Name)

			// For context and properties, we need deeper comparison
			if tt.request.Context != nil {
				assert.Equal(t, len(tt.request.Context), len(decoded.Context))
			}
		})
	}
}

func TestEvaluationResponseSerialization(t *testing.T) {
	tests := []struct {
		name     string
		response EvaluationResponse
		expected string
		wantErr  bool
	}{
		{
			name: "simple allow response",
			response: EvaluationResponse{
				Decision: true,
			},
			expected: `{"decision":true}`,
			wantErr:  false,
		},
		{
			name: "simple deny response",
			response: EvaluationResponse{
				Decision: false,
			},
			expected: `{"decision":false}`,
			wantErr:  false,
		},
		{
			name: "response with context",
			response: EvaluationResponse{
				Decision: true,
				Context: &struct {
					ID          string                 `json:"id"`
					ReasonAdmin map[string]interface{} `json:"reason_admin,omitempty"`
					ReasonUser  map[string]interface{} `json:"reason_user,omitempty"`
				}{
					ID: "decision-123",
					ReasonAdmin: map[string]interface{}{
						"rule":      "certificate-validation",
						"timestamp": "2025-10-02T12:00:00Z",
					},
					ReasonUser: map[string]interface{}{
						"message": "Certificate is valid and trusted",
					},
				},
			},
			expected: `{
				"decision": true,
				"context": {
					"id": "decision-123",
					"reason_admin": {
						"rule": "certificate-validation",
						"timestamp": "2025-10-02T12:00:00Z"
					},
					"reason_user": {
						"message": "Certificate is valid and trusted"
					}
				}
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal the response
			data, err := json.Marshal(tt.response)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))

			// Unmarshal back to verify roundtrip
			var decoded EvaluationResponse
			err = json.Unmarshal(data, &decoded)
			assert.NoError(t, err)
			assert.Equal(t, tt.response.Decision, decoded.Decision)

			// Check context if it exists
			if tt.response.Context != nil {
				assert.NotNil(t, decoded.Context)
				assert.Equal(t, tt.response.Context.ID, decoded.Context.ID)

				// Check reason_admin
				if tt.response.Context.ReasonAdmin != nil {
					assert.Equal(t, len(tt.response.Context.ReasonAdmin), len(decoded.Context.ReasonAdmin))
				} else {
					assert.Nil(t, decoded.Context.ReasonAdmin)
				}

				// Check reason_user
				if tt.response.Context.ReasonUser != nil {
					assert.Equal(t, len(tt.response.Context.ReasonUser), len(decoded.Context.ReasonUser))
				} else {
					assert.Nil(t, decoded.Context.ReasonUser)
				}
			} else {
				assert.Nil(t, decoded.Context)
			}
		})
	}
}

func TestX509CertificateHandling(t *testing.T) {
	// Test case for a certificate entity with X509 properties
	certEntity := Entity{
		Type: "certificate",
		ID:   "x509-cert-1",
		Properties: map[string]interface{}{
			"subject":     "CN=Test Subject,O=Test Org",
			"issuer":      "CN=Test CA,O=Test Authority",
			"notBefore":   "2025-01-01T00:00:00Z",
			"notAfter":    "2026-01-01T00:00:00Z",
			"fingerprint": "01:23:45:67:89:AB:CD:EF:01:23:45:67:89:AB:CD:EF",
			"keyUsage":    []string{"digitalSignature", "keyEncipherment"},
		},
	}

	// Test serialization of certificate entity
	data, err := json.Marshal(certEntity)
	assert.NoError(t, err)

	var decodedCert Entity
	err = json.Unmarshal(data, &decodedCert)
	assert.NoError(t, err)

	// Check basic properties
	assert.Equal(t, "certificate", decodedCert.Type)
	assert.Equal(t, "x509-cert-1", decodedCert.ID)

	// Check certificate-specific properties
	assert.Equal(t, "CN=Test Subject,O=Test Org", decodedCert.Properties["subject"])
	assert.Equal(t, "CN=Test CA,O=Test Authority", decodedCert.Properties["issuer"])

	// Check date properties
	assert.Equal(t, "2025-01-01T00:00:00Z", decodedCert.Properties["notBefore"])
	assert.Equal(t, "2026-01-01T00:00:00Z", decodedCert.Properties["notAfter"])

	// Create a certificate validation test case
	validationTest := EvaluationRequest{
		Subject: Entity{
			Type: "user",
			ID:   "user-1",
		},
		Resource: certEntity,
		Action: struct {
			Name       string                 `json:"name"`
			Properties map[string]interface{} `json:"properties,omitempty"`
		}{
			Name: "validate",
			Properties: map[string]interface{}{
				"timestamp": time.Now().Format(time.RFC3339),
			},
		},
	}

	// Test serialization of validation request
	reqData, err := json.Marshal(validationTest)
	assert.NoError(t, err)

	var decodedReq EvaluationRequest
	err = json.Unmarshal(reqData, &decodedReq)
	assert.NoError(t, err)

	// Check that the certificate data is preserved
	assert.Equal(t, certEntity.Type, decodedReq.Resource.Type)
	assert.Equal(t, certEntity.ID, decodedReq.Resource.ID)
	assert.Equal(t, certEntity.Properties["subject"], decodedReq.Resource.Properties["subject"])
}
