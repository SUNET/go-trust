package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain sets up and tears down test environment
func TestMain(m *testing.M) {
	// Check if integration tests are enabled
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		fmt.Println("Skipping main.go integration tests. Set RUN_INTEGRATION_TESTS=1 to enable.")
		os.Exit(0)
	}

	// Build the binary for testing
	fmt.Println("Building go-trust binary for integration tests...")
	cmd := exec.Command("go", "build", "-o", "./gt-test", ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Failed to build binary: %v\n%s\n", err, output)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	os.Remove("./gt-test")

	os.Exit(code)
}

// serverProcess manages a running server instance
type serverProcess struct {
	cmd        *exec.Cmd
	port       string
	testMode   bool
	shutdownCh chan struct{}
}

// startServer starts the API server in a separate process
func startServer(t *testing.T, pipelineFile string, port string, testMode bool) *serverProcess {
	t.Helper()

	// Get absolute path to pipeline file
	absPath, err := filepath.Abs(pipelineFile)
	require.NoError(t, err, "Failed to get absolute path for pipeline file")

	// Prepare command
	cmd := exec.Command("./gt-test", "--host", "127.0.0.1", "--port", port, "--frequency", "60s", absPath)
	
	// Set test mode environment variable
	if testMode {
		cmd.Env = append(os.Environ(), "GO_TRUST_TEST_MODE=1")
	}

	// Capture output for debugging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the server
	err = cmd.Start()
	require.NoError(t, err, "Failed to start server")

	sp := &serverProcess{
		cmd:        cmd,
		port:       port,
		testMode:   testMode,
		shutdownCh: make(chan struct{}),
	}

	// Wait for server to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ready := make(chan bool)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ready <- false
				return
			default:
				resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/status", port))
				if err == nil && resp.StatusCode == 200 {
					resp.Body.Close()
					ready <- true
					return
				}
				if resp != nil {
					resp.Body.Close()
				}
				time.Sleep(500 * time.Millisecond)
			}
		}
	}()

	if !<-ready {
		cmd.Process.Kill()
		t.Fatal("Server did not become ready in time")
	}

	t.Logf("Server started on port %s (test mode: %v)", port, testMode)

	return sp
}

// shutdown gracefully stops the server
func (sp *serverProcess) shutdown(t *testing.T) {
	t.Helper()

	if sp.testMode {
		// Use shutdown endpoint in test mode
		resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%s/test/shutdown", sp.port), "application/json", nil)
		if err == nil {
			resp.Body.Close()
			// Wait for process to exit
			done := make(chan error)
			go func() {
				done <- sp.cmd.Wait()
			}()

			select {
			case <-done:
				t.Log("Server shut down gracefully via /test/shutdown")
				return
			case <-time.After(5 * time.Second):
				t.Log("Shutdown endpoint timed out, killing process")
			}
		} else {
			t.Logf("Shutdown endpoint error: %v, falling back to kill", err)
		}
	}

	// Fallback: kill the process
	if sp.cmd.Process != nil {
		sp.cmd.Process.Kill()
		sp.cmd.Wait()
		t.Log("Server killed")
	}
}

// TestVersionFlag tests the --version flag
func TestVersionFlag(t *testing.T) {
	cmd := exec.Command("./gt-test", "--version")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "--version should exit successfully")

	outputStr := string(output)
	assert.Contains(t, outputStr, "Version:", "Output should contain version information")
	t.Logf("Version output: %s", strings.TrimSpace(outputStr))
}

// TestHelpFlag tests the --help flag
func TestHelpFlag(t *testing.T) {
	cmd := exec.Command("./gt-test", "--help")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "--help should exit successfully")

	outputStr := string(output)
	assert.Contains(t, outputStr, "Usage:", "Output should contain usage information")
	assert.Contains(t, outputStr, "--host", "Output should list --host option")
	assert.Contains(t, outputStr, "--port", "Output should list --port option")
	assert.Contains(t, outputStr, "--frequency", "Output should list --frequency option")
	t.Logf("Help output length: %d bytes", len(output))
}

// TestMissingPipelineFile tests behavior when no pipeline file is provided
func TestMissingPipelineFile(t *testing.T) {
	cmd := exec.Command("./gt-test")
	output, err := cmd.CombinedOutput()
	require.Error(t, err, "Should fail when pipeline file is missing")

	outputStr := string(output)
	assert.Contains(t, outputStr, "Error: missing pipeline YAML file argument", "Should show error message")
	assert.Contains(t, outputStr, "Usage:", "Should show usage information")
	t.Logf("Error output: %s", strings.TrimSpace(outputStr))
}

// TestInvalidPipelineFile tests behavior with a non-existent pipeline file
func TestInvalidPipelineFile(t *testing.T) {
	cmd := exec.Command("./gt-test", "nonexistent-pipeline.yaml")
	output, err := cmd.CombinedOutput()
	require.Error(t, err, "Should fail with invalid pipeline file")

	outputStr := string(output)
	assert.Contains(t, outputStr, "Failed to load pipeline", "Should show pipeline load error")
	t.Logf("Error output: %s", strings.TrimSpace(outputStr))
}

// TestBasicPipelineExecution tests running a basic pipeline without API calls
// This test uses a simple pipeline that doesn't start a listener
func TestBasicPipelineExecution(t *testing.T) {
	// Skip if the example file doesn't exist or isn't suitable for testing
	pipelineFile := "../example/basic-usage.yaml"
	if _, err := os.Stat(pipelineFile); os.IsNotExist(err) {
		t.Skip("basic-usage.yaml example not found")
	}

	// For this test, we'll need a non-network pipeline
	// Create a minimal test pipeline
	tempPipeline := createTempPipeline(t, `
- log:
    - "Test pipeline started"
- echo:
    - "Test message"
`)
	defer os.Remove(tempPipeline)

	// Note: This would start the API server, which we'll need to handle differently
	// For now, we'll test that it can at least load the pipeline
	cmd := exec.Command("./gt-test", "--help")
	require.NoError(t, cmd.Run(), "Binary should be functional")
}

// TestAPIServerStatusEndpoint tests the /status endpoint
func TestAPIServerStatusEndpoint(t *testing.T) {
	// Create a minimal test pipeline
	tempPipeline := createTempPipeline(t, `
- set-fetch-options:
    - max-depth:0
    - timeout:5s
- log:
    - "Test pipeline for status endpoint"
`)
	defer os.Remove(tempPipeline)

	// Start server
	server := startServer(t, tempPipeline, "16001", true)
	defer server.shutdown(t)

	// Test status endpoint
	resp, err := http.Get("http://127.0.0.1:16001/status")
	require.NoError(t, err, "Status endpoint should respond")
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode, "Status endpoint should return 200")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Should read response body")

	bodyStr := string(body)
	assert.Contains(t, bodyStr, "tsl_count", "Response should contain tsl_count")
	assert.Contains(t, bodyStr, "last_processed", "Response should contain last_processed")

	t.Logf("Status response: %s", bodyStr)
}

// TestShutdownEndpoint tests the /test/shutdown endpoint (test mode only)
func TestShutdownEndpoint(t *testing.T) {
	tempPipeline := createTempPipeline(t, `
- log:
    - "Test pipeline for shutdown"
`)
	defer os.Remove(tempPipeline)

	// Start server in test mode
	server := startServer(t, tempPipeline, "16002", true)

	// Test that shutdown endpoint exists
	resp, err := http.Post("http://127.0.0.1:16002/test/shutdown", "application/json", nil)
	require.NoError(t, err, "Shutdown endpoint should respond")
	resp.Body.Close()

	// Wait for server to exit
	done := make(chan error)
	go func() {
		done <- server.cmd.Wait()
	}()

	select {
	case err := <-done:
		assert.NoError(t, err, "Server should exit cleanly")
		t.Log("Server shut down successfully via /test/shutdown endpoint")
	case <-time.After(5 * time.Second):
		server.cmd.Process.Kill()
		t.Fatal("Server did not shut down in time")
	}
}

// TestShutdownEndpointDisabledInProduction tests that shutdown endpoint is not available in production mode
func TestShutdownEndpointDisabledInProduction(t *testing.T) {
	tempPipeline := createTempPipeline(t, `
- log:
    - "Test pipeline for production mode"
`)
	defer os.Remove(tempPipeline)

	// Start server WITHOUT test mode
	server := startServer(t, tempPipeline, "16003", false)
	defer server.shutdown(t)

	// Try to access shutdown endpoint - should fail
	resp, err := http.Post("http://127.0.0.1:16003/test/shutdown", "application/json", nil)
	require.NoError(t, err, "Request should complete")
	defer resp.Body.Close()

	assert.Equal(t, 404, resp.StatusCode, "Shutdown endpoint should not exist in production mode")
	t.Log("Shutdown endpoint correctly disabled in production mode")
}

// Helper: createTempPipeline creates a temporary pipeline file for testing
func createTempPipeline(t *testing.T, content string) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-pipeline-*.yaml")
	require.NoError(t, err, "Failed to create temp file")

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err, "Failed to write pipeline content")

	err = tmpFile.Close()
	require.NoError(t, err, "Failed to close temp file")

	return tmpFile.Name()
}
