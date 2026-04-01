package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sashabaranov/go-openai"

	"claudego/internal/config"
	"claudego/internal/tools"
	"claudego/pkg/logger"
)

type Message struct {
	Role    string
	Content interface{} // string or []ToolCallResult
}

type ToolCallResult struct {
	Role       string `json:"role"`
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
}

type Agent struct {
	cfg    *config.Config
	logger *logger.Logger
	tools  []tools.Tool
	client *openai.Client
}

func New(cfg *config.Config, l *logger.Logger, toolList []tools.Tool) *Agent {
	clientCfg := openai.DefaultConfig(cfg.APIKey)
	clientCfg.BaseURL = cfg.BaseURL + "/chat/completions"
	clientCfg.HTTPClient = &http.Client{Timeout: 120 * time.Second}

	client := openai.NewClientWithConfig(clientCfg)
	return &Agent{cfg: cfg, logger: l, tools: toolList, client: client}
}

func (a *Agent) Run(ctx context.Context, messages []Message) error {
	systemPrompt := "You are a coding agent. Use bash to solve tasks. Act, don't explain."

	for {
		resp, err := a.callLLM(ctx, messages, systemPrompt)
		if err != nil {
			return fmt.Errorf("LLM call failed: %w", err)
		}

		if len(resp.Choices) == 0 {
			return fmt.Errorf("no choices returned")
		}

		choice := resp.Choices[0]

		// Append assistant message
		messages = append(messages, Message{Role: "assistant", Content: choice.Message.Content})

		if choice.FinishReason != openai.FinishReasonToolCalls {
			// Output final response
			if choice.Message.Content != "" {
				fmt.Println(choice.Message.Content)
			}
			return nil
		}

		// Execute tools
		results, err := a.executeTools(choice.Message.ToolCalls)
		if err != nil {
			return fmt.Errorf("tool execution failed: %w", err)
		}

		// Append tool results
		messages = append(messages, Message{Role: "user", Content: results})
	}
}

func (a *Agent) callLLM(ctx context.Context, messages []Message, system string) (*openai.ChatCompletionResponse, error) {
	// Convert messages to OpenAI format
	openaiMsgs := make([]openai.ChatCompletionMessage, 0, len(messages)+1)
	openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: system,
	})
	for _, m := range messages {
		if m.Role == "user" {
			if content, ok := m.Content.(string); ok {
				openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: content,
				})
			} else if results, ok := m.Content.([]ToolCallResult); ok {
				for _, r := range results {
					openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    r.Content,
						ToolCallID: r.ToolCallID,
					})
				}
			}
		} else if m.Role == "assistant" {
			if content, ok := m.Content.(string); ok {
				openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: content,
				})
			}
		}
	}

	// Build tool definitions
	toolDefs := make([]openai.Tool, 0, len(a.tools))
	for _, t := range a.tools {
		toolDefs = append(toolDefs, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{"command": map[string]interface{}{"type": "string"}}},
			},
		})
	}

	resp, err := a.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    a.cfg.Model,
			Messages: openaiMsgs,
			Tools:    toolDefs,
		},
	)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned")
	}

	return &resp, nil
}

func (a *Agent) executeTools(toolCalls []openai.ToolCall) ([]ToolCallResult, error) {
	var results []ToolCallResult
	for _, tc := range toolCalls {
		var input map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
			input = map[string]interface{}{}
		}

		var output string
		for _, t := range a.tools {
			if t.Name() == tc.Function.Name {
				var err error
				output, err = t.Execute(input)
				if err != nil {
					output = "Error: " + err.Error()
				}
				break
			}
		}

		results = append(results, ToolCallResult{
			Role:       "tool",
			ToolCallID: tc.ID,
			Content:    output,
		})

		// Print the command being executed
		if cmd, ok := input["command"].(string); ok {
			fmt.Printf("\033[33m$ %s\033[0m\n", cmd)
			if len(output) > 200 {
				fmt.Println(output[:200] + "...")
			} else if output != "" {
				fmt.Println(output)
			}
		}
	}
	return results, nil
}
