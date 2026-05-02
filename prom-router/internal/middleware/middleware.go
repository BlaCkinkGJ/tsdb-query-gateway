package middleware

import (
	"log"

	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/config"
	"github.com/BlaCkinkGJ/query-gateway/prom-router/internal/service"
	"github.com/gin-gonic/gin"
)

// Middleware is a function that wraps a QueryService and returns a new QueryService.
type Middleware func(service.QueryService) service.QueryService

// Factory defines a constructor for a Middleware.
// It accepts configuration and a Gin RouterGroup to optionally bind its own routes.
type Factory func(cfg *config.Config, router *gin.RouterGroup) Middleware

var registry = make(map[string]Factory)

// Register adds a new Middleware factory to the registry.
// This is typically called in an init() function.
func Register(name string, factory Factory) {
	if _, exists := registry[name]; exists {
		log.Fatalf("Middleware factory '%s' is already registered", name)
	}
	registry[name] = factory
}

// Get retrieves a middleware factory from the registry by name.
func Get(name string) Factory {
	return registry[name]
}

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
