# Setup Guide

## Requirements

- Go 1.24 or later
- Docker (optional, for containerized deployment)
- Access to one or more Prometheus instances

## Installation

### From Source

```bash
git clone https://github.com/BlaCkinkGJ/tsdb-query-gateway.git
cd tsdb-query-gateway
make build
```

The binary will be placed at `bin/query-gateway`.

### Using Docker

```bash
docker build -t tsdb-query-gateway .
```

## Running Locally

### Default Configuration

Without a config file, the gateway starts on port `8080` and targets a single Prometheus at `http://localhost:9090`:

```bash
./bin/query-gateway
```

### Custom Configuration

Create a `config.json`:

```json
{
  "port": 8080,
  "prom_endpoints": [
    "http://prom-1:9090",
    "http://prom-2:9090"
  ],
  "middlewares": [
    {
      "name": "ai",
      "config": {
        "endpoint": "http://ai-service:5000"
      }
    }
  ]
}
```

Run with the config file:

```bash
./bin/query-gateway -config config.json
```

## Configuration Validation

On startup, the gateway validates:

- `port` must be greater than 0
- `prom_endpoints` must contain at least one URL
- All declared `middlewares` must have a registered factory

If validation fails, the gateway exits with a fatal error before starting the HTTP server.

## Deploying with Docker Compose

```yaml
version: "3.8"
services:
  gateway:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./config.json:/app/config.json
    command: ["/app/query-gateway", "-config", "/app/config.json"]
    depends_on:
      - prometheus-1
      - prometheus-2
```

## Health Check

The gateway exposes a simple readiness probe at `/api/v1/query?query=up` (standard Prometheus API). A successful response with HTTP 200 indicates the gateway is operational.

> **Warning:** Using a functional query as a readiness probe executes against all downstream Prometheus instances on every probe, generating unnecessary load. Additionally, because the gateway fails fast if any downstream is unavailable, Kubernetes may mark the gateway as unready when a downstream instance is down. Consider implementing a dedicated `/health` or `/ready` endpoint for production deployments.

## Troubleshooting

- **"no prometheus endpoints configured"** — `prom_endpoints` is missing or empty in config
- **"middleware ... not registered"** — Declared middleware name does not match any registered factory. Check `middleware.Register` calls.
- **Downstream timeout** — Use less complex queries. (Note: the gateway's 30s HTTP timeout is hardcoded in `cmd/main.go`; increasing it requires a code change and rebuild.)
