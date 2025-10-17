# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Comprehensive developer tooling and documentation
  - Enhanced Makefile with 30+ targets for development workflows
  - VS Code integration (settings, launch configs, tasks, recommended extensions)
  - .editorconfig for cross-editor consistency
  - Developer setup script (scripts/setup-dev.sh)
  - DEVELOPER.md with comprehensive developer guide
  - CONTRIBUTING.md with contribution guidelines
  - Pre-commit hooks for code quality

- Prometheus metrics endpoints
  - `/metrics` endpoint with comprehensive operational metrics
  - Pipeline execution metrics (duration, count, errors, TSL count)
  - API metrics (requests, latency, in-flight requests)
  - Certificate validation metrics
  - Error tracking by type and operation

- Kubernetes-compatible health check endpoints
  - `/health` and `/healthz` for liveness probes
  - `/ready` and `/readiness` for readiness probes
  - Integration with Kubernetes deployment examples

- Production deployment documentation
  - Kubernetes deployment manifests with probes and metrics
  - Docker deployment examples
  - Prometheus ServiceMonitor configuration
  - Health check configuration guidelines

### Changed

- Enhanced README.md with:
  - Production-ready features section
  - Quality and reliability metrics
  - Detailed API endpoint documentation
  - Prometheus metrics examples
  - Kubernetes deployment guide
  - Health check integration examples

### Performance

- Concurrent TSL processing with 2-3x speedup on multi-core systems
- XSLT stylesheet caching for 5-10% additional performance improvement
- Benchmark suite for performance validation
  - API middleware: ~6µs overhead per request
  - Pipeline recording: ~57ns per operation
  - Certificate validation: ~96ns per operation

### Quality

- Test coverage improved to >80% overall, >85% for critical packages
- Added comprehensive benchmark tests for performance-critical operations
- Multiple linters integrated (golangci-lint, gosec, staticcheck)
- Pre-commit hooks for automated quality checks
- CI/CD pipeline with automated testing and coverage tracking

## [Previous Releases]

### Phase 3: Configuration & Performance

#### Added

- Comprehensive configuration system (YAML + environment variables)
- Per-IP API rate limiting with token bucket algorithm
- Concurrent TSL processing with worker pool (2-3x speedup)
- XSLT stylesheet caching for improved performance
- Input validation and sanitization for security
- Path traversal and null byte injection protection

### Phase 2: Quality & Coverage

#### Added

- Comprehensive test coverage for all packages
- Custom error types for pipeline package
- Edge case tests for critical functions
- Integration tests for API and pipeline
- XSLT package tests (0% → 94.1% coverage)
- dsig package tests (64.1% → 81.7% coverage)

#### Changed

- Enhanced error handling with structured errors
- Improved logging with structured fields
- Refactored pipeline steps into focused files

### Phase 1: Core Features

#### Added

- AuthZEN policy decision point for trust evaluation
- TSL management and validation (ETSI TS 119612)
- Certificate validation against trusted services
- Pipeline processing with configurable steps
- XML publishing with digital signatures
- PKCS#11 hardware security module support
- File-based XML digital signatures
- XSLT transformation for TSL to HTML
- Index generation for TSL collections
- Structured logging system
- Command-line and API server modes
- Background TSL processing with configurable frequency

#### Features

- Pipeline YAML configuration
- Embedded XSLT stylesheets
- TSL tree structure for hierarchical organization
- Filtering by territory and service type
- Customizable TSL fetch options (user-agent, timeout)
- Multiple publishing formats (flat, territory-based, index-based)
- SoftHSM testing utilities for PKCS#11

## Version History

### v0.x.x Series

Initial development releases with core functionality:

- Trust decision evaluation via AuthZEN
- TSL processing and validation
- Certificate chain validation
- Digital signature support
- HTML transformation and generation
- API endpoints for trust decisions
- Pipeline-based architecture

---

## Upgrade Notes

### Upgrading to Latest Version

The latest version includes several production-ready features:

1. **Health Checks**: Update Kubernetes deployments to use `/healthz` (liveness) and `/readiness` (readiness)
2. **Metrics**: Add Prometheus scraping for `/metrics` endpoint
3. **Configuration**: Consider migrating to YAML configuration files for easier management
4. **Rate Limiting**: Configure `rate_limit_rps` for production deployments

See [README.md](README.md) and [DEVELOPER.md](DEVELOPER.md) for detailed documentation.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on contributing to this project.

## Links

- [Repository](https://github.com/SUNET/go-trust)
- [Issue Tracker](https://github.com/SUNET/go-trust/issues)
- [Documentation](README.md)
- [Developer Guide](DEVELOPER.md)
