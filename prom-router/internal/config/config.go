package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Port          int                      `json:"port"`
	PromEndpoints []string                 `json:"prom_endpoints"`
	Middlewares   []map[string]interface{} `json:"middlewares,omitempty"`
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

	return &cfg, nil
}
