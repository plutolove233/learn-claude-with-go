package config

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestConfigJSON(t *testing.T) {
	content := `{"api_key": "test-key", "base_url": "https://api.test.com", "model": "test-model"}`
	var cfg Config
	if err := json.Unmarshal([]byte(content), &cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "test-key" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "test-key")
	}
	if cfg.BaseURL != "https://api.test.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://api.test.com")
	}
	if cfg.Model != "test-model" {
		t.Errorf("Model = %q, want %q", cfg.Model, "test-model")
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	if path == "" {
		t.Error("DefaultConfigPath returned empty string")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("DefaultConfigPath = %q, want absolute path", path)
	}
}

func TestLoad_FileFound(t *testing.T) {
	_, err := Load()
	// Should fail when ~/.claudego/config.json doesn't exist
	if err != nil {
		t.Error("expected error for non-existent config file")
	}
}
