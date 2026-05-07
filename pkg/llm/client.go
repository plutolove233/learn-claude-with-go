package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"

	"claudego/internal/config"
	"claudego/pkg/interfaces"
	"claudego/pkg/types"
	"claudego/pkg/ui"
)

type Client struct {
	client *openai.Client
	model  string
}

func NewClient(cfg *config.Config) *Client {
	c := openai.DefaultConfig(cfg.APIKey)
	c.BaseURL = cfg.BaseURL
	return &Client{
		client: openai.NewClientWithConfig(c),
		model:  cfg.Model,
	}
}

func (c *Client) Model() string {
	return c.model
}

func (c *Client) Complete(ctx context.Context, messages []types.Message, system string, registry interfaces.ToolRegistry) (*types.CompleteResult, error) {
	var toolDefs []openai.Tool
	if registry != nil {
		toolDefs = c.buildToolDefs(registry)
	}

	stream, err := c.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:    c.model,
		Messages: c.buildMessages(messages, system),
		Tools:    toolDefs,

		StreamOptions: &openai.StreamOptions{
			IncludeUsage: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create chat completion stream: %w", err)
	}
	defer stream.Close()

	var fullContent strings.Builder
	var reasoningContent strings.Builder
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

	for {
		event, err := stream.Recv()
		if err != nil {
			break
		}

		if event.Usage != nil {
			usage.PromptTokens = event.Usage.PromptTokens
			usage.CompletionTokens = event.Usage.CompletionTokens
			usage.TotalTokens = event.Usage.TotalTokens
			continue
		}

		if len(event.Choices) == 0 {
			continue
		}

		delta := event.Choices[0].Delta

		if fr := string(event.Choices[0].FinishReason); fr != "" {
			finishReason = fr
		}

		if delta.ReasoningContent != "" {
			reasoningContent.WriteString(delta.ReasoningContent)
		}

		if delta.Content != "" {
			fullContent.WriteString(assistantStream.Write(delta.Content))
		}

		for _, tc := range delta.ToolCalls {
			idx := *tc.Index
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
	fullContent.WriteString(assistantStream.Finish())

	toolCalls := make([]openai.ToolCall, 0, len(toolCallOrder))
	for _, idx := range toolCallOrder {
		p := toolCallsByIdx[idx]
		toolCalls = append(toolCalls, openai.ToolCall{
			ID:   p.id,
			Type: openai.ToolType(p.toolType),
			Function: openai.FunctionCall{
				Name:      p.name,
				Arguments: p.arguments.String(),
			},
		})
	}

	return &types.CompleteResult{
		Content:          fullContent.String(),
		ToolCalls:        toolCalls,
		FinishReason:     finishReason,
		Usage:            usage,
		ReasoningContent: reasoningContent.String(),
	}, nil
}

func (c *Client) buildMessages(messages []types.Message, system string) []openai.ChatCompletionMessage {
	openaiMsgs := make([]openai.ChatCompletionMessage, 0, len(messages)+1)
	openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: system,
	})

	for _, m := range messages {
		switch m.Role {
		case openai.ChatMessageRoleUser:
			openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: m.Content,
			})
		case openai.ChatMessageRoleTool:
			for _, r := range m.ToolResults {
				openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    r.Content,
					ToolCallID: r.ToolCallID,
				})
			}
		case openai.ChatMessageRoleAssistant:
			openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
				Role:             openai.ChatMessageRoleAssistant,
				ReasoningContent: m.ReasoningContent,
				Content:          m.Content,
				ToolCalls:        m.ToolCalls,
			})
		}
	}
	return openaiMsgs
}

func (c *Client) buildToolDefs(registry interfaces.ToolRegistry) []openai.Tool {
	toolDefs := registry.EnabledTools()
	if len(toolDefs) == 0 {
		return nil
	}

	result := make([]openai.Tool, len(toolDefs))
	for i, t := range toolDefs {
		result[i] = openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		}
	}
	return result
}
