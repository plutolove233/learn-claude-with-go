package compaction

import (
	"claudego/pkg/types"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStorage(t *testing.T) {
	tmpDir := t.TempDir()
	storage, err := NewStorage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	if storage.basePath != tmpDir {
		t.Errorf("Expected basePath to be %s, got %s", tmpDir, storage.basePath)
	}
}

func TestSaveToolResult(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewStorage(tmpDir)

	result := types.StoredToolResult{
		ToolCallID: "call_123",
		ToolName:   "bash",
		Arguments:  "ls -la",
		Content:    "file1.txt\\nfile2.txt",
		Timestamp:  time.Now(),
	}

	filePath, err := storage.SaveToolResult("sess_123", result)
	if err != nil {
		t.Fatalf("Failed to save tool result: %v", err)
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("Expected file to exist at %s", filePath)
	}
}

func TestLoadToolResult(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewStorage(tmpDir)

	original := types.StoredToolResult{
		ToolCallID: "call_123",
		ToolName:   "bash",
		Arguments:  "ls -la",
		Content:    "file1.txt\\nfile2.txt",
		Timestamp:  time.Now(),
	}

	filePath, _ := storage.SaveToolResult("sess_123", original)
	loaded, err := storage.LoadToolResult(filePath)
	if err != nil {
		t.Fatalf("Failed to load tool result: %v", err)
	}
	if loaded.ToolCallID != original.ToolCallID {
		t.Errorf("Expected ToolCallID to be %s, got %s", original.ToolCallID, loaded.ToolCallID)
	}
	if loaded.Content != original.Content {
		t.Errorf("Expected Content to match")
	}
}

func TestListToolResults(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewStorage(tmpDir)

	for i := 1; i <= 2; i++ {
		result := types.StoredToolResult{
			ToolCallID: fmt.Sprintf("call_%d", i),
			ToolName:   "bash",
			Content:    "output",
			Timestamp:  time.Now(),
		}
		_, _ = storage.SaveToolResult("sess_123", result)
	}

	files, err := storage.ListToolResults("sess_123")
	if err != nil {
		t.Fatalf("Failed to list tool results: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
}

func TestSaveCompactionHistory(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewStorage(tmpDir)

	history := types.CompactionHistory{
		SessionID:     "sess_123",
		Timestamp:     time.Now(),
		TokensBefore:  8500,
		TokensAfter:   4000,
		MessagesCount: 20,
		SummaryLength: 500,
		FilesStored:   5,
	}

	err := storage.SaveCompactionHistory("sess_123", history)
	if err != nil {
		t.Fatalf("Failed to save compaction history: %v", err)
	}

	historyPath := filepath.Join(tmpDir, "sess_123", "compaction-history.json")
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		t.Errorf("Expected history file to exist at %s", historyPath)
	}
}

func TestLoadCompactionHistory(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewStorage(tmpDir)

	original := types.CompactionHistory{
		SessionID:     "sess_123",
		Timestamp:     time.Now(),
		TokensBefore:  8500,
		TokensAfter:   4000,
		MessagesCount: 20,
		SummaryLength: 500,
		FilesStored:   5,
	}

	_ = storage.SaveCompactionHistory("sess_123", original)
	loaded, err := storage.LoadCompactionHistory("sess_123")
	if err != nil {
		t.Fatalf("Failed to load compaction history: %v", err)
	}
	if loaded.SessionID != original.SessionID {
		t.Errorf("Expected SessionID to be %s, got %s", original.SessionID, loaded.SessionID)
	}
	if loaded.TokensBefore != original.TokensBefore {
		t.Errorf("Expected TokensBefore to be %d, got %d", original.TokensBefore, loaded.TokensBefore)
	}
}
