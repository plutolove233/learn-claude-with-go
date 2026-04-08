package plan

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
	"github.com/openai/openai-go/v3/shared/constant"

	"claudego/internal/config"
	"claudego/internal/loop"
	"claudego/internal/tools"
	"claudego/pkg/logger"
)

type Executor struct {
	planner  *Planner
	registry interface {
		EnabledTools() []interface {
			Name() string
			Description() string
			Execute(input []byte) (string, error)
			Parameters() map[string]any
		}
	}
	client openai.Client
	model  string
	logger *logger.Logger
}

func NewExecutor(cfg *config.Config, log *logger.Logger, registry *tools.Registry) *Executor {
	planner := NewPlanner(cfg)
	registryAdapter := &registryAdapter{registry}
	return &Executor{
		planner:  planner,
		registry: registryAdapter,
		client:   openai.NewClient(option.WithAPIKey(cfg.APIKey), option.WithBaseURL(cfg.BaseURL)),
		model:    cfg.Model,
		logger:   log,
	}
}

type registryAdapter struct {
	r *tools.Registry
}

func (a *registryAdapter) EnabledTools() []interface {
	Name() string
	Description() string
	Execute(input []byte) (string, error)
	Parameters() map[string]any
} {
	tools := a.r.EnabledTools()
	result := make([]interface {
		Name() string
		Description() string
		Execute(input []byte) (string, error)
		Parameters() map[string]any
	}, len(tools))
	for i, t := range tools {
		result[i] = t
	}
	return result
}

func (e *Executor) RunWithPlan(ctx context.Context, goal string) (*Plan, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create the plan using the LLM
	fmt.Print("\033[36m📋 Analyzing task and creating plan...\033[0m\n\n")
	plan, err := e.planner.CreatePlan(ctx, PlanningRequest{Goal: goal, PWD: pwd})
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	// Display the plan
	e.DisplayPlan(plan)

	// Save initial plan
	if err := plan.Save(); err != nil {
		e.logger.Warning("Failed to save plan: %v", err)
	}

	// Execute the plan
	plan.Status = PlanStatusInProgress
	if err := plan.Save(); err != nil {
		e.logger.Warning("Failed to save plan: %v", err)
	}

	fmt.Print("\033[36m🚀 Starting execution...\033[0m\n\n")
	if err := e.executePlan(ctx, plan); err != nil {
		plan.Status = PlanStatusFailed
		plan.Save()
		return plan, err
	}

	// Check if all steps completed
	if plan.HasFailedSteps() {
		plan.Status = PlanStatusFailed
	} else {
		plan.Status = PlanStatusCompleted
	}
	plan.Save()

	e.displaySummary(plan)
	return plan, nil
}

func (e *Executor) ResumePlan(ctx context.Context, plan *Plan) error {
	if plan == nil {
		return fmt.Errorf("plan is nil")
	}

	e.DisplayPlan(plan)

	plan.Status = PlanStatusInProgress
	if err := plan.Save(); err != nil {
		e.logger.Warning("Failed to save plan: %v", err)
	}

	fmt.Print("\033[36m🚀 Resuming execution...\033[0m\n\n")
	if err := e.executePlan(ctx, plan); err != nil {
		plan.Status = PlanStatusPaused
		if saveErr := plan.Save(); saveErr != nil {
			e.logger.Warning("Failed to save paused plan: %v", saveErr)
		}
		return err
	}

	if plan.HasFailedSteps() {
		plan.Status = PlanStatusFailed
	} else {
		plan.Status = PlanStatusCompleted
	}
	if err := plan.Save(); err != nil {
		e.logger.Warning("Failed to save plan: %v", err)
	}

	e.displaySummary(plan)
	return nil
}

func (e *Executor) executePlan(ctx context.Context, plan *Plan) error {
	for i := range plan.Steps {
		step := &plan.Steps[i]
		if step.Status == StepStatusCompleted {
			continue
		}

		// Update plan current step
		plan.CurrentStep = i
		plan.Save()

		fmt.Printf("\033[33m\n📌 Step %d/%d: %s\033[0m\n\n", i+1, len(plan.Steps), step.Task)

		// Mark step as started
		now := time.Now()
		step.Status = StepStatusInProgress
		step.StartedAt = &now
		step.Error = ""

		// Execute the step
		result, err := e.executeStep(ctx, step)
		if err != nil {
			step.Status = StepStatusFailed
			step.Error = err.Error()
			now = time.Now()
			step.CompletedAt = &now
			plan.Save()
			fmt.Printf("\033[31m❌ Step failed: %v\033[0m\n", err)

			// Ask if user wants to continue
			fmt.Println("\033[33m⚠️  Plan execution paused. Changes have been saved.\033[0m")
			fmt.Println("You can resume later with /plan resume")
			return err
		}

		// Step completed successfully
		step.Status = StepStatusCompleted
		step.Result = truncate(result, 500)
		now = time.Now()
		step.CompletedAt = &now
		plan.Save()

		fmt.Printf("\033[32m✅ Step completed\033[0m\n\n")
	}
	return nil
}

func (e *Executor) executeStep(ctx context.Context, step *Step) (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	systemPrompt := fmt.Sprintf("You are a coding agent at %s. Execute the following task and report results.", pwd)

	messages := []loop.Message{
		{Role: "user", Content: step.Task},
	}

	// Use the same streaming approach as the main agent
	stream := e.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Messages: e.buildMessages(messages, systemPrompt),
		Model:    shared.ChatModel(e.model),
		Tools:    e.buildToolDefs(),
	})
	defer stream.Close()

	var fullContent strings.Builder
	var toolCallOrder []int
	type partialToolCall struct {
		id        string
		toolType  string
		name      string
		arguments strings.Builder
	}
	toolCallsByIdx := map[int]*partialToolCall{}

	for stream.Next() {
		event := stream.Current()
		if len(event.Choices) == 0 {
			continue
		}

		delta := event.Choices[0].Delta

		if delta.Content != "" {
			fullContent.WriteString(delta.Content)
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

	if stream.Err() != nil {
		return "", fmt.Errorf("LLM call failed: %w", stream.Err())
	}

	// Build tool calls
	var toolCalls []openai.ChatCompletionMessageToolCallUnion
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

	// If there are tool calls, execute them
	if len(toolCalls) > 0 {
		fmt.Println()
		results, err := e.executeTools(ctx, toolCalls)
		if err != nil {
			return "", fmt.Errorf("tool execution failed: %w", err)
		}

		// Append assistant message with tool calls
		messages = append(messages, loop.Message{
			Role:      "assistant",
			Content:   fullContent.String(),
			ToolCalls: toolCalls,
		})

		// Append tool results
		messages = append(messages, loop.Message{Role: "user", Content: results})

		// Continue the conversation to get final response
		stream2 := e.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
			Messages: e.buildMessages(messages, systemPrompt),
			Model:    shared.ChatModel(e.model),
			Tools:    e.buildToolDefs(),
		})
		defer stream2.Close()

		fullContent.Reset()
		for stream2.Next() {
			event := stream2.Current()
			if len(event.Choices) == 0 {
				continue
			}
			if delta := event.Choices[0].Delta.Content; delta != "" {
				fullContent.WriteString(delta)
			}
		}
		if stream2.Err() != nil {
			return "", fmt.Errorf("LLM call failed: %w", stream2.Err())
		}
	}

	return fullContent.String(), nil
}

func (e *Executor) executeTools(ctx context.Context, toolCalls []openai.ChatCompletionMessageToolCallUnion) ([]loop.ToolCallResult, error) {
	var results []loop.ToolCallResult
	enabledTools := e.registry.EnabledTools()

	for _, tc := range toolCalls {
		fn := tc.Function
		if fn.Name == "" {
			continue
		}

		fmt.Printf("\033[33m$ Execute %s(%s)\033[0m\n\n", fn.Name, fn.Arguments)

		input := []byte(fn.Arguments)
		var output string
		var toolFound bool

		for _, t := range enabledTools {
			if t.Name() == fn.Name {
				toolFound = true
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

		results = append(results, loop.ToolCallResult{
			Name:       fn.Name,
			ToolCallID: tc.ID,
			Content:    output,
		})
	}
	return results, nil
}

func (e *Executor) buildMessages(messages []loop.Message, system string) []openai.ChatCompletionMessageParamUnion {
	openaiMsgs := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages)+1)
	openaiMsgs = append(openaiMsgs, openai.SystemMessage(system))

	for _, m := range messages {
		switch m.Role {
		case "user":
			if content, ok := m.Content.(string); ok {
				openaiMsgs = append(openaiMsgs, openai.UserMessage(content))
			} else if results, ok := m.Content.([]loop.ToolCallResult); ok {
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

func (e *Executor) buildToolDefs() []openai.ChatCompletionToolUnionParam {
	tools := e.registry.EnabledTools()
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

func (e *Executor) DisplayPlan(plan *Plan) {
	fmt.Println("\033[36m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Printf("\033[36m📋 Plan: %s\033[0m\n", plan.Name)
	fmt.Printf("\033[36m🎯 Goal: %s\033[0m\n", plan.Goal)
	fmt.Println("\033[36m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Println("\033[36mSteps:\033[0m")
	for i, step := range plan.Steps {
		fmt.Printf("  \033[33m%d.\033[0m %s\n", i+1, step.Task)
	}
	fmt.Print("\033[36m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m\n\n")
}

func (e *Executor) displaySummary(plan *Plan) {
	fmt.Println("\033[36m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Printf("\033[36m📊 Plan '%s' - Execution Summary\033[0m\n", plan.Name)

	total := len(plan.Steps)
	completed := 0
	failed := 0

	for i, step := range plan.Steps {
		var icon string
		var color string
		switch step.Status {
		case StepStatusCompleted:
			icon = "✅"
			color = "32"
			completed++
		case StepStatusFailed:
			icon = "❌"
			color = "31"
			failed++
		default:
			icon = "⏳"
			color = "33"
		}
		status := fmt.Sprintf("\033[%sm%s\033[0m", color, step.Status)
		fmt.Printf("  %s Step %d: %s (%s)\n", icon, i+1, step.Task, status)
	}

	fmt.Println()
	fmt.Printf("\033[32mCompleted: %d\033[0m | \033[31mFailed: %d\033[0m | Total: %d\n", completed, failed, total)
	fmt.Print("\033[36m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m\n\n")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
