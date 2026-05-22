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
	var cfg Config

	// Provide a default config if no file is provided
	if path == "" {
		cfg = Config{
			Port: 8080,
			PromEndpoints: []string{
				"http://localhost:9090",
			},
		}
	} else {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	}

	if cfg.Port <= 0 {
		return nil, fmt.Errorf("invalid port: %d", cfg.Port)
	}
	if len(cfg.PromEndpoints) == 0 {
		return nil, fmt.Errorf("prom_endpoints cannot be empty")
	}

	return &cfg, nil
}
