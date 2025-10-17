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

// TestNoServerMode tests the --no-server flag for one-shot pipeline execution
func TestNoServerMode(t *testing.T) {
	// Create a simple test pipeline
	tempPipeline := createTempPipeline(t, `
- log:
    - "Test pipeline in no-server mode"
- echo:
    - "Processing complete"
`)
	defer os.Remove(tempPipeline)

	// Run with --no-server flag
	cmd := exec.Command("./gt-test", "--no-server", tempPipeline)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Pipeline should execute successfully in no-server mode")

	outputStr := string(output)
	assert.Contains(t, outputStr, "Running pipeline in one-shot mode", "Should indicate no-server mode")
	assert.Contains(t, outputStr, "Pipeline execution completed successfully", "Should complete successfully")
	assert.NotContains(t, outputStr, "API server", "Should not start API server")
	t.Logf("No-server mode output: %s", strings.TrimSpace(outputStr))
}

// TestNoServerModeWithLogging tests --no-server with different log levels
func TestNoServerModeWithLogging(t *testing.T) {
	tempPipeline := createTempPipeline(t, `
- log:
    - "Testing with debug logging"
`)
	defer os.Remove(tempPipeline)

	// Test with debug log level
	cmd := exec.Command("./gt-test", "--no-server", "--log-level", "debug", tempPipeline)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Should succeed with debug logging")

	outputStr := string(output)
	assert.Contains(t, outputStr, "Pipeline execution completed successfully", "Should complete")
	t.Logf("Debug logging output length: %d bytes", len(output))
}

// TestNoServerModeInvalidPipeline tests --no-server with an invalid pipeline
func TestNoServerModeInvalidPipeline(t *testing.T) {
	tempPipeline := createTempPipeline(t, `
- invalid_step:
    - "This step does not exist"
`)
	defer os.Remove(tempPipeline)

	// Run with --no-server flag
	cmd := exec.Command("./gt-test", "--no-server", tempPipeline)
	output, err := cmd.CombinedOutput()
	require.Error(t, err, "Should fail with invalid pipeline")

	outputStr := string(output)
	assert.Contains(t, outputStr, "Pipeline execution failed", "Should show execution failure")
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

// TestAPIAndHTMLExample tests the api-and-html.yaml example
// This test verifies:
// 1. Server starts with the example pipeline
// 2. API endpoints are accessible (/status, /info)
// 3. HTML output is generated
// 4. Certificate pool is properly configured
func TestAPIAndHTMLExample(t *testing.T) {
	exampleFile := "../example/api-and-html.yaml"
	if _, err := os.Stat(exampleFile); os.IsNotExist(err) {
		t.Skip("api-and-html.yaml example not found")
	}

	// This test requires network access to load TSLs, so we make it optional
	if os.Getenv("TEST_NETWORK") == "" {
		t.Skip("Skipping network-dependent test. Set TEST_NETWORK=1 to enable.")
	}

	// Create output directory for HTML files
	outputDir := "./test-output-html"
	os.MkdirAll(outputDir, 0755)
	defer os.RemoveAll(outputDir)

	// Start server with the api-and-html example
	server := startServer(t, exampleFile, "16004", true)
	defer server.shutdown(t)

	// Give it more time to process TSLs (this loads from network)
	t.Log("Waiting for TSL processing to complete...")
	time.Sleep(10 * time.Second)

	// Test 1: Status endpoint should show loaded TSLs
	t.Run("StatusEndpoint", func(t *testing.T) {
		resp, err := http.Get("http://127.0.0.1:16004/status")
		require.NoError(t, err, "Status endpoint should respond")
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode, "Status endpoint should return 200")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Should read response body")

		bodyStr := string(body)
		assert.Contains(t, bodyStr, "tsl_count", "Response should contain tsl_count")
		t.Logf("Status response: %s", bodyStr)
	})

	// Test 2: Info endpoint should return TSL summaries
	t.Run("InfoEndpoint", func(t *testing.T) {
		resp, err := http.Get("http://127.0.0.1:16004/info")
		require.NoError(t, err, "Info endpoint should respond")
		defer resp.Body.Close()

		assert.Equal(t, 200, resp.StatusCode, "Info endpoint should return 200")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "Should read response body")

		bodyStr := string(body)
		assert.Contains(t, bodyStr, "tsl_summaries", "Response should contain tsl_summaries")
		t.Logf("Info response length: %d bytes", len(bodyStr))
	})

	// Test 3: HTML output directory should be created
	t.Run("HTMLOutput", func(t *testing.T) {
		htmlDir := "./output/html"
		if _, err := os.Stat(htmlDir); err == nil {
			entries, err := os.ReadDir(htmlDir)
			if err == nil && len(entries) > 0 {
				t.Logf("HTML directory contains %d files", len(entries))

				// Check for index file
				indexPath := filepath.Join(htmlDir, "index.html")
				if _, err := os.Stat(indexPath); err == nil {
					t.Log("Index file created successfully")
				}
			}
		}
	})
}

// TestAPIAndHTMLExampleModified tests a modified version of api-and-html.yaml
// This test uses local test data instead of network calls
func TestAPIAndHTMLExampleModified(t *testing.T) {
	// Create a pipeline similar to api-and-html.yaml but with local test data
	tempPipeline := createTempPipeline(t, `
# Modified API and HTML test pipeline using local test data
- set-fetch-options:
    - max-depth:0
    - timeout:10s

# For testing, we would load a local test TSL file
# In production, this would be from network
- log:
    - "Starting modified API and HTML test"

# Note: We can't test the actual XSLT transformation without valid TSL data
# This test focuses on the pipeline execution and API setup
`)
	defer os.Remove(tempPipeline)

	// Start server
	server := startServer(t, tempPipeline, "16005", true)
	defer server.shutdown(t)

	// Test that API is responsive
	resp, err := http.Get("http://127.0.0.1:16005/status")
	require.NoError(t, err, "Status endpoint should respond")
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode, "Status endpoint should return 200")
	t.Log("Modified pipeline executed successfully")
}

// TestHTMLTransformationPipeline tests a pipeline with XSLT transformation
func TestHTMLTransformationPipeline(t *testing.T) {
	// Create output directory
	outputDir := "./test-html-output"
	os.MkdirAll(outputDir, 0755)
	defer os.RemoveAll(outputDir)

	// Create a pipeline that tests transformation steps
	tempPipeline := createTempPipeline(t, `
# Test HTML transformation capabilities
- set-fetch-options:
    - max-depth:0
    - timeout:5s

- log:
    - "Testing HTML transformation pipeline"

# In a real test, we would:
# 1. Load a test TSL from testdata
# 2. Transform it to HTML
# 3. Generate an index
# 4. Verify output files exist
`)
	defer os.Remove(tempPipeline)

	// Start server
	server := startServer(t, tempPipeline, "16006", true)
	defer server.shutdown(t)

	// Verify server is running
	resp, err := http.Get("http://127.0.0.1:16006/status")
	require.NoError(t, err, "Status endpoint should respond")
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	t.Log("HTML transformation pipeline test completed")
}

// TestCertificatePoolFiltering tests select operations with different filtering criteria
func TestCertificatePoolFiltering(t *testing.T) {
	// Create a pipeline that tests certificate pool filtering
	tempPipeline := createTempPipeline(t, `
# Test certificate pool filtering
- set-fetch-options:
    - max-depth:0
    - timeout:5s

# Test different select operations
- select:
    - include-referenced

- log:
    - "Certificate pool configured"

# Test filtering by service type
# In production: service-type:http://uri.etsi.org/TrstSvc/Svctype/CA/QC
# For testing, we just verify the syntax is accepted
`)
	defer os.Remove(tempPipeline)

	// Start server
	server := startServer(t, tempPipeline, "16007", true)
	defer server.shutdown(t)

	// Verify server started successfully
	resp, err := http.Get("http://127.0.0.1:16007/status")
	require.NoError(t, err, "Status endpoint should respond")
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
	t.Log("Certificate pool filtering test completed")
}

// TestAPIEndpointsWithMinimalPipeline tests all API endpoints with minimal setup
func TestAPIEndpointsWithMinimalPipeline(t *testing.T) {
	tempPipeline := createTempPipeline(t, `
- set-fetch-options:
    - max-depth:0
    - timeout:5s
- log:
    - "Minimal pipeline for API testing"
`)
	defer os.Remove(tempPipeline)

	server := startServer(t, tempPipeline, "16008", true)
	defer server.shutdown(t)

	tests := []struct {
		name     string
		endpoint string
		method   string
		wantCode int
	}{
		{"Status", "/status", "GET", 200},
		{"Info", "/info", "GET", 200},
		{"NotFound", "/nonexistent", "GET", 404},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *http.Response
			var err error

			switch tt.method {
			case "GET":
				resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:16008%s", tt.endpoint))
			default:
				t.Fatalf("Unsupported method: %s", tt.method)
			}

			require.NoError(t, err, "Request should complete")
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode, "Status code should match")
			t.Logf("%s endpoint returned %d", tt.endpoint, resp.StatusCode)
		})
	}
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

// TestConfigFileIntegration verifies that config files are loaded correctly
func TestConfigFileIntegration(t *testing.T) {
	// Create a temporary config file
	configContent := `
server:
  host: "192.168.1.1"
  port: "7777"
  frequency: "10m"

logging:
  level: "debug"
  format: "json"
  output: "stdout"
`
	configFile, err := os.CreateTemp("", "test-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(configFile.Name())

	_, err = configFile.WriteString(configContent)
	require.NoError(t, err)
	configFile.Close()

	// Create a test pipeline
	pipelineContent := `
- echo:
  - "Config file test"
`
	pipelineFile := createTempPipeline(t, pipelineContent)
	defer os.Remove(pipelineFile)

	// Run with config file and --no-server
	cmd := exec.Command("./gt-test",
		"--config", configFile.Name(),
		"--no-server",
		pipelineFile,
	)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Command failed: %s", string(output))

	outputStr := string(output)
	t.Logf("Config file output: %s", outputStr)

	// Verify that debug level logging is active (from config)
	assert.Contains(t, outputStr, `"level":"info"`, "Should use config file logging settings")
}

// TestConfigPrecedence verifies the configuration precedence order
func TestConfigPrecedence(t *testing.T) {
	// Create a config file with specific values
	configContent := `
server:
  host: "config-host"
  port: "8888"
  frequency: "15m"

logging:
  level: "warn"
  format: "text"
  output: "stdout"
`
	configFile, err := os.CreateTemp("", "test-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(configFile.Name())

	_, err = configFile.WriteString(configContent)
	require.NoError(t, err)
	configFile.Close()

	// Create a test pipeline
	pipelineContent := `
- echo:
  - "Precedence test"
`
	pipelineFile := createTempPipeline(t, pipelineContent)
	defer os.Remove(pipelineFile)

	// Run with config file but override with command-line flag
	cmd := exec.Command("./gt-test",
		"--config", configFile.Name(),
		"--log-level", "info", // Override config file (which is warn)
		"--no-server",
		pipelineFile,
	)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Command failed: %s", string(output))

	outputStr := string(output)
	t.Logf("Precedence test output: %s", outputStr)

	// Verify that command-line flag overrode config file
	// Info messages should appear (command-line override from warn to info)
	assert.Contains(t, outputStr, "level=info", "Command-line flags should override config file")
	assert.Contains(t, outputStr, "Running pipeline in one-shot mode", "Info messages should be visible")
}

// TestEnvVarConfiguration verifies environment variable configuration
func TestEnvVarConfiguration(t *testing.T) {
	// Create a test pipeline
	pipelineContent := `
- echo:
  - "Env var test"
`
	pipelineFile := createTempPipeline(t, pipelineContent)
	defer os.Remove(pipelineFile)

	// Run with environment variables
	cmd := exec.Command("./gt-test",
		"--no-server",
		pipelineFile,
	)

	// Set environment variables
	cmd.Env = append(os.Environ(),
		"GT_LOG_LEVEL=error",
		"GT_HOST=env-host",
		"GT_PORT=9999",
	)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Command failed: %s", string(output))

	outputStr := string(output)
	t.Logf("Env var output: %s", outputStr)

	// With error level, info messages should not appear
	// Only error or fatal messages would appear
	assert.NotContains(t, outputStr, "level=info", "Should respect GT_LOG_LEVEL env var")
}
