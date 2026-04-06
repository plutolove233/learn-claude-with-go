package loop

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
	"github.com/openai/openai-go/v3/shared/constant"

	"claudego/internal/config"
	"claudego/internal/tools"
	"claudego/pkg/logger"
)

type Message struct {
	Role      string
	Content   any                                         // string or []ToolCallResult
	ToolCalls []openai.ChatCompletionMessageToolCallUnion // populated for assistant messages
}

type ToolCallResult struct {
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Content    string `json:"content"`
}

type Agent struct {
	cfg      *config.Config
	logger   *logger.Logger
	registry *tools.Registry
	client   openai.Client
}

func New(cfg *config.Config, l *logger.Logger, r *tools.Registry) *Agent {
	client := openai.NewClient(
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(cfg.BaseURL),
	)
	return &Agent{cfg: cfg, logger: l, registry: r, client: client}
}

func (a *Agent) Run(ctx context.Context, messages []Message) error {
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	systemPrompt := fmt.Sprintf("You are a coding agent at %s. Use bash to solve tasks.", pwd)

	for {

		choice, err := a.callLLMStream(ctx, messages, systemPrompt)
		if err != nil {
			return fmt.Errorf("LLM call failed: %w", err)
		}

		a.logger.Info("LLM response: %s", choice.Message.Content)
		a.logger.Info("Stop reason: %s", choice.FinishReason)
		for _, tc := range choice.Message.ToolCalls {
			a.logger.Info("Tool call - ID: %s, Name: %s, Arguments: %s", tc.ID, tc.Function.Name, tc.Function.Arguments)
		}

		// Fix 1: Persist ToolCalls in the assistant message so buildMessages can
		// reconstruct a well-formed history (API requires tool_calls before tool results).
		messages = append(messages, Message{
			Role:      "assistant",
			Content:   choice.Message.Content,
			ToolCalls: choice.Message.ToolCalls,
		})

		if choice.FinishReason == "stop" {
			println("\n\nFinal response:")
			fmt.Println(choice.Message.Content)
			return nil
		}

		if len(choice.Message.ToolCalls) > 0 {
			results, err := a.executeTools(choice.Message.ToolCalls)
			if err != nil {
				return fmt.Errorf("tool execution failed: %w", err)
			}

			a.logger.Info("Tool execution results: %+v", results)

			// Fix 4: Pass []ToolCallResult directly so buildMessages emits proper
			// ToolMessage entries instead of a freeform user string.
			messages = append(messages, Message{Role: "user", Content: results})
			continue
		}
		return nil
	}
}

// callLLMStream streams a completion and reassembles the full choice.
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

	// Fix 2 & 3: Track tool calls by *index* (not ID) to preserve order and
	// correctly merge streamed chunks whose later deltas carry an empty ID.
	type partialToolCall struct {
		id        string
		toolType  string
		name      string
		arguments strings.Builder
	}
	var toolCallOrder []int                      // insertion-order index list
	toolCallsByIdx := map[int]*partialToolCall{} // index → accumulated data

	var finishReason string

	for stream.Next() {
		event := stream.Current()
		if len(event.Choices) == 0 {
			continue
		}

		delta := event.Choices[0].Delta

		// Fix 6: Only update finishReason when the field is non-empty so the
		// last meaningful value is preserved rather than potentially being
		// overwritten by a blank trailing chunk.
		if fr := string(event.Choices[0].FinishReason); fr != "" {
			finishReason = fr
		}

		if delta.Content != "" {
			// fmt.Print(delta.Content)
			fullContent.WriteString(delta.Content)
		}

		for _, tc := range delta.ToolCalls {
			idx := int(tc.Index)
			if _, exists := toolCallsByIdx[idx]; !exists {
				toolCallsByIdx[idx] = &partialToolCall{}
				toolCallOrder = append(toolCallOrder, idx)
			}
			p := toolCallsByIdx[idx]
			// ID and Name only arrive on the first chunk for each index.
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

	if stream.Err() != nil {
		return openai.ChatCompletionChoice{}, stream.Err()
	}

	fmt.Println()

	// Reassemble tool calls in their original order.
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
				// Fix 1 (continued): If the assistant turn contained tool calls,
				// emit them as part of the assistant message so the API receives
				// a valid history: assistant(tool_calls) → tool(results).
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

func (a *Agent) buildToolDefs() []openai.ChatCompletionToolUnionParam {
	tools := a.registry.EnabledTools()
	if len(tools) == 0 {
		return nil
	}

	toolDefs := make([]openai.ChatCompletionToolUnionParam, len(tools))
	for i, t := range tools {
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
	enabledTools := a.registry.EnabledTools()

	for _, tc := range toolCalls {
		fn := tc.Function
		if fn.Name == "" {
			continue
		}

		// Fix 7: Print the "executing" banner *before* the actual call so the
		// output reflects what is about to happen, not what already happened.
		fmt.Printf("\033[33m$ Execute %s(%s)\033[0m\n\n", fn.Name, fn.Arguments)

		input := []byte(fn.Arguments)

		var output string
		var toolFound bool
		for _, t := range enabledTools {
			if t.Name() == fn.Name {
				toolFound = true
				a.logger.Info("Execute tool: %s with args: %+v", t.Name(), input)
				out, execErr := t.Execute(input)
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

		if len(output) > 200 {
			fmt.Println(output[:200] + "...")
		} else if output != "" {
			fmt.Println(output)
		}

		results = append(results, ToolCallResult{
			Name:       fn.Name,
			ToolCallID: tc.ID,
			Content:    output,
		})
	}
	return results, nil
}

// toToolCallParams converts response-side ChatCompletionMessageToolCallUnion to the
// request-side ChatCompletionMessageToolCallUnionParam required by the API param structs.
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
