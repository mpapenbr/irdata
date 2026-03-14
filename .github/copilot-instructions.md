# iRData Copilot Instructions

## Project Overview

`irdata` is a Go CLI application for fetching and caching data from the iRacing API. It's structured as a Cobra-based CLI with sophisticated observability, authentication, and caching capabilities.

## Architecture & Key Components

### Core Application Structure

- **Entry Point**: [main.go](main.go) → [cmd/root.go](cmd/root.go) using Cobra CLI framework
- **Commands**: Modular subcommands in [cmd/](cmd/) - `auth`, `populate`, `check`, `config`
- **Core API Client**: [irdata/irdata.go](irdata/irdata.go) - Main client for iRacing API interactions
- **Authentication**: [auth/auth.go](auth/auth.go) - Token management with refresh logic
- **Caching**: [cache/](cache/) - Interface-based design with Badger and NoOp implementations

### Critical Data Types

Racing domain objects in [irdata/](irdata/):

- `EventResult` - Race session results and driver data
- `Schedule` - Season scheduling information
- `Series` - Racing series metadata
- Focus on iRacing-specific fields like `SubsessionID`, `CustID`, license levels

### Configuration Patterns

- **Viper + Environment Variables**: Use `irdata` prefix for env vars (see [cmd/root.go](cmd/root.go))
- **Global Config**: [cmd/config/config.go](cmd/config/config.go) holds runtime configuration
- **Structured Options**: Functional options pattern throughout (e.g., `WithTokenProvider()`, `WithCache()`)

## Development Workflow

### Essential Make Commands

```bash
make install          # Fetch dependencies
make codestyle        # Run golangci-lint --fix
make lint            # Check code style and run linters
make test            # Run all tests
make fast-tests      # Run tests marked as fast
make slow-tests      # Run tests marked as slow
make run q="--help"  # Run with arguments
```

### Test Organization

Tests are categorized by execution time using `TEST_MODE` environment variable:

- Fast tests for unit logic
- Slow tests for integration/API calls
- Use `make test-suite` for complete validation

### Dev Container Setup

- Uses Ubuntu base with Go tools pre-installed
- Configured for golangci-lint, gofumpt formatting
- Pre-installed extensions: Go, GitLens, GitHub Actions
- Network host mode for API access during development

## Code Patterns & Conventions

### Error Handling

- Always use structured logging: `log.Error("message", log.ErrorField(err))`
- Context propagation throughout call chains
- Custom errors for domain-specific cases (e.g., `ErrNoTokenProvider`)

### Observability Setup

- **Telemetry**: Full OpenTelemetry integration in [otel/otel_setup.go](otel/otel_setup.go)
- **Logging**: Zap-based structured logging in [log/](log/) with OTEL bridge
- **Configuration**: Telemetry toggleable via `--enable-telemetry` flag
- **Development**: Use stdout exporters, production uses GRPC

### HTTP Client Patterns

- **Retryable HTTP**: Uses `hashicorp/go-retryablehttp` for API reliability
- **Rate Limiting**: Built-in rate limit handling in API client
- **S3 Integration**: Separate client for downloading race data files
- **Authentication**: Token injection via functional options

### Interface-Driven Design

Key interfaces to implement/extend:

```go
type Cache interface {
    Get(key string) ([]byte, bool)
    Set(key string, value []byte) error
    Delete(key string) error
}

type TokenProvider func() (string, error)
```

## Build & Release

### Multi-Platform Builds

Uses goreleaser for cross-platform binaries:

```bash
goreleaser build    # Generate binaries for Linux/Windows/Mac
goreleaser release  # Create Docker images and GitHub releases
```

### Directory Structure Conventions

- [cmd/](cmd/) - CLI command implementations
- [irdata/](irdata/) - Core domain logic and API client
- [tmp/](tmp/) - Runtime data files (JSON responses, CSVs)
- [cache/](cache/) - Pluggable caching implementations
- [bin/](bin/) - Built binaries (gitignored)

## Integration Points

### iRacing API Specifics

- **Base URL**: `https://members-ng.iracing.com/data`
- **Authentication**: OAuth2 with refresh token handling
- **Rate Limiting**: Respect API rate limits with backoff
- **Data Formats**: JSON responses, S3-hosted race result files

### External Dependencies

- **Badger v4**: Embedded K/V store for caching
- **Viper**: Configuration management
- **Cobra**: CLI framework
- **OTEL**: Complete observability stack
- **Zap**: High-performance logging

When adding new commands, follow the pattern in [cmd/populate/](cmd/populate/) with separate files per data type and shared utilities in `common.go`.
