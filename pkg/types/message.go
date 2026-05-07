package types

import (
	openai "github.com/sashabaranov/go-openai"
)

// Message represents a conversation message
type Message struct {
	Role             string
	ReasoningContent string            // populated for assistant messages with reasoning
	Content          string            // string or []ToolCallResult
	ToolCalls        []openai.ToolCall // populated for assistant messages
	ToolResults      []ToolCallResult
}

// ToolCallResult represents the result of a tool execution
type ToolCallResult struct {
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Content    string `json:"content"`
}

// TokenUsage records token usage from one model completion.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CompleteResult represents the result of an LLM completion, including content, tool calls, and finish reason.
type CompleteResult struct {
	Content          string
	ToolCalls        []openai.ToolCall
	FinishReason     string
	Usage            *TokenUsage `json:"usage,omitempty"`
	ReasoningContent string      `json:"reasoning_content,omitempty"`
}
