# TSDB Query Gateway

A lightweight HTTP gateway service that routes and merges Prometheus queries across multiple downstream Prometheus instances. It provides a single endpoint for querying distributed time-series databases while supporting a pluggable middleware system for extensibility.

[![Go](https://img.shields.io/badge/go-1.24-blue)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## What is this?

TSDB Query Gateway acts as a **query federation layer** in front of multiple Prometheus endpoints. Instead of querying each Prometheus instance individually, clients send queries to the gateway, which:

1. **Routes** the query to all configured Prometheus endpoints in parallel
2. **Merges** the results intelligently based on Prometheus result types (`vector`, `matrix`, `scalar`, `string`)
3. **Propagates** warnings and structured errors from downstream instances
4. **Supports middleware** for custom query processing, analytics, or AI-powered enhancements

## Features

- **Parallel Query Execution** — queries all downstream Prometheus instances concurrently using `errgroup`
- **Smart Result Merging** — concatenates `vector`/`matrix` results, preserves `scalar`/`string` as-is
- **Warning Aggregation** — collects and deduplicates warnings from all downstream responses
- **Structured Error Propagation** — forwards Prometheus API errors (`errorType`, `error`) with original HTTP status codes
- **Pluggable Middleware** — daisy-chain middleware for custom query interception

> **⚠️ Experimental:** This project is currently experimental and not recommended for production use. APIs, configuration formats, and behavior may change without notice.

## Quick Start

```bash
# Clone and build
git clone https://github.com/BlaCkinkGJ/tsdb-query-gateway.git
cd tsdb-query-gateway
go build -o bin/tsdb-query-gateway ./cmd

# Run with default config (port 8080, localhost:9090)
./bin/tsdb-query-gateway

# Or with a custom config
./bin/tsdb-query-gateway -config config.json
```

## API

### Instant Query
```bash
curl "http://localhost:8080/api/v1/query?query=up"
```

### Range Query
```bash
curl "http://localhost:8080/api/v1/query_range?query=up&start=1716000000&end=1716003600&step=15s"
```

## Documentation

| Document | Description |
|----------|-------------|
| [docs/SETUP.md](docs/SETUP.md) | Installation, configuration, and deployment guide |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System architecture, component diagrams, and design decisions |
| [docs/API.md](docs/API.md) | Full API reference with request/response examples |
| [docs/CONFIGURATION.md](docs/CONFIGURATION.md) | Configuration schema, examples, and environment variables |
| [docs/MIDDLEWARE.md](docs/MIDDLEWARE.md) | How to write and register custom middleware |

## Project Structure

```
.
├── cmd/                    # Application entry point
│   └── main.go
├── internal/               # Internal packages (not importable externally)
│   ├── client/             # Prometheus HTTP client
│   ├── config/             # Configuration loading and validation
│   ├── handler/            # HTTP request handlers (Gin)
│   ├── middleware/         # Middleware system + built-in middlewares
│   └── service/            # Core query routing and merging logic
├── pkg/models/             # Public data models (importable externally)
├── Dockerfile              # Multi-stage Docker build
├── Makefile                # Build automation
└── go.mod                  # Go module definition
```

## Development

```bash
make all      # Format, vet, build, and test
make test     # Run tests with race detection
make build    # Build binary
make run      # Build and run locally
```

## Docker

```bash
docker build -t tsdb-query-gateway .
docker run -p 8080:8080 -v $(pwd)/config.json:/app/config.json tsdb-query-gateway -config /app/config.json
```

## License

MIT
