package compaction

import (
	"strings"
	"testing"

	"claudego/pkg/types"
)

func TestBuildSummaryPrompt(t *testing.T) {
	summarizer := &Summarizer{config: types.DefaultCompactionConfig()}
	messages := []types.Message{
		{Role: "user", Content: "What is Go?"},
		{Role: "assistant", Content: "Go is a programming language."},
	}
	prompt := summarizer.buildSummaryPrompt(messages, "sess_123")

	if !strings.Contains(prompt, "摘要") {
		t.Error("Expected prompt to contain '摘要'")
	}
	if !strings.Contains(prompt, "sess_123") {
		t.Error("Expected prompt to contain session ID")
	}
	if !strings.Contains(prompt, "What is Go?") {
		t.Error("Expected prompt to contain user message")
	}
}

func TestTruncateSummary(t *testing.T) {
	summarizer := &Summarizer{config: types.DefaultCompactionConfig()}
	longSummary := "This is a very long summary. " + strings.Repeat("x", 5000)
	truncated := summarizer.TruncateSummary(longSummary, 100)

	if len([]rune(truncated)) > 500 {
		t.Errorf("Expected truncated summary to be <= 500 chars, got %d", len([]rune(truncated)))
	}
	if !strings.Contains(truncated, "已截断") {
		t.Error("Expected truncated summary to contain truncation marker")
	}
}

func TestTruncateSummary_NoTruncation(t *testing.T) {
	summarizer := &Summarizer{config: types.DefaultCompactionConfig()}
	shortSummary := "This is a short summary."
	truncated := summarizer.TruncateSummary(shortSummary, 1000)
	if truncated != shortSummary {
		t.Error("Expected short summary to not be truncated")
	}
}
