package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type MiddlewareConfig struct {
	Name   string          `json:"name"`
	Config json.RawMessage `json:"config,omitempty"`
}

type Config struct {
	Port          int                `json:"port"`
	PromEndpoints []string           `json:"prom_endpoints"`
	Middlewares   []MiddlewareConfig `json:"middlewares,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	// Provide a default config if no file is provided
	if path == "" {
		return &Config{
			Port: 8080,
			PromEndpoints: []string{
				"http://localhost:9090",
			},
		}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Port <= 0 {
		return nil, fmt.Errorf("invalid port: %d", cfg.Port)
	}
	if len(cfg.PromEndpoints) == 0 {
		return nil, fmt.Errorf("prom_endpoints cannot be empty")
	}

	return &cfg, nil
}
