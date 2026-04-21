package compaction

import (
	"testing"

	"claudego/pkg/types"
)

func TestCountMessage_String(t *testing.T) {
	tc := NewTokenCounter()
	msg := types.Message{Role: "user", Content: "Hello, world!"}
	count := tc.CountMessage(msg)
	if count < 1 {
		t.Errorf("Expected count >= 1, got %d", count)
	}
}

func TestCountMessage_ToolResults(t *testing.T) {
	tc := NewTokenCounter()
	results := []types.ToolCallResult{{
		ToolCallID: "call_1",
		Name:       "bash",
		Content:    "output line 1\\noutput line 2",
	}}
	msg := types.Message{Role: "user", Content: results}
	count := tc.CountMessage(msg)
	if count < 1 {
		t.Errorf("Expected count >= 1, got %d", count)
	}
}

func TestCountMessages(t *testing.T) {
	tc := NewTokenCounter()
	messages := []types.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
		{Role: "user", Content: "How are you?"},
	}
	total := tc.CountMessages(messages)
	if total < 3 {
		t.Errorf("Expected total >= 3, got %d", total)
	}
	if tc.LastCount() != total {
		t.Errorf("Expected LastCount to be %d, got %d", total, tc.LastCount())
	}
}

func TestCountMessages_Empty(t *testing.T) {
	tc := NewTokenCounter()
	total := tc.CountMessages([]types.Message{})
	if total != 0 {
		t.Errorf("Expected total to be 0 for empty messages, got %d", total)
	}
}

func TestEstimateCompression(t *testing.T) {
	tc := NewTokenCounter()
	messages := []types.Message{
		{Role: "user", Content: "First message"},
		{Role: "assistant", Content: "Response 1"},
		{Role: "user", Content: "Second message"},
		{Role: "assistant", Content: "Response 2"},
	}
	estimated := tc.EstimateCompression(messages, 500)
	if estimated < 500 {
		t.Errorf("Expected estimated >= 500, got %d", estimated)
	}
}
