package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"claudego/pkg/interfaces"
	"claudego/pkg/types"
	"claudego/pkg/ui"
)

// TaskInput defines the input for the task tool.
type TaskInput struct {
	Question     string   `json:"question" validate:"required"`
	AllowedTools []string `json:"allowedTools"` // whitelist - only these tools can be used
	SystemHint   string   `json:"systemHint"`   // optional hint added to system prompt
}

type TaskTool struct {
	BaseTool[TaskInput]
	llmClient interfaces.LLMClient
}

// NewTaskTool creates a TaskTool with the given LLM client
func NewTaskTool(llmClient interfaces.LLMClient) *TaskTool {
	return &TaskTool{
		llmClient: llmClient,
	}
}

// Name returns the tool name
func (t *TaskTool) Name() string {
	return "task"
}

// Description returns the tool description
func (t *TaskTool) Description() string {
	return "Delegate a subtask to a sub-agent with isolated context and restricted tools. Returns a plain text summary."
}

// Parameters returns the JSON schema for tool input
func (t *TaskTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"question": map[string]any{
				"type":        "string",
				"description": "The question or task to delegate to the sub-agent",
			},
			"allowedTools": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "List of tool names the sub-agent can use (whitelist)",
			},
			"systemHint": map[string]any{
				"type":        "string",
				"description": "Optional hint to guide the sub-agent's response",
			},
		},
		"required": []string{"question"},
	}
}

// Metadata returns tool metadata
func (t *TaskTool) Metadata() types.ToolMetadata {
	return types.ToolMetadata{
		Category:   types.CategorySystem,
		SafeToSkip: false,
		MaxRetries: 0,
	}
}

// Execute runs the sub-agent with given input
func (t *TaskTool) Execute(ctx context.Context, input []byte) (string, error) {
	var req TaskInput
	if err := json.Unmarshal(input, &req); err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	ui.ToolCall("task", req.Question)

	// Build messages with optional context
	messages := []types.Message{
		{Role: "user", Content: req.Question},
	}

	system := "You are a helpful assistant."
	if req.SystemHint != "" {
		system = system + "\n\nHint: " + req.SystemHint
	}

	// Create filtered registry if whitelist specified
	var registry interfaces.ToolRegistry
	if len(req.AllowedTools) > 0 {
		registry = GetRegistry().Filter(req.AllowedTools)
	} else {
		registry = NewRegistry()
	}

	for {
		response, err := t.llmClient.Complete(ctx, messages, system, registry)
		if err != nil {
			return "", fmt.Errorf("subtask failed: %w", err)
		}
		messages = append(messages, types.Message{
			Role: "assistant", 
			Content: response.Content,
		})

		if response.FinishReason == "stop" {
			break
		}

		if len(response.ToolCalls) > 0 {
			results := t.llmClient.ExecuteTools(ctx, response.ToolCalls, registry)

			messages = append(messages, types.Message{Role: "user", Content: results})
			continue
		}
		break
	}
	// Call LLM with filtered registry

	ui.ToolOutput(messages[len(messages)-1].Content.(string))
	return messages[len(messages)-1].Content.(string), nil
}
