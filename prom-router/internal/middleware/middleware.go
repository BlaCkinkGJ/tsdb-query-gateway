package middleware

import (
	"fmt"

	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/config"
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/service"
	"github.com/gin-gonic/gin"
)

// Middleware defines an interface for components that can intercept QueryService requests.
type Middleware interface {
	Wrap(next service.QueryService) service.QueryService
}

// Factory defines a constructor for a Middleware.
// It accepts its specific config block and a Gin RouterGroup.
type Factory func(mwConfig config.MiddlewareConfig, router *gin.RouterGroup) Middleware

var registry = make(map[string]Factory)

// Register adds a new Middleware factory to the registry.
// This is typically called in an init() function.
func Register(name string, factory Factory) error {
	if _, exists := registry[name]; exists {
		return fmt.Errorf("middleware factory %q already registered", name)
	}
	registry[name] = factory
	return nil
}

// Get retrieves a middleware factory from the registry by name.
func Get(name string) Factory {
	return registry[name]
}

// Chain applies a series of middlewares to a base QueryService in daisy-chain fashion.
// For example: Chain(base, m1, m2) will result in m1.Wrap(m2.Wrap(base)).
// So when a request comes in, it goes m1 -> m2 -> base -> m2 -> m1.
func Chain(base service.QueryService, middlewares ...Middleware) service.QueryService {
	result := base
	// Apply middlewares in reverse order so the first one in the list is the outermost layer.
	for i := len(middlewares) - 1; i >= 0; i-- {
		result = middlewares[i].Wrap(result)
	}
	return result
}
