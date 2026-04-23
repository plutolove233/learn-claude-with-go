package graph

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claudego/pkg/interfaces"
	"claudego/pkg/types"
)

// MessageBuilder builds the prompt history for one graph node run.
type MessageBuilder func(state *State) []types.Message

// PromptBuilder builds a single user message for one graph node run.
type PromptBuilder func(state *State) string

// ResultHandler receives the final model result and may update state.
type ResultHandler func(state *State, result *types.CompleteResult) error

// LLMNodeConfig configures an LLM-backed graph node.
type LLMNodeConfig struct {
	Client            interfaces.LLMClient
	Registry          interfaces.ToolRegistry
	SystemPrompt      string
	BuildSystemPrompt func(state *State) string
	BuildMessages     MessageBuilder
	BuildUserPrompt   PromptBuilder
	ResponseKey       string
	RawResultKey      string
	ToolResultsKey    string
	HandleResult      ResultHandler
	MaxToolRounds     int  // Maximum tool call rounds (0 = unlimited, default 10)
	FailOnToolError   bool // If true, node fails when any tool returns error
}

// NewLLMNode adapts the existing LLM client + tool registry into a graph node.
func NewLLMNode(cfg LLMNodeConfig) (NodeFunc, error) {
	if cfg.Client == nil {
		return nil, errors.New("llm client is required")
	}
	if cfg.BuildMessages == nil && cfg.BuildUserPrompt == nil {
		return nil, errors.New("either BuildMessages or BuildUserPrompt must be provided")
	}

	responseKey := cfg.ResponseKey
	if responseKey == "" {
		responseKey = "last_response"
	}

	maxRounds := cfg.MaxToolRounds
	if maxRounds == 0 {
		maxRounds = 10
	}

	return func(ctx context.Context, state *State) (*Command, error) {
		// Check context cancellation before starting
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		systemPrompt := cfg.SystemPrompt
		if cfg.BuildSystemPrompt != nil {
			systemPrompt = cfg.BuildSystemPrompt(state)
		}

		messages := buildNodeMessages(state, cfg)
		result, err := cfg.Client.Complete(ctx, messages, systemPrompt, cfg.Registry)
		if err != nil {
			return nil, err
		}

		// Multi-round tool calling loop
		round := 0
		for len(result.ToolCalls) > 0 && round < maxRounds {
			// Check context cancellation before tool execution
			if err := ctx.Err(); err != nil {
				return nil, fmt.Errorf("context cancelled during tool execution: %w", err)
			}

			toolResults := cfg.Client.ExecuteTools(ctx, result.ToolCalls, cfg.Registry)

			// Check for tool errors if FailOnToolError is enabled
			if cfg.FailOnToolError {
				var toolErrors []string
				for _, tr := range toolResults {
					if strings.HasPrefix(tr.Content, "Error:") {
						toolErrors = append(toolErrors, fmt.Sprintf("%s: %s", tr.Name, tr.Content))
					}
				}
				if len(toolErrors) > 0 {
					return nil, fmt.Errorf("tool execution failed: %s", strings.Join(toolErrors, "; "))
				}
			}

			if cfg.ToolResultsKey != "" {
				state.Set(cfg.ToolResultsKey, toolResults)
			}

			messages = append(messages,
				types.Message{
					Role:      "assistant",
					Content:   result.Content,
					ToolCalls: result.ToolCalls,
				},
				types.Message{
					Role:    "user",
					Content: toolResults,
				},
			)

			result, err = cfg.Client.Complete(ctx, messages, systemPrompt, cfg.Registry)
			if err != nil {
				return nil, err
			}
			round++
		}

		if round >= maxRounds && len(result.ToolCalls) > 0 {
			return nil, fmt.Errorf("exceeded max tool rounds (%d)", maxRounds)
		}

		state.Set(responseKey, result.Content)
		if cfg.RawResultKey != "" {
			state.Set(cfg.RawResultKey, *result)
		}

		if cfg.HandleResult != nil {
			if err := cfg.HandleResult(state, result); err != nil {
				return nil, err
			}
		}

		return nil, nil
	}, nil
}

func buildNodeMessages(state *State, cfg LLMNodeConfig) []types.Message {
	if cfg.BuildMessages != nil {
		return cfg.BuildMessages(state)
	}

	return []types.Message{
		{
			Role:    "user",
			Content: cfg.BuildUserPrompt(state),
		},
	}
}
