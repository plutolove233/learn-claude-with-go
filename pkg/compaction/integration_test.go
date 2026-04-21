package compaction

import (
	"strings"
	"testing"
	"time"

	"claudego/pkg/types"
)

func TestFullCompactionFlow(t *testing.T) {
	tmpDir := t.TempDir()
	config := types.DefaultCompactionConfig()
	config.SessionStoragePath = tmpDir
	config.AutoCompactTokens = 100

	compactor := &Compactor{
		storage:      &Storage{basePath: tmpDir},
		tokenCounter: NewTokenCounter(),
		config:       config,
	}

	messages := make([]types.Message, 0)
	for i := 0; i < 30; i++ {
		messages = append(messages, types.Message{Role: "user", Content: "This is a test message with some content to increase token count"})
		messages = append(messages, types.Message{Role: "assistant", Content: "This is a response message with some content to increase token count"})
	}

	shouldCompact, tokens, err := compactor.ShouldAutoCompact(messages)
	if err != nil {
		t.Fatalf("Failed to check auto compact: %v", err)
	}
	if !shouldCompact {
		t.Errorf("Expected shouldCompact to be true, tokens: %d", tokens)
	}

	truncated := compactor.FallbackTruncate(messages)
	if len(truncated) >= len(messages) {
		t.Error("Expected truncated messages to be fewer than original")
	}
	if len(truncated) != 5 {
		t.Errorf("Expected 5 messages after fallback truncation, got %d", len(truncated))
	}
}

func TestL1L2Interaction(t *testing.T) {
	tmpDir := t.TempDir()
	config := types.DefaultCompactionConfig()
	config.SessionStoragePath = tmpDir
	config.LargeResultThreshold = 100
	config.OldMessageThreshold = 5

	compactor := &Compactor{
		storage:      &Storage{basePath: tmpDir},
		tokenCounter: NewTokenCounter(),
		config:       config,
	}

	largeContent := strings.Repeat("x", 500)
	results := []types.ToolCallResult{{ToolCallID: "call_1", Name: "bash", Content: largeContent}}

	processed, err := compactor.ProcessToolResults("sess_123", results)
	if err != nil {
		t.Fatalf("Failed to process results: %v", err)
	}
	if !strings.Contains(processed[0].Content, "tool_result_reference") {
		t.Error("Expected L1 to replace with reference")
	}

	messages := make([]types.Message, 0)
	for i := 0; i < 10; i++ {
		messages = append(messages, types.Message{Role: "user", Content: "User message"})
		messages = append(messages, types.Message{Role: "assistant", Content: "Assistant response"})
	}
	messages[2] = types.Message{Role: "user", Content: processed}

	err = compactor.CompactOldMessages(messages, 10)
	if err != nil {
		t.Fatalf("Failed to compact old messages: %v", err)
	}
	results2, ok := messages[2].Content.([]types.ToolCallResult)
	if !ok {
		t.Error("Expected message content to be ToolCallResult")
	}
	if !strings.Contains(results2[0].Content, "tool_result_reference") {
		t.Error("Expected L2 to skip already referenced content")
	}
}

func TestStorageRecovery(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewStorage(tmpDir)

	result := types.StoredToolResult{
		ToolCallID: "call_1",
		ToolName:   "bash",
		Content:    "test output",
		Timestamp:  time.Now(),
	}
	filePath, _ := storage.SaveToolResult("sess_123", result)

	history := types.CompactionHistory{
		SessionID:     "sess_123",
		Timestamp:     time.Now(),
		TokensBefore:  1000,
		TokensAfter:   500,
		MessagesCount: 20,
		SummaryLength: 300,
		FilesStored:   1,
	}
	_ = storage.SaveCompactionHistory("sess_123", history)

	loadedResult, _ := storage.LoadToolResult(filePath)
	if loadedResult.Content != "test output" {
		t.Error("Failed to recover tool result")
	}
	loadedHistory, _ := storage.LoadCompactionHistory("sess_123")
	if loadedHistory.TokensBefore != 1000 {
		t.Error("Failed to recover compaction history")
	}
}
