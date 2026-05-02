package middleware

import (
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/service"
)

// Middleware is a function that wraps a QueryService and returns a new QueryService.
type Middleware func(service.QueryService) service.QueryService

// Chain applies a series of middlewares to a base QueryService in daisy-chain fashion.
// For example: Chain(base, m1, m2) will result in m1(m2(base)).
// So when a request comes in, it goes m1 -> m2 -> base -> m2 -> m1.
func Chain(base service.QueryService, middlewares ...Middleware) service.QueryService {
	result := base
	// Apply middlewares in reverse order so the first one in the list is the outermost layer.
	for i := len(middlewares) - 1; i >= 0; i-- {
		result = middlewares[i](result)
	}
	return result
}
