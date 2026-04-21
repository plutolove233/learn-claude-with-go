package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"claudego/pkg/types"
)

type Config struct {
	APIKey           string                  `json:"api_key"`
	BaseURL          string                  `json:"base_url"`
	Model            string                  `json:"model"`
	CompactionConfig *types.CompactionConfig `json:"compaction_config,omitempty"`
}

func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(home, ".claudego", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.CompactionConfig == nil {
		cfg.CompactionConfig = types.DefaultCompactionConfig()
	}
	return &cfg, nil
}

func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// 降级到当前目录
		return "./.claudego/config.json"
	}
	return filepath.Join(home, ".claudego", "config.json")
}
