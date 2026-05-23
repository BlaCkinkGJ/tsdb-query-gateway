# Configuration

## Configuration File

TSDB Query Gateway accepts a JSON configuration file passed via the `-config` flag.

```bash
./bin/tsdb-query-gateway -config config.json
```

## Schema

```json
{
  "port": 8080,
  "prom_endpoints": [
    "http://localhost:9090",
    "http://prom-2:9090"
  ],
  "middlewares": [
    {
      "name": "ai",
      "config": { "endpoint": "http://ai-service:5000" }
    }
  ]
}
```

### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `port` | int | Yes* | `8080` | HTTP server port |
| `prom_endpoints` | []string | Yes* | `["http://localhost:9090"]` | List of Prometheus base URLs |
| `middlewares` | []object | No | `[]` | List of middleware configurations |

\* Required only when providing a config file. Defaults apply when running without `-config`.

### Middleware Config

Each middleware entry has:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Middleware factory name (must be registered) |
| `config` | object | No | Middleware-specific configuration (opaque to core) |

## Environment Variables

The gateway does not natively read from environment variables. For Docker or Kubernetes deployments, use a config file mounted as a volume or generate `config.json` from a ConfigMap.

### Kubernetes ConfigMap Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: gateway-config
data:
  config.json: |
    {
      "port": 8080,
      "prom_endpoints": [
        "http://prometheus-0:9090",
        "http://prometheus-1:9090"
      ]
    }
```

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tsdb-query-gateway
spec:
  template:
    spec:
      containers:
        - name: gateway
          image: tsdb-query-gateway:latest
          args: ["-config", "/app/config.json"]
          volumeMounts:
            - name: config
              mountPath: /app
      volumes:
        - name: config
          configMap:
            name: gateway-config
```

## Validation Rules

On startup, the following validations are enforced:

1. `port > 0` — Invalid port causes fatal error.
2. `len(prom_endpoints) > 0` — At least one downstream Prometheus is required.
3. All `middlewares[*].name` must have a registered factory — Unknown middleware causes fatal error.

## Default Behavior

Running without `-config`:

```bash
./bin/tsdb-query-gateway
```

Defaults to:
- Port: `8080`
- Prometheus: `http://localhost:9090`
- Middlewares: none
