package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
	"github.com/openai/openai-go/v3/shared/constant"

	"claudego/internal/config"
	"claudego/pkg/interfaces"
	"claudego/pkg/types"
	"claudego/pkg/ui"
)

type Client struct {
	client openai.Client
	model  string
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		client: openai.NewClient(
			option.WithAPIKey(cfg.APIKey),
			option.WithBaseURL(cfg.BaseURL),
		),
		model: cfg.Model,
	}
}

func (c *Client) Model() string {
	return c.model
}

func (c *Client) Complete(ctx context.Context, messages []types.Message, system string, registry interfaces.ToolRegistry) (*types.CompleteResult, error) {
	var toolDefs []openai.ChatCompletionToolUnionParam
	if registry != nil {
		toolDefs = c.buildToolDefs(registry)
	}

	stream := c.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Messages: c.buildMessages(messages, system),
		Model:    shared.ChatModel(c.model),
		Tools:    toolDefs,
	})
	defer stream.Close()

	var fullContent strings.Builder
	assistantStream := ui.NewAssistantStreamer()
	stopSpin := make(chan struct{})
	go func() {
		for {
			select {
			case <-stopSpin:
				return
			case <-time.After(80 * time.Millisecond):
				if frame := assistantStream.Spin(); frame != "" {
					fmt.Print("\r  " + frame + "  ")
				}
			}
		}
	}()

	type partialToolCall struct {
		id        string
		toolType  string
		name      string
		arguments strings.Builder
	}
	var toolCallOrder []int
	toolCallsByIdx := map[int]*partialToolCall{}
	var finishReason string
	usage := &types.TokenUsage{}

	for stream.Next() {
		event := stream.Current()
		println(strings.Repeat("=", 20))
		fmt.Printf("Received event: %+v\n", event)
		println(strings.Repeat("=", 20))
		if len(event.Choices) == 0 {
			continue
		}

		delta := event.Choices[0].Delta

		if fr := string(event.Choices[0].FinishReason); fr != "" {
			finishReason = fr
		}

		if event.Usage.PromptTokens > 0 || event.Usage.CompletionTokens > 0 || event.Usage.TotalTokens > 0 {
			usage.PromptTokens = int(event.Usage.PromptTokens)
			usage.CompletionTokens = int(event.Usage.CompletionTokens)
			usage.TotalTokens = int(event.Usage.TotalTokens)
		}

		if delta.Content != "" {
			fullContent.WriteString(assistantStream.Write(delta.Content))
		}

		for _, tc := range delta.ToolCalls {
			idx := int(tc.Index)
			if _, exists := toolCallsByIdx[idx]; !exists {
				toolCallsByIdx[idx] = &partialToolCall{}
				toolCallOrder = append(toolCallOrder, idx)
			}
			p := toolCallsByIdx[idx]
			if tc.ID != "" {
				p.id = tc.ID
			}
			if tc.Type != "" {
				p.toolType = string(tc.Type)
			}
			if tc.Function.Name != "" {
				p.name = tc.Function.Name
			}
			p.arguments.WriteString(tc.Function.Arguments)
		}
	}

	close(stopSpin)
	if stream.Err() != nil {
		return nil, stream.Err()
	}
	fullContent.WriteString(assistantStream.Finish())

	toolCalls := make([]openai.ChatCompletionMessageToolCallUnion, 0, len(toolCallOrder))
	for _, idx := range toolCallOrder {
		p := toolCallsByIdx[idx]
		toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallUnion{
			ID:   p.id,
			Type: p.toolType,
			Function: openai.ChatCompletionMessageFunctionToolCallFunction{
				Name:      p.name,
				Arguments: p.arguments.String(),
			},
		})
	}

	var usageResult *types.TokenUsage
	if usage.TotalTokens > 0 || usage.PromptTokens > 0 || usage.CompletionTokens > 0 {
		usageResult = usage
	}

	return &types.CompleteResult{
		Content:      fullContent.String(),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage:        usageResult,
	}, nil
}

func (c *Client) buildMessages(messages []types.Message, system string) []openai.ChatCompletionMessageParamUnion {
	openaiMsgs := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages)+1)
	openaiMsgs = append(openaiMsgs, openai.SystemMessage(system))

	for _, m := range messages {
		switch m.Role {
		case "user":
			if content, ok := m.Content.(string); ok {
				openaiMsgs = append(openaiMsgs, openai.UserMessage(content))
			} else if results, ok := m.Content.([]types.ToolCallResult); ok {
				for _, r := range results {
					openaiMsgs = append(openaiMsgs, openai.ToolMessage(r.Content, r.ToolCallID))
				}
			}
		case "assistant":
			if content, ok := m.Content.(string); ok {
				if len(m.ToolCalls) > 0 {
					openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessageParamUnion{
						OfAssistant: &openai.ChatCompletionAssistantMessageParam{
							Content:   openai.ChatCompletionAssistantMessageParamContentUnion{OfString: openai.String(content)},
							ToolCalls: toToolCallParams(m.ToolCalls),
						},
					})
				} else {
					openaiMsgs = append(openaiMsgs, openai.AssistantMessage(content))
				}
			}
		}
	}
	return openaiMsgs
}

func (c *Client) buildToolDefs(registry interfaces.ToolRegistry) []openai.ChatCompletionToolUnionParam {
	toolDefs := registry.EnabledTools()
	if len(toolDefs) == 0 {
		return nil
	}

	result := make([]openai.ChatCompletionToolUnionParam, len(toolDefs))
	for i, t := range toolDefs {
		result[i] = openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        t.Name(),
			Description: openai.String(t.Description()),
			Parameters:  t.Parameters(),
		})
	}
	return result
}

func (c *Client) ExecuteTools(ctx context.Context, toolCalls []openai.ChatCompletionMessageToolCallUnion, registry interfaces.ToolRegistry) []types.ToolCallResult {
	var results []types.ToolCallResult
	enabledTools := registry.EnabledTools()

	for _, tc := range toolCalls {
		fn := tc.Function
		if fn.Name == "" {
			continue
		}

		ui.ToolCall(fn.Name, fn.Arguments)

		input := []byte(fn.Arguments)
		var output string
		var toolFound bool

		for _, t := range enabledTools {
			if t.Name() == fn.Name {
				toolFound = true
				out, execErr := t.Execute(ctx, input)
				if execErr != nil {
					output = "Error: " + execErr.Error()
				} else {
					output = out
				}
				break
			}
		}
		if !toolFound {
			output = fmt.Sprintf("Error: tool %q not found or not enabled", fn.Name)
		}

		ui.ToolOutput(output)

		results = append(results, types.ToolCallResult{
			Name:       fn.Name,
			ToolCallID: tc.ID,
			Content:    output,
		})
	}
	return results
}

func toToolCallParams(tcs []openai.ChatCompletionMessageToolCallUnion) []openai.ChatCompletionMessageToolCallUnionParam {
	params := make([]openai.ChatCompletionMessageToolCallUnionParam, len(tcs))
	for i, tc := range tcs {
		params[i] = openai.ChatCompletionMessageToolCallUnionParam{
			OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
				ID: tc.ID,
				Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
				Type: constant.Function(tc.Type),
			},
		}
	}
	return params
}
