package compaction

import (
	"fmt"
	"strings"
	"testing"

	"claudego/pkg/types"
)

func TestProcessToolResults_SmallResult(t *testing.T) {
	tmpDir := t.TempDir()
	config := types.DefaultCompactionConfig()
	config.SessionStoragePath = tmpDir
	config.LargeResultThreshold = 100

	compactor := &Compactor{
		storage:      &Storage{basePath: tmpDir},
		tokenCounter: NewTokenCounter(),
		config:       config,
	}

	results := []types.ToolCallResult{{
		ToolCallID: "call_1",
		Name:       "bash",
		Content:    "small output",
	}}

	processed, err := compactor.ProcessToolResults("sess_123", results)
	if err != nil {
		t.Fatalf("Failed to process results: %v", err)
	}
	if len(processed) != 1 {
		t.Errorf("Expected 1 result, got %d", len(processed))
	}
	if processed[0].Content != "small output" {
		t.Errorf("Expected content to be unchanged for small result")
	}
}

func TestProcessToolResults_LargeResult(t *testing.T) {
	tmpDir := t.TempDir()
	config := types.DefaultCompactionConfig()
	config.SessionStoragePath = tmpDir
	config.LargeResultThreshold = 100

	compactor := &Compactor{
		storage:      &Storage{basePath: tmpDir},
		tokenCounter: NewTokenCounter(),
		config:       config,
	}

	largeContent := strings.Repeat("x", 500)
	results := []types.ToolCallResult{{
		ToolCallID: "call_1",
		Name:       "bash",
		Content:    largeContent,
	}}

	processed, err := compactor.ProcessToolResults("sess_123", results)
	if err != nil {
		t.Fatalf("Failed to process results: %v", err)
	}
	if len(processed) != 1 {
		t.Errorf("Expected 1 result, got %d", len(processed))
	}
	if processed[0].Content == largeContent {
		t.Error("Expected content to be replaced with reference")
	}
	if !strings.Contains(processed[0].Content, "tool_result_reference") {
		t.Error("Expected content to contain tool_result_reference marker")
	}
}

func TestCompactOldMessages_BelowThreshold(t *testing.T) {
	config := types.DefaultCompactionConfig()
	compactor := &Compactor{config: config}
	messages := []types.Message{{Role: "user", Content: "Hello"}, {Role: "assistant", Content: "Hi"}}

	err := compactor.CompactOldMessages(messages, 10)
	if err != nil {
		t.Fatalf("Failed to compact messages: %v", err)
	}
	if messages[0].Content != "Hello" {
		t.Error("Expected message to be unchanged when below threshold")
	}
}

func TestCompactOldMessages_AboveThreshold(t *testing.T) {
	config := types.DefaultCompactionConfig()
	compactor := &Compactor{config: config}

	messages := make([]types.Message, 0)
	for i := 0; i < 25; i++ {
		messages = append(messages, types.Message{Role: "user", Content: "User message"})
		messages = append(messages, types.Message{Role: "assistant", Content: "Assistant response"})
	}

	largeContent := strings.Repeat("x", 500)
	messages[2] = types.Message{
		Role: "user",
		Content: []types.ToolCallResult{{
			ToolCallID: "call_1",
			Name:       "bash",
			Content:    largeContent,
		}},
	}

	err := compactor.CompactOldMessages(messages, 25)
	if err != nil {
		t.Fatalf("Failed to compact messages: %v", err)
	}

	results, ok := messages[2].Content.([]types.ToolCallResult)
	if !ok {
		t.Error("Expected message content to be ToolCallResult")
	}
	if len(results[0].Content) >= 500 {
		t.Error("Expected content to be truncated")
	}
	if !strings.Contains(results[0].Content, "截断") {
		t.Error("Expected truncation marker in content")
	}
}

func TestShouldAutoCompact_BelowThreshold(t *testing.T) {
	config := types.DefaultCompactionConfig()
	compactor := &Compactor{config: config, tokenCounter: NewTokenCounter()}
	messages := []types.Message{{Role: "user", Content: "Hello"}, {Role: "assistant", Content: "Hi"}}

	shouldCompact, tokens, err := compactor.ShouldAutoCompact(messages)
	if err != nil {
		t.Fatalf("Failed to check auto compact: %v", err)
	}
	if shouldCompact {
		t.Error("Expected shouldCompact to be false for small messages")
	}
	if tokens <= 0 {
		t.Errorf("Expected tokens > 0, got %d", tokens)
	}
}

func TestFallbackTruncate(t *testing.T) {
	config := types.DefaultCompactionConfig()
	compactor := &Compactor{config: config}
	messages := make([]types.Message, 10)
	for i := 0; i < 10; i++ {
		messages[i] = types.Message{Role: "user", Content: fmt.Sprintf("Message %d", i)}
	}

	truncated := compactor.FallbackTruncate(messages)
	if len(truncated) != 5 {
		t.Errorf("Expected 5 messages after truncation, got %d", len(truncated))
	}
	if truncated[0].Content != "Message 5" {
		t.Error("Expected to keep last 5 messages")
	}
}
