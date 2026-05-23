# Middleware

## Overview

The middleware system allows you to intercept and modify queries before they reach the core `GatewayService`, and optionally transform responses on the way back. Middlewares are chained in a **daisy chain** using the decorator pattern.

## Middleware Interface

```go
package middleware

type Middleware interface {
    Wrap(next service.QueryService) service.QueryService
}
```

A middleware receives the next `QueryService` in the chain and returns a wrapped version. The outermost middleware is called first, then the request flows inward to the core `GatewayService`, and the response flows back out through each middleware.

## Built-in Middlewares

### AI Middleware (`ai`)

Placeholder for AI-powered query enhancement. Currently a no-op but structurally ready for integration with LLM-based query rewriting or anomaly detection.

### Statistical Middleware (`statistical`)

Placeholder for query analytics and metrics collection. Can be extended to emit Prometheus metrics about query latency, error rates, or downstream distribution.

## Writing Custom Middleware

1. Implement the `Middleware` interface:

```go
package mymiddleware

import (
    "context"
    "log"

    "github.com/BlaCkinkGJ/tsdb-query-gateway/internal/service"
    "github.com/BlaCkinkGJ/tsdb-query-gateway/pkg/models"
)

type loggingMiddleware struct {
    next service.QueryService
}

func (m *loggingMiddleware) Query(ctx context.Context, req models.PromQueryRequest) (*models.PromResponse, error) {
    log.Printf("query: %s", req.Query)
    return m.next.Query(ctx, req)
}

func (m *loggingMiddleware) QueryRange(ctx context.Context, req models.PromQueryRangeRequest) (*models.PromResponse, error) {
    log.Printf("query_range: %s", req.Query)
    return m.next.QueryRange(ctx, req)
}

type LoggingMiddleware struct{}

func (LoggingMiddleware) Wrap(next service.QueryService) service.QueryService {
    return &loggingMiddleware{next: next}
}
```

2. Register the factory in `init()`:

```go
import (
    "github.com/BlaCkinkGJ/tsdb-query-gateway/internal/config"
    "github.com/BlaCkinkGJ/tsdb-query-gateway/internal/middleware"
    "github.com/gin-gonic/gin"
)

func init() {
    middleware.Register("logging", func(mwConfig config.MiddlewareConfig, router *gin.RouterGroup) middleware.Middleware {
        return LoggingMiddleware{}
    })
}
```

3. Add to `config.json`:

```json
{
  "middlewares": [
    { "name": "logging" }
  ]
}
```

## Registration Rules

- Factory names must be unique. Registering twice returns an error.
- The factory receives the middleware-specific config block and a `gin.RouterGroup` for optional custom route registration.
- Unregistered middleware names in config cause a fatal error at startup.

## Chain Behavior

```go
finalService := middleware.Chain(baseGatewayService, m1, m2, m3)
```

Execution order:
1. `m1.Wrap(...)` called first (outermost)
2. `m2.Wrap(...)`
3. `m3.Wrap(...)`
4. `baseGatewayService` (innermost)

Request flow for a query:
```
m1.Query() -> m2.Query() -> m3.Query() -> GatewayService.Query()
m1 returns <- m2 returns <- m3 returns <- GatewayService returns
```

## Testing Middleware

Mock the next `QueryService` and verify that your middleware correctly delegates:

```go
type mockService struct{}

func (m *mockService) Query(ctx context.Context, req models.PromQueryRequest) (*models.PromResponse, error) {
    return &models.PromResponse{Status: "success"}, nil
}

func (m *mockService) QueryRange(ctx context.Context, req models.PromQueryRangeRequest) (*models.PromResponse, error) {
    return &models.PromResponse{Status: "success"}, nil
}

func TestLoggingMiddleware(t *testing.T) {
    base := &mockService{}
    mw := LoggingMiddleware{}
    wrapped := mw.Wrap(base)
    res, err := wrapped.Query(context.Background(), models.PromQueryRequest{Query: "up"})
    // Assert response and side effects
}
```
