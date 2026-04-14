package types

import (
	"github.com/openai/openai-go/v3"
)

// Message represents a conversation message
type Message struct {
	Role      string
	Content   any                                         // string or []ToolCallResult
	ToolCalls []openai.ChatCompletionMessageToolCallUnion // populated for assistant messages
}

// ToolCallResult represents the result of a tool execution
type ToolCallResult struct {
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Content    string `json:"content"`
}

// CompleteResult represents the result of an LLM completion, including content, tool calls, and finish reason.
type CompleteResult struct {
	Content      string
	ToolCalls    []openai.ChatCompletionMessageToolCallUnion
	FinishReason string
}
