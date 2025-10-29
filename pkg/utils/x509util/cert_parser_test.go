package x509util

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"testing"
	"time"
)

// generateTestCert creates a self-signed test certificate for testing
func generateTestCert() (*x509.Certificate, []byte, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Test Corp"},
			CommonName:   "test.example.com",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, err
	}

	return cert, certDER, nil
}

func TestParseX5CFromArray_Success(t *testing.T) {
	// Generate test certificate
	_, certDER, err := generateTestCert()
	if err != nil {
		t.Fatalf("Failed to generate test cert: %v", err)
	}

	// Encode as base64
	certB64 := base64.StdEncoding.EncodeToString(certDER)

	// Create input array
	input := []interface{}{certB64}

	// Parse
	certs, err := ParseX5CFromArray(input)
	if err != nil {
		t.Fatalf("ParseX5CFromArray failed: %v", err)
	}

	if len(certs) != 1 {
		t.Errorf("Expected 1 certificate, got %d", len(certs))
	}

	if certs[0].Subject.CommonName != "test.example.com" {
		t.Errorf("Expected CN=test.example.com, got %s", certs[0].Subject.CommonName)
	}
}

func TestParseX5CFromArray_MultipleCerts(t *testing.T) {
	// Generate two test certificates
	_, cert1DER, err := generateTestCert()
	if err != nil {
		t.Fatalf("Failed to generate test cert 1: %v", err)
	}

	_, cert2DER, err := generateTestCert()
	if err != nil {
		t.Fatalf("Failed to generate test cert 2: %v", err)
	}

	// Encode as base64
	cert1B64 := base64.StdEncoding.EncodeToString(cert1DER)
	cert2B64 := base64.StdEncoding.EncodeToString(cert2DER)

	// Create input array
	input := []interface{}{cert1B64, cert2B64}

	// Parse
	certs, err := ParseX5CFromArray(input)
	if err != nil {
		t.Fatalf("ParseX5CFromArray failed: %v", err)
	}

	if len(certs) != 2 {
		t.Errorf("Expected 2 certificates, got %d", len(certs))
	}
}

func TestParseX5CFromArray_InvalidBase64(t *testing.T) {
	input := []interface{}{"not-valid-base64!!!"}

	_, err := ParseX5CFromArray(input)
	if err == nil {
		t.Error("Expected error for invalid base64, got nil")
	}
}

func TestParseX5CFromArray_NotString(t *testing.T) {
	input := []interface{}{12345}

	_, err := ParseX5CFromArray(input)
	if err == nil {
		t.Error("Expected error for non-string item, got nil")
	}
}

func TestParseX5CFromArray_InvalidCertificate(t *testing.T) {
	// Valid base64 but not a valid certificate
	invalidCert := base64.StdEncoding.EncodeToString([]byte("invalid certificate data"))
	input := []interface{}{invalidCert}

	_, err := ParseX5CFromArray(input)
	if err == nil {
		t.Error("Expected error for invalid certificate, got nil")
	}
}

func TestParseX5CFromJWK_Success(t *testing.T) {
	// Generate test certificate
	_, certDER, err := generateTestCert()
	if err != nil {
		t.Fatalf("Failed to generate test cert: %v", err)
	}

	// Encode as base64
	certB64 := base64.StdEncoding.EncodeToString(certDER)

	// Create JWK input
	jwk := map[string]interface{}{
		"kty": "EC",
		"crv": "P-256",
		"x":   "some-x-value",
		"y":   "some-y-value",
		"x5c": []interface{}{certB64},
	}
	input := []interface{}{jwk}

	// Parse
	certs, err := ParseX5CFromJWK(input)
	if err != nil {
		t.Fatalf("ParseX5CFromJWK failed: %v", err)
	}

	if len(certs) != 1 {
		t.Errorf("Expected 1 certificate, got %d", len(certs))
	}

	if certs[0].Subject.CommonName != "test.example.com" {
		t.Errorf("Expected CN=test.example.com, got %s", certs[0].Subject.CommonName)
	}
}

func TestParseX5CFromJWK_EmptyArray(t *testing.T) {
	input := []interface{}{}

	_, err := ParseX5CFromJWK(input)
	if err == nil {
		t.Error("Expected error for empty array, got nil")
	}
}

func TestParseX5CFromJWK_NotJWKObject(t *testing.T) {
	input := []interface{}{"not-a-jwk-object"}

	_, err := ParseX5CFromJWK(input)
	if err == nil {
		t.Error("Expected error for non-JWK object, got nil")
	}
}

func TestParseX5CFromJWK_MissingX5C(t *testing.T) {
	jwk := map[string]interface{}{
		"kty": "EC",
		"crv": "P-256",
		"x":   "some-x-value",
		"y":   "some-y-value",
		// Missing x5c claim
	}
	input := []interface{}{jwk}

	_, err := ParseX5CFromJWK(input)
	if err == nil {
		t.Error("Expected error for missing x5c claim, got nil")
	}
}

func TestParseX5CFromJWK_X5CNotArray(t *testing.T) {
	jwk := map[string]interface{}{
		"kty": "EC",
		"crv": "P-256",
		"x5c": "not-an-array",
	}
	input := []interface{}{jwk}

	_, err := ParseX5CFromJWK(input)
	if err == nil {
		t.Error("Expected error for x5c not being an array, got nil")
	}
}

func TestParseX5CFromJWK_X5CJSONString(t *testing.T) {
	// Generate test certificate
	_, certDER, err := generateTestCert()
	if err != nil {
		t.Fatalf("Failed to generate test cert: %v", err)
	}

	// Encode as base64
	certB64 := base64.StdEncoding.EncodeToString(certDER)

	// Create JWK with x5c as JSON string (some implementations do this)
	jwk := map[string]interface{}{
		"kty": "EC",
		"crv": "P-256",
		"x5c": `["` + certB64 + `"]`, // x5c as JSON string
	}
	input := []interface{}{jwk}

	// Parse
	certs, err := ParseX5CFromJWK(input)
	if err != nil {
		t.Fatalf("ParseX5CFromJWK failed with JSON string x5c: %v", err)
	}

	if len(certs) != 1 {
		t.Errorf("Expected 1 certificate, got %d", len(certs))
	}

	if certs[0].Subject.CommonName != "test.example.com" {
		t.Errorf("Expected CN=test.example.com, got %s", certs[0].Subject.CommonName)
	}
}
