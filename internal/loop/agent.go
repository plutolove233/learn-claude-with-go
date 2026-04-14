package loop

import (
	"context"
	"fmt"
	"os"

	"claudego/internal/config"
	"claudego/internal/tools"
	"claudego/pkg/llm"
	"claudego/pkg/logger"
	"claudego/pkg/types"
)

type Agent struct {
	cfg       *config.Config
	logger    *logger.Logger
	registry  *tools.Registry
	llmClient *llm.Client
}

func New(cfg *config.Config, l *logger.Logger, r *tools.Registry) *Agent {
	return &Agent{
		cfg:       cfg,
		logger:    l,
		registry:  r,
		llmClient: llm.NewClient(cfg),
	}
}

func (a *Agent) Run(ctx context.Context, messages []types.Message) error {
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	systemPrompt := fmt.Sprintf("You are a coding agent at %s. Use bash to solve tasks.", pwd)

	for {
		result, err := a.llmClient.Stream(ctx, messages, systemPrompt, a.registry)
		if err != nil {
			return fmt.Errorf("LLM call failed: %w", err)
		}

		a.logger.Info("LLM response: %s", result.Content)
		a.logger.Info("Stop reason: %s", result.FinishReason)
		for _, tc := range result.ToolCalls {
			a.logger.Info("Tool call - ID: %s, Name: %s, Arguments: %s", tc.ID, tc.Function.Name, tc.Function.Arguments)
		}

		// Persist ToolCalls in the assistant message so BuildMessages can
		// reconstruct a well-formed history (API requires tool_calls before tool results).
		messages = append(messages, types.Message{
			Role:      "assistant",
			Content:   result.Content,
			ToolCalls: result.ToolCalls,
		})

		if result.FinishReason == "stop" {
			break
		}

		if len(result.ToolCalls) > 0 {
			results := a.llmClient.ExecuteTools(ctx, result.ToolCalls, a.registry)

			a.logger.Info("Tool execution results: %+v", results)

			// Pass []ToolCallResult directly so BuildMessages emits proper
			// ToolMessage entries instead of a freeform user string.
			messages = append(messages, types.Message{Role: "user", Content: results})
			continue
		}
		break
	}
	return nil
}
