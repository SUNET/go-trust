package dsig

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFileSigner(t *testing.T) {
	// Skip test if we're in CI
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping FileSigner test in CI environment")
	}

	// Check if we can run openssl to generate a test certificate
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("Skipping test: openssl not available")
	}

	// Create temp test directory
	tmpDir, err := os.MkdirTemp("", "dsig-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test certificate and key paths
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	// Generate self-signed certificate using openssl
	cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
		"-keyout", keyPath, "-out", certPath, "-days", "1", "-nodes",
		"-subj", "/CN=Test Certificate")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("Failed to generate test certificate: %v, output: %s", err, output)
		return
	}

	// Create FileSigner
	signer := NewFileSigner(certPath, keyPath)

	// Test XML data
	xmlData := []byte(`<test>Test XML for signing</test>`)

	// Sign the XML
	signedData, err := signer.Sign(xmlData)
	if err != nil {
		t.Fatalf("Signing failed: %v", err)
	}

	// Verify that signature was added
	if len(signedData) <= len(xmlData) {
		t.Fatal("Signed data should be longer than original")
	}
}
