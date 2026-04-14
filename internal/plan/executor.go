package plan

import (
	"context"
	"fmt"
	"os"
	"time"

	"claudego/internal/config"
	"claudego/pkg/interfaces"
	"claudego/pkg/llm"
	"claudego/pkg/logger"
	"claudego/pkg/types"
	"claudego/pkg/ui"
)

type Executor struct {
	planner   *Planner
	registry  interfaces.ToolRegistry
	llmClient *llm.Client
	logger    *logger.Logger
}

func NewExecutor(cfg *config.Config, log *logger.Logger, registry interfaces.ToolRegistry) *Executor {
	return &Executor{
		planner:   NewPlanner(cfg),
		registry:  registry,
		llmClient: llm.NewClient(cfg),
		logger:    log,
	}
}

func (e *Executor) RunWithPlan(ctx context.Context, goal string) (*Plan, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create the plan using the LLM
	ui.Info("Analyzing task and creating plan...")
	ui.Blank()
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

	ui.Info("Starting execution...")
	ui.Blank()
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

	ui.Info("Resuming execution...")
	ui.Blank()
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

		ui.Blank()
		ui.Step(i+1, len(plan.Steps), step.Task)
		ui.Blank()

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
			ui.Error(fmt.Sprintf("Step failed: %v", err))

			// Ask if user wants to continue
			ui.Warning("Plan execution paused. Changes have been saved.")
			fmt.Println("You can resume later with /plan resume")
			return err
		}

		// Step completed successfully
		step.Status = StepStatusCompleted
		step.Result = truncate(result, 500)
		now = time.Now()
		step.CompletedAt = &now
		plan.Save()

		ui.Success("Step completed")
		ui.Blank()
	}
	return nil
}

func (e *Executor) executeStep(ctx context.Context, step *Step) (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	systemPrompt := fmt.Sprintf("You are a coding agent at %s. Execute the following task and report results.", pwd)

	messages := []types.Message{
		{Role: "user", Content: step.Task},
	}

	// Use llm.Client for streaming
	result, err := e.llmClient.Complete(ctx, messages, systemPrompt, e.registry)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	// If there are tool calls, execute them and continue conversation
	if len(result.ToolCalls) > 0 {
		ui.Blank()
		toolResults := e.llmClient.ExecuteTools(ctx, result.ToolCalls, e.registry)

		// Append assistant message with tool calls
		messages = append(messages, types.Message{
			Role:      "assistant",
			Content:   result.Content,
			ToolCalls: result.ToolCalls,
		})

		// Append tool results
		messages = append(messages, types.Message{Role: "user", Content: toolResults})

		// Continue the conversation to get final response
		result2, err := e.llmClient.Complete(ctx, messages, systemPrompt, e.registry)
		if err != nil {
			return "", fmt.Errorf("LLM call failed: %w", err)
		}
		return result2.Content, nil
	}

	return result.Content, nil
}

func (e *Executor) DisplayPlan(plan *Plan) {
	ui.Divider()
	ui.Box("Plan: "+plan.Name, plan.Goal)
	fmt.Println()
	fmt.Println("Steps:")
	for i, step := range plan.Steps {
		fmt.Printf("  %d. %s\n", i+1, step.Task)
	}
	fmt.Println()
	ui.Divider()
	fmt.Println()
}

func (e *Executor) displaySummary(plan *Plan) {
	ui.Divider()
	ui.Info(fmt.Sprintf("Plan '%s' - Execution Summary", plan.Name))

	total := len(plan.Steps)
	completed := 0
	failed := 0

	for i, step := range plan.Steps {
		switch step.Status {
		case StepStatusCompleted:
			completed++
		case StepStatusFailed:
			failed++
		}
		fmt.Printf("  Step %d: %s [%s]\n", i+1, step.Task, step.Status)
	}

	fmt.Println()
	fmt.Printf("Completed: %d | Failed: %d | Total: %d\n", completed, failed, total)
	ui.Divider()
	fmt.Println()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
