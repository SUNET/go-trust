package authzen

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Helper function to create a test X.509 certificate
func createTestCertificate(t *testing.T) (*x509.Certificate, []byte) {
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "Test Certificate",
			Organization: []string{"Test Organization"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// In a real test, we would generate a key pair and sign the certificate
	// For this test, we'll just use a placeholder value for the raw certificate
	certDER := []byte("SAMPLE_CERTIFICATE_DER")

	cert := &x509.Certificate{
		Raw:                   certDER,
		SerialNumber:          template.SerialNumber,
		Subject:               template.Subject,
		NotBefore:             template.NotBefore,
		NotAfter:              template.NotAfter,
		KeyUsage:              template.KeyUsage,
		ExtKeyUsage:           template.ExtKeyUsage,
		BasicConstraintsValid: template.BasicConstraintsValid,
		IsCA:                  false,
		MaxPathLen:            0,
		SubjectKeyId:          []byte{1, 2, 3, 4},
		AuthorityKeyId:        []byte{5, 6, 7, 8},
		OCSPServer:            []string{"http://ocsp.example.com"},
		IssuingCertificateURL: []string{"http://ca.example.com/ca.crt"},
		CRLDistributionPoints: []string{"http://crl.example.com/crl"},
		PolicyIdentifiers:     []asn1.ObjectIdentifier{{1, 2, 3, 4}},
	}

	return cert, certDER
}

// Test for converting X.509 certificate to Entity
func TestX509CertificateToEntity(t *testing.T) {
	cert, _ := createTestCertificate(t)

	// Convert certificate to PEM format
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})

	// Create an Entity from the certificate
	entity := Entity{
		Type: "certificate",
		ID:   cert.SerialNumber.String(),
		Properties: map[string]interface{}{
			"subject":         cert.Subject.String(),
			"issuer":          cert.Issuer.String(),
			"notBefore":       cert.NotBefore.Format(time.RFC3339),
			"notAfter":        cert.NotAfter.Format(time.RFC3339),
			"serialNumber":    cert.SerialNumber.String(),
			"isCA":            cert.IsCA,
			"keyUsage":        int(cert.KeyUsage),
			"pemCertificate":  string(pemData),
			"ocspServer":      cert.OCSPServer,
			"crlDistribution": cert.CRLDistributionPoints,
		},
	}

	// Verify entity properties
	assert.Equal(t, "certificate", entity.Type)
	assert.Equal(t, cert.SerialNumber.String(), entity.ID)
	assert.Equal(t, cert.Subject.String(), entity.Properties["subject"])
	assert.Equal(t, cert.NotBefore.Format(time.RFC3339), entity.Properties["notBefore"])
	assert.Equal(t, cert.NotAfter.Format(time.RFC3339), entity.Properties["notAfter"])
	assert.Equal(t, cert.IsCA, entity.Properties["isCA"])

	// Test certificate in AuthZ request
	request := EvaluationRequest{
		Subject: Entity{
			Type: "user",
			ID:   "user1",
			Properties: map[string]interface{}{
				"roles": []string{"validator"},
			},
		},
		Resource: entity,
		Action: struct {
			Name       string                 `json:"name"`
			Properties map[string]interface{} `json:"properties,omitempty"`
		}{
			Name: "validate",
			Properties: map[string]interface{}{
				"checkOCSP": true,
				"checkCRL":  false,
			},
		},
	}

	// Check request integrity
	assert.Equal(t, "user", request.Subject.Type)
	assert.Equal(t, "certificate", request.Resource.Type)
	assert.Equal(t, "validate", request.Action.Name)
	assert.Equal(t, true, request.Action.Properties["checkOCSP"])
	assert.Equal(t, false, request.Action.Properties["checkCRL"])
	assert.Equal(t, cert.SerialNumber.String(), request.Resource.ID)
	assert.Equal(t, cert.Subject.String(), request.Resource.Properties["subject"])
}

// Test validation of EvaluationRequest
func TestEvaluationRequestValidation(t *testing.T) {
	tests := []struct {
		name      string
		request   EvaluationRequest
		wantValid bool
	}{
		{
			name: "valid request",
			request: EvaluationRequest{
				Subject: Entity{
					Type: "user",
					ID:   "user1",
				},
				Resource: Entity{
					Type: "document",
					ID:   "doc1",
				},
				Action: struct {
					Name       string                 `json:"name"`
					Properties map[string]interface{} `json:"properties,omitempty"`
				}{
					Name: "read",
				},
			},
			wantValid: true,
		},
		{
			name: "missing subject type",
			request: EvaluationRequest{
				Subject: Entity{
					ID: "user1",
				},
				Resource: Entity{
					Type: "document",
					ID:   "doc1",
				},
				Action: struct {
					Name       string                 `json:"name"`
					Properties map[string]interface{} `json:"properties,omitempty"`
				}{
					Name: "read",
				},
			},
			wantValid: false,
		},
		{
			name: "missing subject ID",
			request: EvaluationRequest{
				Subject: Entity{
					Type: "user",
				},
				Resource: Entity{
					Type: "document",
					ID:   "doc1",
				},
				Action: struct {
					Name       string                 `json:"name"`
					Properties map[string]interface{} `json:"properties,omitempty"`
				}{
					Name: "read",
				},
			},
			wantValid: false,
		},
		{
			name: "missing action name",
			request: EvaluationRequest{
				Subject: Entity{
					Type: "user",
					ID:   "user1",
				},
				Resource: Entity{
					Type: "document",
					ID:   "doc1",
				},
				Action: struct {
					Name       string                 `json:"name"`
					Properties map[string]interface{} `json:"properties,omitempty"`
				}{
					// Name is missing
				},
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validateRequest(&tt.request)
			assert.Equal(t, tt.wantValid, valid)
		})
	}
}

// Simple validator function for demonstration
func validateRequest(req *EvaluationRequest) bool {
	if req.Subject.Type == "" || req.Subject.ID == "" {
		return false
	}
	if req.Resource.Type == "" || req.Resource.ID == "" {
		return false
	}
	if req.Action.Name == "" {
		return false
	}
	return true
}

// Test different decision scenarios
func TestDecisionScenarios(t *testing.T) {
	tests := []struct {
		name           string
		request        EvaluationRequest
		expectedResult bool
	}{
		{
			name: "allow valid certificate",
			request: EvaluationRequest{
				Subject: Entity{
					Type: "user",
					ID:   "user1",
				},
				Resource: Entity{
					Type: "certificate",
					ID:   "cert1",
					Properties: map[string]interface{}{
						"valid":     true,
						"notBefore": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
						"notAfter":  time.Now().Add(24 * time.Hour).Format(time.RFC3339),
						"revoked":   false,
					},
				},
				Action: struct {
					Name       string                 `json:"name"`
					Properties map[string]interface{} `json:"properties,omitempty"`
				}{
					Name: "validate",
				},
			},
			expectedResult: true,
		},
		{
			name: "deny expired certificate",
			request: EvaluationRequest{
				Subject: Entity{
					Type: "user",
					ID:   "user1",
				},
				Resource: Entity{
					Type: "certificate",
					ID:   "cert2",
					Properties: map[string]interface{}{
						"valid":     false,
						"notBefore": time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
						"notAfter":  time.Now().Add(-24 * time.Hour).Format(time.RFC3339), // Expired
						"revoked":   false,
					},
				},
				Action: struct {
					Name       string                 `json:"name"`
					Properties map[string]interface{} `json:"properties,omitempty"`
				}{
					Name: "validate",
				},
			},
			expectedResult: false,
		},
		{
			name: "deny revoked certificate",
			request: EvaluationRequest{
				Subject: Entity{
					Type: "user",
					ID:   "user1",
				},
				Resource: Entity{
					Type: "certificate",
					ID:   "cert3",
					Properties: map[string]interface{}{
						"valid":     true,
						"notBefore": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
						"notAfter":  time.Now().Add(24 * time.Hour).Format(time.RFC3339),
						"revoked":   true, // Revoked
					},
				},
				Action: struct {
					Name       string                 `json:"name"`
					Properties map[string]interface{} `json:"properties,omitempty"`
				}{
					Name: "validate",
				},
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// In a real implementation, this would call the decision logic
			// For this test, we're simulating the decision based on the certificate properties
			result := evaluateDecision(&tt.request)
			assert.Equal(t, tt.expectedResult, result)

			// Create a response based on the decision
			response := EvaluationResponse{
				Decision: result,
				Context: &struct {
					ID          string                 `json:"id"`
					ReasonAdmin map[string]interface{} `json:"reason_admin,omitempty"`
					ReasonUser  map[string]interface{} `json:"reason_user,omitempty"`
				}{
					ID: "decision-" + tt.request.Resource.ID,
					ReasonAdmin: map[string]interface{}{
						"certificateValid":   tt.request.Resource.Properties["valid"],
						"certificateRevoked": tt.request.Resource.Properties["revoked"],
					},
				},
			}

			// Check response structure
			assert.Equal(t, result, response.Decision)
			assert.NotNil(t, response.Context)
			assert.Equal(t, "decision-"+tt.request.Resource.ID, response.Context.ID)
		})
	}
}

// Simple decision evaluation for demonstration
func evaluateDecision(req *EvaluationRequest) bool {
	// Only handle certificate validation
	if req.Resource.Type != "certificate" || req.Action.Name != "validate" {
		return false
	}

	// Check if the certificate is valid
	valid, _ := req.Resource.Properties["valid"].(bool)
	if !valid {
		return false
	}

	// Check if the certificate is revoked
	revoked, _ := req.Resource.Properties["revoked"].(bool)
	if revoked {
		return false
	}

	// Check certificate expiration
	notAfterStr, _ := req.Resource.Properties["notAfter"].(string)
	if notAfterStr != "" {
		notAfter, err := time.Parse(time.RFC3339, notAfterStr)
		if err != nil || time.Now().After(notAfter) {
			return false
		}
	}

	// Check certificate validity start
	notBeforeStr, _ := req.Resource.Properties["notBefore"].(string)
	if notBeforeStr != "" {
		notBefore, err := time.Parse(time.RFC3339, notBeforeStr)
		if err != nil || time.Now().Before(notBefore) {
			return false
		}
	}

	return true
}
