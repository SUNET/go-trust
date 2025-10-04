# Go-Trust

<div align="center">

[![Go Reference](https://pkg.go.dev/badge/github.com/SUNET/go-trust.svg)](https://pkg.go.dev/github.com/SUNET/go-trust)
[![Go Report Card](https://goreportcard.com/badge/github.com/SUNET/go-trust)](https://goreportcard.com/report/github.com/SUNET/go-trust)
[![Coverage](https://raw.githubusercontent.com/SUNET/go-trust/badges/.badges/main/coverage.svg)](https://github.com/SUNET/go-trust/actions/workflows/go.yml)
[![Go Compatibility](https://raw.githubusercontent.com/SUNET/go-trust/badges/.badges/main/golang.svg)](https://go.dev/)
[![Build Status](https://img.shields.io/github/actions/workflow/status/SUNET/go-trust/go.yml?branch=main)](https://github.com/SUNET/go-trust/actions)
[![License](https://img.shields.io/badge/License-BSD_2--Clause-orange.svg)](https://opensource.org/licenses/BSD-2-Clause)
[![Latest Release](https://img.shields.io/github/v/release/SUNET/go-trust?include_prereleases)](https://github.com/SUNET/go-trust/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/SUNET/go-trust)](https://go.dev/)

[![Issues](https://img.shields.io/github/issues/SUNET/go-trust)](https://github.com/SUNET/go-trust/issues)
[![Last Commit](https://img.shields.io/github/last-commit/SUNET/go-trust)](https://github.com/SUNET/go-trust/commits/main)
[![CodeQL](https://github.com/SUNET/go-trust/actions/workflows/codeql.yml/badge.svg)](https://github.com/SUNET/go-trust/actions/workflows/codeql.yml)
[![Dependency Status](https://img.shields.io/librariesio/github/SUNET/go-trust)](https://libraries.io/github/SUNET/go-trust)

</div>

## Overview

Go-Trust is a local trust engine that provides trust decisions based on ETSI TS 119612 Trust Status Lists (TSLs). It allows clients to abstract trust decisions through an AuthZEN policy decision point (PDP). The service evaluates trust in subjects identified by X509 certificates using a set of TSLs.

## Features

- **AuthZEN Integration**: Policy decision point for trust evaluation
- **TSL Management**: Process and validate ETSI TS 119612 Trust Status Lists
- **Certificate Validation**: Evaluate X509 certificates against trusted services
- **Pipeline Processing**: Flexible TSL processing with configurable steps
- **XML Publishing**: Serialize TSLs to XML for distribution
- **XML Signing**: Sign XML documents using file-based keys or PKCS#11 hardware security modules

## Installation

```bash
# Clone the repository
git clone https://github.com/SUNET/go-trust.git
cd go-trust

# Build the project
make build

# Run tests
make test
```

## Examples

The [example](./example/) directory contains:
- Example directory structure for generating TSLs
- Sample pipeline configuration
- Usage examples for various trust scenarios

## XSLT Transformation & HTML Index Generation

Go-Trust includes built-in tools for transforming TSLs into user-friendly HTML documents and creating index pages for collections of TSLs.

### Using Embedded XSLT in Pipeline Configuration

The stylesheet is embedded in the binary, so you don't need to distribute separate files:

```yaml
- transform:
- embedded:tsl-to-html.xslt
- /output/directory
- html
```

This configuration transforms all TSLs in the pipeline to HTML using the embedded stylesheet and writes the output files to the specified directory.

### Available Embedded Stylesheets

- **tsl-to-html.xslt**: Transforms TSLs into comprehensive HTML documents with PicoCSS styling

### Generating an Index for HTML TSLs

After transforming TSLs to HTML, you can generate an index.html file that lists all the TSLs with key metadata:

```yaml
- generate_index:
- /output/directory
- "Trust Service Lists Index"
```

The index page includes:
- Links to each TSL HTML file
- Territory codes and badges
- Sequence numbers and dates
- Service counts
- TSL types

For a complete example, see [transform-with-index.yaml](./example/transform-with-index.yaml) in the examples directory.

## Digital Signatures

Go-Trust includes a dedicated package for XML digital signatures in [pkg/dsig](./pkg/dsig/). This package supports:

- File-based certificate and key signing
- PKCS#11 hardware security module integration
- Standardized interface for all signing methods
- Testing utilities for PKCS#11 with SoftHSM

## Usage

### Command Line Interface

Go-Trust provides a command line interface for processing TSLs and managing trust decisions:

```bash
# Run the trust service with a pipeline configuration
./gt ./pipeline.yaml

# Run with custom settings
./gt --host 0.0.0.0 --port 8080 --frequency 1h ./pipeline.yaml
```

### API Endpoints

The service exposes several HTTP endpoints:

- **GET /status**: Check service health status
- **GET /info**: Get information about loaded TSLs
- **POST /authzen/decision**: Evaluate trust decisions for X509 certificates

Example AuthZEN decision request:

```json
{
  "subject": {
    "type": "x509_certificate",
    "id": "subject-123",
    "properties": {
      "x5c": [
        "MIIDQjCCAiqgAwIBAgIUJlq+zz4..."
      ]
    }
  },
  "resource": {
    "type": "service",
    "id": "resource-123",
    "properties": {}
  },
  "action": {
    "name": "trust",
    "properties": {}
  },
  "context": {}
}
```

Example response:

```json
{
  "decision": true
}
```

Or with error details:

```json
{
  "decision": false,
  "context": {
    "id": "err-123",
    "reason_admin": {
      "error": "certificate has expired or is not yet valid"
    },
    "reason_user": {
      "message": "The certificate is not trusted"
    }
  }
}
```

## Pipeline Steps

Go-Trust uses a pipeline architecture for TSL processing:

1. **Load**: Read TSLs from files or URLs
2. **Select**: Filter TSLs based on criteria
3. **Publish**: Serialize TSLs to XML files
4. **Custom**: Add your own processing steps

### XML Digital Signatures

Go-Trust supports XML-DSIG signatures for published TSLs using either:

1. **File-based certificates and keys**: Standard PEM-encoded X.509 certificates and private keys
2. **PKCS#11 hardware tokens**: HSMs or smart cards for secure key storage and operations

#### File-Based Signing

For development and testing environments, you can use file-based certificates and private keys:

```yaml
- publish: ["./output", "/path/to/cert.pem", "/path/to/key.pem"]
```

This method reads the certificate and private key from PEM-encoded files.

#### PKCS#11 Hardware Token Signing

For production environments, you can use PKCS#11 hardware security modules (HSMs) or smart cards:

```yaml
- publish: ["./output", "pkcs11:module=/path/to/lib;pin=1234;slot-id=0", "key-label", "cert-label"]
```

The PKCS#11 URI format follows RFC 7512 and supports these parameters:

- `module`: Path to PKCS#11 library/middleware (required)
- `pin`: PIN code for token access (required)
- `slot-id`: Numeric slot identifier (optional)
- `token`: Token label for identifying the token (optional, alternative to slot-id)

The `key-label` and `cert-label` arguments specify the labels used to identify the private key and certificate in the HSM.

Example pipeline configuration (YAML):

```yaml
# Pipeline steps are defined as a sequence of operations
- generate: ["./example/example-tsl"]  # Generate TSL from directory
- select: []                          # Extract certificates into a pool
- publish: ["./output"]               # Publish TSLs as XML files
- publish: ["./output", "/path/to/cert.pem", "/path/to/key.pem"]  # Publish with file-based XML-DSIG signatures
- publish: ["./output", "pkcs11:module=/usr/lib/softhsm/libsofthsm2.so;pin=1234;slot-id=0", "tsl-signing-key", "tsl-signing-cert"]  # Publish with PKCS#11 XML-DSIG signatures
```

#### HSM Compatibility

The PKCS#11 implementation has been tested with:
- SoftHSM (for development/testing)
- Thales Luna HSM
- YubiKey (via PIV application)

For other HSMs, you may need to adjust the configuration according to your device's specifications.

## Configuration

The application is configured using command-line flags:

```bash
# Start the API server with custom settings
./gt --host 0.0.0.0 --port 8080 --frequency 1h ./path/to/pipeline.yaml
```

Available command-line options:
- `--host`: API server hostname (default: 127.0.0.1)
- `--port`: API server port (default: 6001)
- `--frequency`: Pipeline update frequency (default: 5m)
- `--help`: Show help message
- `--version`: Show version information
```

## Development

### Requirements

- Go 1.18+
- Access to ETSI TS 119612 TSLs or sample data
- Make for build automation

### Building from Source

```bash
# Build binary
make build

# Run tests
make test

# Check code coverage
make coverage
```

### Project Structure

```
go-trust/
├── .github/        # GitHub configuration
│   └── workflows/  # GitHub Actions workflows
├── cmd/            # Command line tools
├── pkg/            # Core packages
│   ├── api/        # HTTP API implementation
│   ├── authzen/    # AuthZEN integration
│   └── pipeline/   # TSL processing pipeline
├── example/        # Example configurations and data
└── tests/          # Integration tests
```

### CI/CD Workflows

This project uses GitHub Actions for continuous integration and delivery:

- **Go Workflow** (`go.yml`): Builds, tests, and checks code coverage
- **Release Workflow** (`release.yml`): Creates releases when new tags are pushed
- **CodeQL Analysis** (`codeql.yml`): Scans code for security vulnerabilities
- **Dependency Review** (`dependency-review.yml`): Checks dependencies for security issues

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Testing

Before submitting a pull request, please ensure:

```bash
# All tests pass
make test

# Code coverage is maintained or improved
make coverage

# Code follows project style guidelines
go fmt ./...
go vet ./...
```

The CI pipeline will automatically run these checks when you submit a pull request. All checks must pass before a PR can be merged.

#### PKCS#11 Testing with SoftHSM

Go-Trust includes tests for PKCS#11-based XML-DSIG signing using SoftHSM. These tests are skipped if SoftHSM is not installed.

To run the PKCS#11 tests with SoftHSM:

1. Install SoftHSM version 2:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install softhsm2

   # CentOS/RHEL
   sudo yum install softhsm2

   # macOS with Homebrew
   brew install softhsm
   ```

2. Run the tests:
   ```bash
   # Run all tests, including SoftHSM tests if available
   go test ./...

   # Run only the SoftHSM tests
   go test ./pkg/pipeline -run TestPKCS11SignerWithSoftHSM
   ```

The test will:
1. Create a temporary SoftHSM token
2. Generate a test certificate and private key
3. Import them into the token
4. Use the PKCS11Signer to sign XML data
5. Clean up the temporary token when done

These tests ensure that the PKCS#11 signing functionality works correctly with hardware security modules.

### PKCS#11 Signing Implementation

Go-Trust now uses the improved Signer interface from the goxmldsig library to handle XML-DSIG signatures with PKCS#11 hardware tokens. This integration provides several benefits:

1. **More Consistent API**: The signing code follows a unified interface approach
2. **Better Abstraction**: The signing mechanism is abstracted behind the Signer interface
3. **Simpler Maintenance**: Reduced code duplication and complexity
4. **Improved Security**: Direct hardware token integration with no private key exposure

When using a PKCS#11 token for signing, the library:
1. Connects to the token using the provided configuration
2. Locates the key and certificate based on their labels/IDs
3. Creates a PKCS11Signer that implements the goxmldsig Signer interface
4. Uses the PKCS11Signer to sign the XML document without ever exposing the private key material

## Logging System

Go-Trust includes a flexible, structured logging system built around an abstract Logger interface, with implementations available for different logging backends.

### Logging Interface

The logging system is designed with the following features:

- **Structured Logging**: Log entries include structured fields, not just text messages
- **Log Levels**: Support for Debug, Info, Warn, Error, and Fatal levels
- **Context Awareness**: Logging with context propagation
- **Extensible**: Support for different logging backends through adapters

### Using Logging in Pipeline Configuration

Pipeline YAML configurations support logging configuration through a dedicated `config` section:

```yaml
config:
  logging:
    level: info    # debug, info, warn, error, or fatal
    format: text   # text or json

pipes:
  - log:
  - "Processing TSL files"
  - count=5
  - source=example.com
  
  - log:
  - level=debug "Detailed debugging information"
  - tsl_id=SETSL123
```

### Log Pipeline Step

The `log` pipeline step allows logging messages with structured data:

```yaml
- log:
- "Message to log"
- key1=value1
- key2=value2
```

To specify a log level other than the default (info):

```yaml
- log:
- level=debug "Debug message with more details"
- operation=validation
- result=success
```

### Programmatic Usage

When extending Go-Trust, you can use the logging system programmatically:

```go
import "github.com/SUNET/go-trust/pkg/logging"

func MyFunction() {
    logger := logging.DefaultLogger()
    
    // Simple logging
    logger.Info("Processing started")
    
    // With structured fields
    logger.Debug("Validation details", 
        logging.F("certCount", 5),
        logging.F("valid", true),
    )
    
    // With context
    ctx := context.Background()
    ctxLogger := logger.WithContext(ctx)
    ctxLogger.Info("Operation completed")
}
```

## License

This project is licensed under the BSD 2-Clause License - see the [LICENSE.txt](LICENSE.txt) file for details.

## Acknowledgments

- [ETSI TS 119612](https://www.etsi.org/deliver/etsi_ts/119600_119699/119612/) - Trust-service status list format
- [AuthZEN](https://authzen.dev/) - Authorization framework
- [SUNET](https://www.sunet.se/) - Swedish University Network

