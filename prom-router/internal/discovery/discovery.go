package discovery

import (
	"fmt"
	"sync"
)

// Registry defines a service locator interface adhering to SOLID principles.
// It allows different layers (handlers, middlewares, services) to decouple
// and locate their dependencies at runtime.
type Registry interface {
	Register(name string, service interface{})
	Get(name string) (interface{}, error)
}

type syncRegistry struct {
	mu       sync.RWMutex
	services map[string]interface{}
}

// NewRegistry creates a new concurrency-safe service registry.
func NewRegistry() Registry {
	return &syncRegistry{
		services: make(map[string]interface{}),
	}
}

func (r *syncRegistry) Register(name string, service interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[name] = service
}

func (r *syncRegistry) Get(name string) (interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if svc, exists := r.services[name]; exists {
		return svc, nil
	}
	return nil, fmt.Errorf("service not found: %s", name)
}
