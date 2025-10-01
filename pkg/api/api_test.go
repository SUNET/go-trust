package api

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/SUNET/g119612/pkg/etsi119612"
	"github.com/SUNET/go-trust/pkg/pipeline"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var testCertBase64 string
var testCertDER []byte
var testCert *x509.Certificate

// generateTestCertBase64 runs openssl to generate a self-signed cert and returns the base64-encoded DER string.
func generateTestCertBase64() (string, []byte, *x509.Certificate, error) {
	// Use unique temp files for key, cert, and der
	keyFile, err := os.CreateTemp("", "testkey-*.pem")
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create temp key file: %w", err)
	}
	defer os.Remove(keyFile.Name())
	keyFile.Close()
	certFile, err := os.CreateTemp("", "testcert-*.pem")
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create temp cert file: %w", err)
	}
	defer os.Remove(certFile.Name())
	certFile.Close()
	derFile, err := os.CreateTemp("", "testcert-*.der")
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create temp der file: %w", err)
	}
	defer os.Remove(derFile.Name())
	derFile.Close()

	// Build the openssl command using the temp files
	opensslCmd := fmt.Sprintf("openssl req -x509 -newkey rsa:2048 -keyout %s -out %s -days 365 -nodes -subj '/CN=Test Cert' 2>/dev/null && openssl x509 -outform der -in %s -out %s 2>/dev/null && openssl base64 -in %s -A 2>/dev/null", keyFile.Name(), certFile.Name(), certFile.Name(), derFile.Name(), derFile.Name())
	cmd := exec.Command("bash", "-c", opensslCmd)
	var out bytes.Buffer
	cmd.Stdout = &out
	// Do not capture Stderr, as it is redirected in the shell command
	err = cmd.Run()
	output := out.String()
	if err != nil {
		// Print the OpenSSL output for debugging
		return "", nil, nil, fmt.Errorf("openssl error: %v\noutput: %s", err, output)
	}
	certBase64 := strings.TrimSpace(output)
	certDER, err := base64.StdEncoding.DecodeString(certBase64)
	if err != nil {
		return certBase64, nil, nil, fmt.Errorf("base64 decode error: %v\noutput: %s", err, output)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return certBase64, certDER, nil, fmt.Errorf("parse cert error: %v\noutput: %s", err, output)
	}
	return certBase64, certDER, cert, nil
}

func init() {
	var err error
	testCertBase64, testCertDER, testCert, err = generateTestCertBase64()
	if err != nil {
		panic("failed to generate test cert: " + err.Error())
	}
}

func setupTestServer() (*gin.Engine, *ServerContext) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	// Add the test certificate to the CertPool for x5c validation
	certPool := x509.NewCertPool()
	certPool.AddCert(testCert)
	serverCtx := &ServerContext{
		PipelineContext: &pipeline.Context{
			CertPool: certPool,
		},
		LastProcessed: time.Now(),
	}
	// Store the certBase64 for use in tests
	RegisterAPIRoutes(r, serverCtx)
	return r, serverCtx
}

func TestStatusEndpoint(t *testing.T) {
	r, serverCtx := setupTestServer()
	serverCtx.Lock()
	serverCtx.PipelineContext.TSLs = make([]*etsi119612.TSL, 2) // Simulate 2 TSLs loaded
	serverCtx.Unlock()

	req, _ := http.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "tsl_count")
	assert.Contains(t, w.Body.String(), "last_processed")
}

func TestInfoEndpoint_Empty(t *testing.T) {
	r, _ := setupTestServer()
	req, _ := http.NewRequest("GET", "/info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "tsl_summaries")
}

func TestAuthzenDecisionEndpoint(t *testing.T) {
	r, _ := setupTestServer()
	body := `{
	       "subject": {
		       "type": "user",
		       "id": "alice",
		       "properties": {
			       "x5c": ["` + testCertBase64 + `"]
		       }
	       },
	       "resource": {
		       "type": "document",
		       "id": "doc1",
		       "properties": {}
	       },
	       "action": {
		       "name": "read",
		       "properties": {}
	       },
	       "context": {}
       }`
	req, _ := http.NewRequest("POST", "/authzen/decision", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"decision":true`)
}

func TestStartBackgroundUpdater(t *testing.T) {
	// Register a mock pipeline step that always adds a known value
	pipeline.RegisterFunction("mockstep", func(pl *pipeline.Pipeline, ctx *pipeline.Context, args ...string) (*pipeline.Context, error) {
		return &pipeline.Context{TSLs: []*etsi119612.TSL{nil}}, nil
	})
	pipes := []pipeline.Pipe{{MethodName: "mockstep", MethodArguments: []string{}}}
	pl := &pipeline.Pipeline{Pipes: pipes}
	serverCtx := &ServerContext{}
	interval := 10 * time.Millisecond
	_ = StartBackgroundUpdater(pl, serverCtx, interval)

	// Wait for the updater to run at least once
	time.Sleep(30 * time.Millisecond)

	serverCtx.RLock()
	defer serverCtx.RUnlock()
	if serverCtx.PipelineContext == nil || len(serverCtx.PipelineContext.TSLs) != 1 {
		t.Errorf("ServerContext was not updated by StartBackgroundUpdater")
	}
}

func TestBuildResponse(t *testing.T) {
	// Decision true: should return true and nil context
	resp := buildResponse(true, "")
	if !resp.Decision {
		t.Errorf("Expected Decision true, got false")
	}
	if resp.Context != nil {
		t.Errorf("Expected nil Context for true decision")
	}

	// Decision false: should return false and context with reason
	reason := "some error"
	resp = buildResponse(false, reason)
	if resp.Decision {
		t.Errorf("Expected Decision false, got true")
	}
	if resp.Context == nil {
		t.Errorf("Expected non-nil Context for false decision")
	} else {
		// Check that ReasonAdmin contains the error
		admin, ok := resp.Context.ReasonAdmin["error"]
		if !ok || admin != reason {
			t.Errorf("Expected ReasonAdmin to contain error '%s', got '%v'", reason, admin)
		}
	}

	// Decision false, empty reason
	resp = buildResponse(false, "")
	if resp.Decision {
		t.Errorf("Expected Decision false, got true")
	}
	if resp.Context == nil {
		t.Errorf("Expected non-nil Context for false decision with empty reason")
	}
}

func TestParseX5C_Errors(t *testing.T) {
	// Invalid base64
	props := map[string]interface{}{"x5c": []interface{}{">>notbase64<<"}}
	certs, err := parseX5C(props)
	if err == nil || certs != nil {
		t.Errorf("Expected error for invalid base64, got: %v", err)
	}

	// Malformed certificate (valid base64, but not a cert)
	props = map[string]interface{}{"x5c": []interface{}{base64.StdEncoding.EncodeToString([]byte("notacert"))}}
	certs, err = parseX5C(props)
	if err == nil || certs != nil {
		t.Errorf("Expected error for malformed cert, got: %v", err)
	}

	// x5c property is not a list
	props = map[string]interface{}{"x5c": "notalist"}
	certs, err = parseX5C(props)
	if err == nil || certs != nil {
		t.Errorf("Expected error for non-list x5c, got: %v", err)
	}

	// x5c entry is not a string
	props = map[string]interface{}{"x5c": []interface{}{1234}}
	certs, err = parseX5C(props)
	if err == nil || certs != nil {
		t.Errorf("Expected error for non-string x5c entry, got: %v", err)
	}

	// Nil props and missing x5c should not error, should return empty slice
	certs, err = parseX5C(nil)
	if err != nil || len(certs) != 0 {
		t.Errorf("Expected empty result for nil props, got: %v, %v", certs, err)
	}
	certs, err = parseX5C(map[string]interface{}{})
	if err != nil || len(certs) != 0 {
		t.Errorf("Expected empty result for missing x5c, got: %v, %v", certs, err)
	}
}
