package compaction

import (
	"claudego/pkg/types"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Storage manages on-disk persistence for compaction artifacts.
type Storage struct {
	basePath string
}

func NewStorage(basePath string) (*Storage, error) {
	if strings.HasPrefix(basePath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		basePath = filepath.Join(home, strings.TrimPrefix(basePath, "~/"))
	}
	return &Storage{basePath: basePath}, nil
}

func (s *Storage) SaveToolResult(sessionID string, result types.StoredToolResult) (string, error) {
	sessionDir := filepath.Join(s.basePath, sessionID, "tool-results")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create session directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	sanitizedToolName := strings.ReplaceAll(result.ToolName, string(os.PathSeparator), "_")
	if sanitizedToolName == "" {
		sanitizedToolName = "tool"
	}
	filename := fmt.Sprintf("%s-%s.json", timestamp, sanitizedToolName)
	filePath := filepath.Join(sessionDir, filename)

	counter := 1
	for {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			break
		}
		filePath = filepath.Join(sessionDir, fmt.Sprintf("%s-%s-%d.json", timestamp, sanitizedToolName, counter))
		counter++
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	return filePath, nil
}

func (s *Storage) LoadToolResult(filePath string) (*types.StoredToolResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	var result types.StoredToolResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	return &result, nil
}

func (s *Storage) GetSessionDir(sessionID string) string {
	return filepath.Join(s.basePath, sessionID)
}

func (s *Storage) GetToolResultsDir(sessionID string) string {
	return filepath.Join(s.basePath, sessionID, "tool-results")
}

func (s *Storage) ListToolResults(sessionID string) ([]string, error) {
	dir := s.GetToolResultsDir(sessionID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".json" {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}

func (s *Storage) SaveCompactionHistory(sessionID string, history types.CompactionHistory) error {
	sessionDir := filepath.Join(s.basePath, sessionID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}
	historyPath := filepath.Join(sessionDir, "compaction-history.json")
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}
	if err := os.WriteFile(historyPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}
	return nil
}

func (s *Storage) LoadCompactionHistory(sessionID string) (*types.CompactionHistory, error) {
	historyPath := filepath.Join(s.basePath, sessionID, "compaction-history.json")
	data, err := os.ReadFile(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read history file: %w", err)
	}

	var history types.CompactionHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("failed to unmarshal history: %w", err)
	}
	return &history, nil
}
