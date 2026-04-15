package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigJSON(t *testing.T) {
	content := `{"api_key": "test-key", "base_url": "https://api.test.com", "model": "test-model"}`
	var cfg Config
	err := json.Unmarshal([]byte(content), &cfg)
	require.NoError(t, err)
	assert.Equal(t, "test-key", cfg.APIKey)
	assert.Equal(t, "https://api.test.com", cfg.BaseURL)
	assert.Equal(t, "test-model", cfg.Model)
}

func TestDefaultConfigPath_NormalCase(t *testing.T) {
	path := DefaultConfigPath()

	assert.NotEmpty(t, path, "DefaultConfigPath should not return empty string")
	assert.True(t, filepath.IsAbs(path), "DefaultConfigPath should return absolute path")
	assert.Contains(t, path, ".claudego", "path should contain .claudego directory")
	assert.Contains(t, path, "config.json", "path should contain config.json filename")
}

func TestDefaultConfigPath_HomeEnvNotSet(t *testing.T) {
	// Save original HOME value
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
	}()

	// Unset HOME to simulate the fallback scenario
	os.Unsetenv("HOME")

	path := DefaultConfigPath()

	assert.NotEmpty(t, path, "DefaultConfigPath should not return empty string even without HOME")
	assert.Equal(t, "./.claudego/config.json", path, "should fallback to current directory when HOME is not available")
}

func TestLoad_Success(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".claudego")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "config.json")
	validConfig := `{
		"api_key": "sk-test-key-12345",
		"base_url": "https://api.anthropic.com",
		"model": "claude-3-sonnet-20240229"
	}`
	err = os.WriteFile(configPath, []byte(validConfig), 0644)
	require.NoError(t, err)

	// Temporarily override HOME to point to tmpDir
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Test Load function
	cfg, err := Load()

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "sk-test-key-12345", cfg.APIKey)
	assert.Equal(t, "https://api.anthropic.com", cfg.BaseURL)
	assert.Equal(t, "claude-3-sonnet-20240229", cfg.Model)
}

func TestLoad_FileNotExists(t *testing.T) {
	// Create temporary directory without config file
	tmpDir := t.TempDir()

	// Temporarily override HOME to point to tmpDir
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Test Load function
	cfg, err := Load()

	assert.Error(t, err, "Load should return error when config file does not exist")
	assert.Nil(t, cfg, "Config should be nil when file does not exist")
	assert.True(t, os.IsNotExist(err), "error should indicate file does not exist")
}

func TestLoad_InvalidJSON(t *testing.T) {
	// Create temporary config file with invalid JSON
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".claudego")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "config.json")
	invalidJSON := `{
		"api_key": "test-key",
		"base_url": "https://api.test.com"
		"model": "test-model"
	}`
	err = os.WriteFile(configPath, []byte(invalidJSON), 0644)
	require.NoError(t, err)

	// Temporarily override HOME to point to tmpDir
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Test Load function
	cfg, err := Load()

	assert.Error(t, err, "Load should return error for invalid JSON")
	assert.Nil(t, cfg, "Config should be nil when JSON is invalid")
}

func TestLoad_EmptyJSON(t *testing.T) {
	// Create temporary config file with empty JSON object
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".claudego")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "config.json")
	emptyJSON := `{}`
	err = os.WriteFile(configPath, []byte(emptyJSON), 0644)
	require.NoError(t, err)

	// Temporarily override HOME to point to tmpDir
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Test Load function
	cfg, err := Load()

	require.NoError(t, err, "Load should succeed with empty JSON object")
	require.NotNil(t, cfg)
	assert.Empty(t, cfg.APIKey, "APIKey should be empty")
	assert.Empty(t, cfg.BaseURL, "BaseURL should be empty")
	assert.Empty(t, cfg.Model, "Model should be empty")
}

func TestLoad_PartialConfig(t *testing.T) {
	// Create temporary config file with partial configuration
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".claudego")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	configPath := filepath.Join(configDir, "config.json")
	partialConfig := `{"api_key": "test-key-only"}`
	err = os.WriteFile(configPath, []byte(partialConfig), 0644)
	require.NoError(t, err)

	// Temporarily override HOME to point to tmpDir
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Test Load function
	cfg, err := Load()

	require.NoError(t, err, "Load should succeed with partial config")
	require.NotNil(t, cfg)
	assert.Equal(t, "test-key-only", cfg.APIKey)
	assert.Empty(t, cfg.BaseURL, "BaseURL should be empty when not provided")
	assert.Empty(t, cfg.Model, "Model should be empty when not provided")
}
