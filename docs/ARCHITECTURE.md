# Architecture

## Overview

TSDB Query Gateway follows a **layered architecture** with clean separation between transport, service, and client layers. Middleware wraps the service layer in a **daisy chain** pattern, allowing extensible query interception.

## Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        HTTP Client                           │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  Gin Router  ──►  QueryHandler  ──►  Middleware Chain       │
│                                     (QueryService interface)  │
└───────────────────────────┬─────────────────────────────────┘
                            │
              ┌─────────────┴─────────────┐
              ▼                           ▼
┌─────────────────────┐      ┌─────────────────────┐
│   AI Middleware     │      │ Statistical MW      │
│   (placeholder)     │      │ (placeholder)       │
└─────────┬───────────┘      └─────────┬───────────┘
          │                            │
          └─────────────┬──────────────┘
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                  GatewayService (core)                       │
│  • Parallel query execution (errgroup)                       │
│  • Result merging (vector/matrix/scalar/string)              │
│  • Warning aggregation & deduplication                       │
└───────────────────────────┬─────────────────────────────────┘
                            │
              ┌─────────────┴─────────────┐
              ▼                           ▼
┌─────────────────────┐      ┌─────────────────────┐
│ PrometheusClient    │      │ PrometheusClient    │
│ (endpoint A)        │      │ (endpoint B)        │
└─────────────────────┘      └─────────────────────┘
```

## Key Design Decisions

### 1. QueryService Interface

All components implement `QueryService`:

```go
type QueryService interface {
    Query(ctx context.Context, req models.PromQueryRequest) (*models.PromResponse, error)
    QueryRange(ctx context.Context, req models.PromQueryRangeRequest) (*models.PromResponse, error)
}
```

This enables:
- **Middleware wrapping** — each middleware wraps the next `QueryService` in the chain
- **Testability** — mock `QueryService` for unit tests without real HTTP calls
- **Swappability** — replace the core `GatewayService` with a different implementation

### 2. Parallel Execution

`GatewayService.Query` and `QueryRange` use `errgroup.WithContext` to fan out requests to all configured Prometheus endpoints concurrently. If any request fails, the entire operation fails fast.

### 3. Result Type Handling

Prometheus returns four result types, handled differently:

| Type | Structure | Merge Strategy |
|------|-----------|----------------|
| `vector` | `[]metric` | Concatenate across endpoints |
| `matrix` | `[]series` | Concatenate across endpoints |
| `scalar` | `[timestamp, value]` | Return first valid result only |
| `string` | `[timestamp, value]` | Return first valid result only |

Unknown types are rejected with an error.

### 4. Middleware Registration

Middleware factories are registered at runtime via `middleware.Register(name, factory)`. This allows:
- Compiled-in middlewares (AI, statistical analysis)
- Future hot-pluggable middlewares without modifying core code

### 5. Error Propagation

The `PrometheusClient` parses structured Prometheus API errors into `client.DownstreamError`, which carries:
- `StatusCode` — original HTTP status (e.g., 400 for `bad_data`)
- `ErrorType` — Prometheus error type (e.g., `bad_data`, `timeout`)
- `Message` — human-readable error message

The `QueryHandler` uses `errors.As` to detect downstream errors and propagate the correct HTTP status code and error payload to the client.

## Data Flow

1. Client sends `/api/v1/query?query=up`
2. `QueryHandler.HandleQuery` builds `PromQueryRequest`
3. Middleware chain processes request (e.g., AI enrichment)
4. `GatewayService.Query` fans out to all Prometheus endpoints
5. `PrometheusClient` executes HTTP GET against each endpoint
6. `mergeResults` merges responses based on `resultType`
7. Merged `PromResponse` flows back through middleware chain
8. `QueryHandler` returns JSON response to client
