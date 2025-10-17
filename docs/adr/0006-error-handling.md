# Error Handling Strategy

- Status: Accepted
- Deciders: Development Team
- Date: 2025-10-17

## Context and Problem Statement

Go-trust processes TSLs, validates certificates, transforms XML, and serves API requests. Each of these operations can fail in various ways. How should we handle errors consistently across the codebase to provide good debugging information while maintaining clean APIs?

## Decision Drivers

- Need consistent error handling across packages
- Must provide actionable error messages
- Should preserve error context and stack traces
- Need to distinguish error types (validation, network, parsing, etc.)
- Must support error wrapping and unwrapping
- Should integrate with logging system
- Need to handle both recoverable and fatal errors
- Must provide user-friendly API error responses

## Considered Options

- Return error strings only
- Custom error types for each package
- Sentinel errors (exported error variables)
- Error wrapping with fmt.Errorf and %w
- Third-party error libraries (pkg/errors)
- Panic for errors (Go anti-pattern)

## Decision Outcome

Chosen option: "Custom error types with fmt.Errorf wrapping", because it provides type-safe error handling, preserves context, supports error chains, and integrates well with Go 1.13+ error handling.

### Positive Consequences

- Type-safe error checking with `errors.As()`
- Error wrapping preserves full context
- Stack of errors shows transformation path
- Custom error types carry structured data
- Compatible with standard library
- Easy to test error conditions
- Good error messages for debugging

### Negative Consequences

- More verbose than simple error strings
- Requires defining error types
- Need to document which errors are returned
- Wrapping adds slight overhead

## Error Type Hierarchy

### Pipeline Errors

```go
// pkg/pipeline/errors.go

type PipelineError struct {
    Step      string
    Operation string
    Err       error
}

func (e *PipelineError) Error() string {
    return fmt.Sprintf("pipeline error in step '%s' during %s: %v",
        e.Step, e.Operation, e.Err)
}

func (e *PipelineError) Unwrap() error {
    return e.Err
}
```

Usage:
```go
if err := loadTSL(url); err != nil {
    return &PipelineError{
        Step:      "load",
        Operation: "fetching TSL",
        Err:       err,
    }
}
```

### Validation Errors

```go
type ValidationError struct {
    Field   string
    Value   interface{}
    Reason  string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed for %s (value: %v): %s",
        e.Field, e.Value, e.Reason)
}
```

### Certificate Errors

```go
type CertificateError struct {
    Subject string
    Reason  string
    Err     error
}

func (e *CertificateError) Error() string {
    return fmt.Sprintf("certificate error for '%s': %s",
        e.Subject, e.Reason)
}
```

## Error Wrapping Pattern

Use `fmt.Errorf` with `%w` for wrapping:

```go
func ProcessTSL(url string) error {
    data, err := fetchTSL(url)
    if err != nil {
        return fmt.Errorf("failed to fetch TSL from %s: %w", url, err)
    }
    
    tsl, err := parseTSL(data)
    if err != nil {
        return fmt.Errorf("failed to parse TSL from %s: %w", url, err)
    }
    
    return nil
}
```

This creates an error chain:
```
failed to parse TSL from https://example.com/tsl.xml:
  invalid XML structure:
    unexpected token at line 42
```

## Error Checking Pattern

### Type Checking

```go
var pipelineErr *PipelineError
if errors.As(err, &pipelineErr) {
    logger.Error("Pipeline failed",
        logging.F("step", pipelineErr.Step),
        logging.F("operation", pipelineErr.Operation))
}
```

### Sentinel Errors

```go
var (
    ErrTSLNotFound = errors.New("TSL not found")
    ErrInvalidCert = errors.New("invalid certificate")
)

if errors.Is(err, ErrTSLNotFound) {
    // Handle missing TSL
}
```

## API Error Responses

Convert internal errors to AuthZEN responses:

```go
func handleError(c *gin.Context, err error) {
    response := AuthZENResponse{
        Decision: false,
        Context: map[string]interface{}{
            "id": generateErrorID(),
        },
    }
    
    var certErr *CertificateError
    if errors.As(err, &certErr) {
        response.Context["reason_admin"] = map[string]string{
            "error": certErr.Error(),
        }
        response.Context["reason_user"] = map[string]string{
            "message": "The certificate is not trusted",
        }
        c.JSON(200, response)
        return
    }
    
    // Generic error
    response.Context["reason_admin"] = map[string]string{
        "error": err.Error(),
    }
    c.JSON(500, response)
}
```

## Logging Integration

Errors should be logged with context:

```go
if err != nil {
    logger.Error("Failed to process TSL",
        logging.F("url", url),
        logging.F("error", err.Error()),
        logging.F("error_type", fmt.Sprintf("%T", err)))
    return err
}
```

## Pros and Cons of the Options

### Return error strings only

- Good, because simple
- Good, because no extra types
- Bad, because no structure
- Bad, because difficult to distinguish types
- Bad, because loses context

### Custom error types

- Good, because type-safe
- Good, because structured data
- Good, because easy to test
- Bad, because more code
- Bad, because need to define types

### Sentinel errors

- Good, because simple to check
- Good, because singleton pattern
- Bad, because no context
- Bad, because difficult to parameterize
- Bad, because limited information

### Error wrapping with %w

- Good, because preserves context
- Good, because error chains
- Good, because standard library
- Bad, because can create deep chains
- Bad, because slightly verbose

### Third-party error libraries

- Good, because additional features
- Good, because stack traces
- Bad, because external dependency
- Bad, because standard library is sufficient

### Panic for errors

- Good, because simple (just panic)
- Bad, because breaks control flow
- Bad, because difficult to recover
- Bad, because Go anti-pattern
- Bad, because crashes program

## Error Recovery

### Pipeline Errors

Pipeline errors are recoverable - log and continue:

```go
for _, step := range steps {
    if err := step.Execute(); err != nil {
        logger.Error("Step failed", logging.F("error", err))
        metrics.RecordError(err)
        // Continue with next iteration
    }
}
```

### Fatal Errors

Only panic for truly unrecoverable errors:

```go
if err := initializeLogging(); err != nil {
    panic(fmt.Sprintf("Cannot initialize logging: %v", err))
}
```

## Testing Error Conditions

```go
func TestLoadTSL_NetworkError(t *testing.T) {
    _, err := LoadTSL("http://invalid.example.com/tsl.xml")
    
    require.Error(t, err)
    
    var pipelineErr *PipelineError
    require.True(t, errors.As(err, &pipelineErr))
    assert.Equal(t, "load", pipelineErr.Step)
}
```

## Error Documentation

Document errors in function comments:

```go
// LoadTSL loads a Trust Status List from the given URL.
//
// Returns:
//   - PipelineError: if fetching or parsing fails
//   - ValidationError: if TSL structure is invalid
//   - nil: on success
func LoadTSL(url string) (*TSL, error) {
    // ...
}
```

## Metrics Integration

Record errors by type:

```go
func RecordError(err error) {
    errorType := "unknown"
    
    switch err.(type) {
    case *PipelineError:
        errorType = "pipeline"
    case *ValidationError:
        errorType = "validation"
    case *CertificateError:
        errorType = "certificate"
    }
    
    errorsTotal.WithLabelValues(errorType).Inc()
}
```

## Links

- Implementation: `pkg/pipeline/errors.go`
- Error types: Each package defines its own
- Tests: `pkg/pipeline/errors_test.go`
- Related: [ADR-0007](0007-observability.md) - Observability
- Related: [ADR-0005](0005-api-design.md) - API Design
