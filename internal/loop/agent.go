package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"

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
	Name       string `json:"name"`
	Content    string `json:"content"`
}

type Agent struct {
	cfg    *config.Config
	logger *logger.Logger
	tools  []tools.Tool
	client openai.Client
}

func New(cfg *config.Config, l *logger.Logger, toolList []tools.Tool) *Agent {
	client := openai.NewClient(
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(cfg.BaseURL),
	)
	return &Agent{cfg: cfg, logger: l, tools: toolList, client: client}
}

func (a *Agent) Run(ctx context.Context, messages []Message) error {
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	systemPrompt := fmt.Sprintf("You are a coding agent at %s. Use bash to solve tasks.", pwd)

	for {
		var choice openai.ChatCompletionChoice
		var err error

		a.logger.Info("User query: %s", messages[len(messages)-1])
		println()
		for _, m := range messages {
			fmt.Printf("%+v\n", m)
		}
		fmt.Println()
		choice, err = a.callLLMStream(ctx, messages, systemPrompt)
		if err != nil {
			return fmt.Errorf("LLM call failed: %w", err)
		}

		a.logger.Info("LLM response: %s", choice.Message.Content)
		a.logger.Info("Stop reason: %s", choice.FinishReason)
		a.logger.Info("LLM tool calls: %+v", choice.Message.ToolCalls)
		// Append assistant message
		messages = append(messages, Message{Role: "assistant", Content: choice.Message.Content})

		// No tool calls - output final response
		if choice.Message.Content != "" {
			fmt.Println(choice.Message.Content)
			return nil
		}

		// Check if model called tools
		if len(choice.Message.ToolCalls) > 0 {
			// Execute tools
			results, err := a.executeTools(choice.Message.ToolCalls)
			if err != nil {
				return fmt.Errorf("tool execution failed: %w", err)
			}

			// Append tool results
			a.logger.Info("Tool execution results: %+v", results)
			var feedback strings.Builder
			for _, r := range results {
				if r.Content == "" {
					fmt.Fprintf(&feedback, "Tool %s executed successfully with no output.\n", r.Name)
				} else {
					fmt.Fprintf(&feedback, "Tool %s executed, output: %s\n", r.Name, r.Content)
				}
			}
			messages = append(messages, Message{Role: "user", Content: feedback.String()})
			continue
		}
		return nil
	}
}

// callLLM non-streaming
func (a *Agent) callLLM(ctx context.Context, messages []Message, system string) (*openai.ChatCompletion, error) {
	openaiMsgs := a.buildMessages(messages, system)
	toolDefs := a.buildToolDefs()

	resp, err := a.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: openaiMsgs,
		Model:    shared.ChatModel(a.cfg.Model),
		Tools:    toolDefs,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// callLLMStream streaming版本
func (a *Agent) callLLMStream(ctx context.Context, messages []Message, system string) (openai.ChatCompletionChoice, error) {
	openaiMsgs := a.buildMessages(messages, system)
	toolDefs := a.buildToolDefs()

	stream := a.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Messages: openaiMsgs,
		Model:    shared.ChatModel(a.cfg.Model),
		Tools:    toolDefs,
	})
	defer stream.Close()

	var fullContent strings.Builder
	var toolCalls []openai.ChatCompletionMessageToolCallUnion
	var finishReason string

	for stream.Next() {
		event := stream.Current()
		if len(event.Choices) == 0 {
			continue
		}

		delta := event.Choices[0].Delta
		finishReason = event.Choices[0].FinishReason

		// Accumulate content
		if delta.Content != "" {
			fmt.Print(delta.Content)
			fullContent.WriteString(delta.Content)
		}

		// Accumulate tool calls from streaming delta
		for _, tc := range delta.ToolCalls {
			// Convert ChatCompletionChunkChoiceDeltaToolCall to ChatCompletionMessageToolCallUnion
			toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallUnion{
				ID:   tc.ID,
				Type: tc.Type,
				Function: openai.ChatCompletionMessageFunctionToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
	}

	if stream.Err() != nil {
		return openai.ChatCompletionChoice{}, stream.Err()
	}

	fmt.Println() // newline after streaming output

	return openai.ChatCompletionChoice{
		Message: openai.ChatCompletionMessage{
			Content:   fullContent.String(),
			ToolCalls: toolCalls,
		},
		FinishReason: finishReason,
	}, nil
}

func (a *Agent) buildMessages(messages []Message, system string) []openai.ChatCompletionMessageParamUnion {
	openaiMsgs := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages)+1)
	openaiMsgs = append(openaiMsgs, openai.SystemMessage(system))

	for _, m := range messages {
		switch m.Role {
		case "user":
			if content, ok := m.Content.(string); ok {
				openaiMsgs = append(openaiMsgs, openai.UserMessage(content))
			} else if results, ok := m.Content.([]ToolCallResult); ok {
				for _, r := range results {
					openaiMsgs = append(openaiMsgs, openai.ToolMessage(r.Content, r.ToolCallID))
				}
			}
		case "assistant":
			if content, ok := m.Content.(string); ok {
				openaiMsgs = append(openaiMsgs, openai.AssistantMessage(content))
			}
		}
	}
	return openaiMsgs
}

func (a *Agent) buildToolDefs() []openai.ChatCompletionToolUnionParam {
	if len(a.tools) == 0 {
		return nil
	}

	toolDefs := make([]openai.ChatCompletionToolUnionParam, len(a.tools))
	for i, t := range a.tools {
		toolDefs[i] = openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        t.Name(),
			Description: openai.String(t.Description()),
			Parameters:  t.Parameters(),
		})
	}
	return toolDefs
}

func (a *Agent) executeTools(toolCalls []openai.ChatCompletionMessageToolCallUnion) ([]ToolCallResult, error) {
	var results []ToolCallResult
	for _, tc := range toolCalls {
		fn := tc.Function
		if fn.Name == "" {
			continue
		}

		var input map[string]any
		if err := json.Unmarshal([]byte(fn.Arguments), &input); err != nil {
			input = map[string]any{}
		}

		var output string
		var execErr error
		for _, t := range a.tools {
			// Match tool by name and execute
			if t.Name() == fn.Name {
				a.logger.Info("Execute tool: %s with args: %+v", t.Name(), input)
				output, execErr = t.Execute(input)
				if execErr != nil {
					output = "Error: " + execErr.Error()
				}
				break
			}
		}

		results = append(results, ToolCallResult{
			Role:       "tool",
			Name:       fn.Name,
			ToolCallID: tc.ID,
			Content:    output,
		})

		// Print the command being executed
		fmt.Printf("\033[33m$ Execute %s(%s)\033[0m\n\n", fn.Name, fn.Arguments)
		if len(output) > 200 {
			fmt.Println(output[:200] + "...")
		} else if output != "" {
			fmt.Println(output)
		}
	}
	return results, nil
}
