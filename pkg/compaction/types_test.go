package compaction

import (
	"claudego/pkg/types"
	"testing"
	"time"
)

func TestDefaultCompactionConfig(t *testing.T) {
	cfg := types.DefaultCompactionConfig()
	if !cfg.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if cfg.LargeResultThreshold != 1024 {
		t.Errorf("Expected LargeResultThreshold to be 1024, got %d", cfg.LargeResultThreshold)
	}
	if cfg.OldMessageThreshold != 20 {
		t.Errorf("Expected OldMessageThreshold to be 20, got %d", cfg.OldMessageThreshold)
	}
	if cfg.AutoCompactTokens != 8000 {
		t.Errorf("Expected AutoCompactTokens to be 8000, got %d", cfg.AutoCompactTokens)
	}
	if cfg.SummaryMinTokens != 500 {
		t.Errorf("Expected SummaryMinTokens to be 500, got %d", cfg.SummaryMinTokens)
	}
	if cfg.SummaryMaxTokens != 1000 {
		t.Errorf("Expected SummaryMaxTokens to be 1000, got %d", cfg.SummaryMaxTokens)
	}
	if cfg.ShowTokenUsage != true {
		t.Error("Expected ShowTokenUsage to be true")
	}
}

func TestStoredToolResult(t *testing.T) {
	now := time.Now()
	result := types.StoredToolResult{
		ToolCallID: "call_123",
		ToolName:   "bash_exec",
		Arguments:  "ls -la",
		Content:    "file1.txt\\nfile2.txt",
		Timestamp:  now,
	}
	if result.ToolCallID != "call_123" {
		t.Errorf("Expected ToolCallID to be call_123, got %s", result.ToolCallID)
	}
	if result.ToolName != "bash_exec" {
		t.Errorf("Expected ToolName to be bash_exec, got %s", result.ToolName)
	}
	if result.CompressedAt != nil {
		t.Error("Expected CompressedAt to be nil initially")
	}
}

func TestToolResultReference(t *testing.T) {
	ref := types.ToolResultReference{
		Type:     "tool_result_reference",
		FilePath: "~/.claudego/sessions/sess_123/tool-results/20260420-150301-bash.json",
		Summary:  "file1.txt\\nfile2.txt",
	}
	if ref.Type != "tool_result_reference" {
		t.Errorf("Expected Type to be tool_result_reference, got %s", ref.Type)
	}
	if ref.FilePath == "" {
		t.Error("Expected FilePath to be non-empty")
	}
}
