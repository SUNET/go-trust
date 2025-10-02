# Go-Trust

<div align="center">

[![Go Reference](https://pkg.go.dev/badge/github.com/SUNET/go-trust.svg)](https://pkg.go.dev/github.com/SUNET/go-trust)
[![Go Report Card](https://goreportcard.com/badge/github.com/SUNET/go-trust)](https://goreportcard.com/report/github.com/SUNET/go-trust)
![Coverage](https://img.shields.io/badge/coverage-75.9%25-brightgreen)
[![Build Status](https://img.shields.io/github/actions/workflow/status/SUNET/go-trust/go.yml?branch=main)](https://github.com/SUNET/go-trust/actions)
[![License](https://img.shields.io/badge/License-BSD_2--Clause-orange.svg)](https://opensource.org/licenses/BSD-2-Clause)
[![Latest Release](https://img.shields.io/github/v/release/SUNET/go-trust?include_prereleases)](https://github.com/SUNET/go-trust/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/SUNET/go-trust)](https://go.dev/)
[![Issues](https://img.shields.io/github/issues/SUNET/go-trust)](https://github.com/SUNET/go-trust/issues)
[![Last Commit](https://img.shields.io/github/last-commit/SUNET/go-trust)](https://github.com/SUNET/go-trust/commits/main)

</div>

## Overview

Go-Trust is a local trust engine that provides trust decisions based on ETSI TS 119612 Trust Status Lists (TSLs). It allows clients to abstract trust decisions through an AuthZEN policy decision point (PDP). The service evaluates trust in subjects identified by X509 certificates using a set of TSLs.

## Features

- **AuthZEN Integration**: Policy decision point for trust evaluation
- **TSL Management**: Process and validate ETSI TS 119612 Trust Status Lists
- **Certificate Validation**: Evaluate X509 certificates against trusted services
- **Pipeline Processing**: Flexible TSL processing with configurable steps
- **XML Publishing**: Serialize TSLs to XML for distribution

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

## Usage

### Command Line Interface

Go-Trust provides a command line interface for processing TSLs and managing trust decisions:

```bash
# Run the trust service
./gt serve --config config.yaml

# Process TSLs using a pipeline configuration
./gt pipeline --config pipeline.yaml
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
    "id": "MIIC..."
  },
  "action": {
    "name": "trust"
  }
}
```

## Pipeline Steps

Go-Trust uses a pipeline architecture for TSL processing:

1. **Load**: Read TSLs from files or URLs
2. **Select**: Filter TSLs based on criteria
3. **Publish**: Serialize TSLs to XML files
4. **Custom**: Add your own processing steps

Example pipeline configuration (YAML):

```yaml
pipeline:
  - method: load_tsl
    args:
      - "https://example.com/trustlist.xml"
  - method: select_cert_pool
  - method: publish_tsl
    args:
      - "/path/to/output/directory"
```

## Configuration

Example configuration file:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

tsl:
  sources:
    - url: "https://example.com/trustlist.xml"
      refresh_interval: "24h"
  cache_dir: "/var/cache/go-trust"

logging:
  level: "info"
  format: "json"
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

## License

This project is licensed under the BSD 2-Clause License - see the [LICENSE.txt](LICENSE.txt) file for details.

## Acknowledgments

- [ETSI TS 119612](https://www.etsi.org/deliver/etsi_ts/119600_119699/119612/) - Trust-service status list format
- [AuthZEN](https://authzen.dev/) - Authorization framework
- [SUNET](https://www.sunet.se/) - Swedish University Network

