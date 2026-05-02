package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Test default
	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("expected no error for default config, got: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if len(cfg.PromEndpoints) != 1 || cfg.PromEndpoints[0] != "http://localhost:9090" {
		t.Errorf("expected default prom endpoint, got %v", cfg.PromEndpoints)
	}

	// Test with file
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := `{"port": 9091, "prom_endpoints": ["http://prom1", "http://prom2"], "ai_endpoint": "http://ai1"}`
	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	cfg, err = LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("expected no error when loading file, got: %v", err)
	}
	if cfg.Port != 9091 {
		t.Errorf("expected port 9091, got %d", cfg.Port)
	}
	if len(cfg.PromEndpoints) != 2 {
		t.Errorf("expected 2 endpoints, got %d", len(cfg.PromEndpoints))
	}
	if cfg.AIEndpoint != "http://ai1" {
		t.Errorf("expected ai endpoint http://ai1, got %s", cfg.AIEndpoint)
	}
}
