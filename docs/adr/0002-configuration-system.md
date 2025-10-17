# Hierarchical Configuration System

- Status: Accepted
- Deciders: Development Team
- Date: 2025-10-17

## Context and Problem Statement

Go-trust needs a flexible configuration system for deployment in various environments (development, staging, production, Kubernetes, Docker). How should we structure configuration to support different deployment scenarios while maintaining security and ease of use?

## Decision Drivers

- Support multiple deployment environments
- Enable 12-factor app principles (configuration via environment)
- Provide sane defaults for quick start
- Allow per-environment customization without code changes
- Support secrets management (API keys, passwords)
- Clear precedence order for configuration sources
- Type-safe configuration parsing

## Considered Options

- Environment variables only
- Configuration file only (YAML/TOML)
- Hierarchical system (defaults → file → env vars → flags)
- Remote configuration service (etcd, Consul)
- Configuration structs in code

## Decision Outcome

Chosen option: "Hierarchical system with defaults → file → environment variables → command-line flags", because it provides maximum flexibility while supporting all deployment scenarios from local development to Kubernetes.

### Positive Consequences

- Developers can run with defaults (no config needed)
- Production can use config files for complex settings
- Kubernetes can override via environment variables
- Command-line flags available for quick testing
- Clear precedence order prevents confusion
- Secrets can be injected via environment variables
- Configuration is documented in one place

### Negative Consequences

- More complex implementation than single source
- Need to document precedence order
- Potential for configuration conflicts
- Requires validation at multiple levels

## Configuration Precedence

1. **Command-line flags** (highest priority)
   - Explicit user intent
   - Best for testing and one-off changes
   - Example: `--port 8080 --log-level debug`

2. **Environment variables**
   - Kubernetes/Docker standard
   - Secrets management integration
   - Prefix: `GT_` (e.g., `GT_PORT=8080`)

3. **Configuration file**
   - Complex multi-value settings
   - Version-controlled defaults
   - Format: YAML for readability

4. **Built-in defaults** (lowest priority)
   - Sensible defaults for development
   - Quick start without configuration

## Implementation Details

### Configuration Structure

```go
type Config struct {
    Server ServerConfig `yaml:"server"`
    Logging LoggingConfig `yaml:"logging"`
    Pipeline PipelineConfig `yaml:"pipeline"`
    Security SecurityConfig `yaml:"security"`
}
```

### Example Configuration File

```yaml
server:
  host: "0.0.0.0"
  port: "6001"
  frequency: "5m"

logging:
  level: "info"
  format: "json"
  output: "stdout"

security:
  rate_limit_rps: 100
  enable_cors: false
```

### Environment Variable Mapping

- `GT_HOST` → `server.host`
- `GT_PORT` → `server.port`
- `GT_LOG_LEVEL` → `logging.level`
- `GT_RATE_LIMIT_RPS` → `security.rate_limit_rps`

### Built-in Defaults

```go
var DefaultConfig = Config{
    Server: ServerConfig{
        Host: "127.0.0.1",
        Port: "6001",
        Frequency: "5m",
    },
    Logging: LoggingConfig{
        Level: "info",
        Format: "text",
        Output: "stdout",
    },
}
```

## Pros and Cons of the Options

### Environment variables only

- Good, because 12-factor app compliant
- Good, because Kubernetes native
- Good, because simple
- Bad, because complex configs are unwieldy
- Bad, because no structure for nested values
- Bad, because difficult to document defaults

### Configuration file only

- Good, because structured and readable
- Good, because version-controllable
- Good, because supports complex nested values
- Bad, because requires file distribution
- Bad, because secrets in files are risky
- Bad, because inflexible for containers

### Hierarchical system

- Good, because flexible for all environments
- Good, because supports secrets via env vars
- Good, because version-controlled defaults via files
- Good, because quick testing via flags
- Bad, because more implementation complexity
- Bad, because precedence must be documented

### Remote configuration service

- Good, because centralized configuration
- Good, because dynamic updates
- Bad, because requires infrastructure
- Bad, because additional failure point
- Bad, because overkill for this project

### Configuration structs in code

- Good, because type-safe
- Good, because compile-time checking
- Bad, because requires recompilation
- Bad, because no flexibility
- Bad, because cannot support multiple environments

## Security Considerations

- **Secrets**: Never in configuration files
- **Environment variables**: Preferred for sensitive values
- **File permissions**: Config files should be readable only by service user
- **Logging**: Sanitize config values in logs (redact secrets)

## Migration Path

For existing deployments:

1. Defaults work out of the box (no changes needed)
2. Add config file for custom settings: `--config config.yaml`
3. Override with environment variables in Kubernetes
4. Use flags for ad-hoc testing

## Links

- Implementation: `pkg/config/config.go` (if created)
- Command-line parsing: `cmd/main.go`
- Example: `example/config.yaml`
- Related: [ADR-0005](0005-api-design.md) - API Design
