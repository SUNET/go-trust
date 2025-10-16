# Integration Tests for main.go

This directory contains integration tests for the `go-trust` command-line application. These tests verify the complete application behavior including:

- Command-line argument parsing
- Pipeline loading and execution
- API server startup and endpoints
- Graceful shutdown in test mode

## Running Integration Tests

Integration tests are **disabled by default** because they:
- Start actual API servers on network ports
- May take longer to execute
- Require more system resources

### Enable Integration Tests

Set the `RUN_INTEGRATION_TESTS` environment variable:

```bash
cd cmd
RUN_INTEGRATION_TESTS=1 go test -v
```

Or use make:

```bash
make test-integration
```

### Test Mode

Some tests require a special **test mode** that enables a `/test/shutdown` endpoint for graceful server shutdown. This endpoint is **only available when `GO_TRUST_TEST_MODE=1` is set** and is automatically disabled in production.

The test framework automatically sets this environment variable when starting servers that need it.

## Test Categories

### 1. Command-Line Interface Tests

These tests verify basic CLI functionality without starting a server:

- `TestVersionFlag`: Tests `--version` flag
- `TestHelpFlag`: Tests `--help` flag  
- `TestMissingPipelineFile`: Tests error handling for missing arguments
- `TestInvalidPipelineFile`: Tests error handling for invalid pipeline files

**These run quickly and don't require network access.**

### 2. API Server Tests

These tests start actual API server instances:

- `TestAPIServerStatusEndpoint`: Tests `/status` endpoint
- `TestShutdownEndpoint`: Tests `/test/shutdown` endpoint (test mode)
- `TestShutdownEndpointDisabledInProduction`: Verifies shutdown endpoint is disabled in production

**These start servers on ports 16001-16003 and require test mode.**

### 3. Pipeline Execution Tests

Tests that verify pipeline YAML files from the `example/` directory work correctly:

- `TestBasicPipelineExecution`: Tests basic pipeline functionality

**These may take longer as they process actual TSL data.**

## Test Utilities

### `serverProcess`

A helper type that manages running server instances:

```go
type serverProcess struct {
    cmd        *exec.Cmd
    port       string
    testMode   bool
    shutdownCh chan struct{}
}
```

#### `startServer(t, pipelineFile, port, testMode) *serverProcess`

Starts an API server in a separate process:
- Builds the binary if needed
- Waits for server to be ready (polls `/status`)
- Returns a `serverProcess` for management

#### `(*serverProcess).shutdown(t)`

Gracefully stops the server:
- In test mode: Uses `/test/shutdown` endpoint
- Fallback: Kills the process

### `createTempPipeline(t, content) string`

Creates a temporary pipeline YAML file for testing:
- Automatically cleaned up after test
- Returns the file path

## Port Allocation

Tests use ports **16001-16003** to avoid conflicts with:
- Default production port (6001)
- Other services
- Multiple simultaneous test runs

Each test uses a different port to enable parallel execution.

## Safety Features

### Test Mode Shutdown Endpoint

The `/test/shutdown` endpoint is:

✅ **Enabled when**: `GO_TRUST_TEST_MODE=1` is set
❌ **Disabled in production**: Returns 404 without the environment variable
⚠️  **Warning logged**: Server logs a warning when test mode is active

This ensures integration tests can cleanly shut down servers without leaving orphaned processes.

### Test Isolation

- Each test uses a unique port
- Temporary files are cleaned up
- Test binary is removed after all tests complete
- Server processes are tracked and terminated

## Example: Running a Specific Test

```bash
cd cmd
RUN_INTEGRATION_TESTS=1 go test -v -run TestVersionFlag
```

## Example: Running All Integration Tests

```bash
cd cmd
RUN_INTEGRATION_TESTS=1 go test -v -timeout 5m
```

## Debugging

### View Server Output

Server stdout/stderr is captured and displayed. Use `-v` flag to see all output:

```bash
RUN_INTEGRATION_TESTS=1 go test -v
```

### Check Server Readiness

Tests wait up to 30 seconds for the server to become ready. If a test times out:

1. Check if the port is already in use
2. Verify the pipeline YAML is valid
3. Check system resources

### Manual Testing

You can manually test with the test mode enabled:

```bash
# Build the binary
go build -o gt-test .

# Run with test mode
GO_TRUST_TEST_MODE=1 ./gt-test test-pipeline.yaml

# In another terminal, test the shutdown endpoint
curl -X POST http://127.0.0.1:6001/test/shutdown
```

## CI/CD Integration

For continuous integration, add to your workflow:

```yaml
- name: Run integration tests
  run: |
    cd cmd
    RUN_INTEGRATION_TESTS=1 go test -v -timeout 5m
```

## Future Enhancements

Potential additions to the integration test suite:

- [ ] Test all example pipeline files
- [ ] Test AuthZEN endpoint with real certificates
- [ ] Test pipeline with network TSL loading (requires mock server)
- [ ] Test XSLT transformations
- [ ] Test certificate pool generation
- [ ] Test background updater frequency
- [ ] Test logging options (--log-level, --log-format, --log-output)
- [ ] Performance benchmarks

## Troubleshooting

### "Skipping main.go integration tests"

**Solution**: Set `RUN_INTEGRATION_TESTS=1`

### "Port already in use"

**Solution**: Kill any running servers or use different ports

### "Server did not become ready in time"

**Solution**: 
- Check if pipeline YAML is valid
- Increase timeout in `startServer()` function
- Verify network connectivity if pipeline loads remote TSLs

### Tests hang

**Solution**:
- Check if servers are shutting down properly
- Verify `/test/shutdown` endpoint is accessible
- Use `killall gt-test` to clean up orphaned processes

## Security Note

⚠️ **NEVER deploy with `GO_TRUST_TEST_MODE=1` in production!**

The test mode shutdown endpoint allows any client to terminate the server. This is intentional for testing but would be a severe security vulnerability in production.

The application logs a warning when test mode is active to prevent accidental production deployment.
