package interfaces

import (
	"claudego/pkg/types"
	"context"
)

// LLMClient abstracts the LLM client for use by tools.
type LLMClient interface {
	// Complete performs a chat completion with tool support.
	Complete(ctx context.Context, messages []types.Message, system string, registry ToolRegistry) (*types.CompleteResult, error)
	// Model returns the model name being used.
	Model() string
}
