package interfaces

import (
	"context"

	"claudego/pkg/types"

	"github.com/openai/openai-go/v3"
)

// LLMClient abstracts the LLM client for use by tools.
type LLMClient interface {
	// Complete performs a chat completion with tool support.
	Complete(ctx context.Context, messages []types.Message, system string, registry ToolRegistry) (*types.CompleteResult, error)
	// Model returns the model name being used.
	Model() string
	// ExecuteTool allows tools to call other tools via the LLM.
	ExecuteTools(ctx context.Context, toolCalls []openai.ChatCompletionMessageToolCallUnion, registry ToolRegistry) []types.ToolCallResult
}
