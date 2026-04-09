package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"

	"claudego/internal/config"
)

type Planner struct {
	client openai.Client
	model  string
}

func NewPlanner(cfg *config.Config) *Planner {
	client := openai.NewClient(
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(cfg.BaseURL),
	)
	return &Planner{client: client, model: cfg.Model}
}

type PlanningRequest struct {
	Goal string
	PWD  string
}

type PlanningResponse struct {
	Name  string   `json:"name"`
	Steps []string `json:"steps"`
}

func (p *Planner) CreatePlan(ctx context.Context, req PlanningRequest) (*Plan, error) {
	systemPrompt := `You are a task planning assistant. Given a user's goal, break it down into clear, executable steps.
Each step should be a concrete action that can be performed in a single iteration.

Respond ONLY with valid JSON in this exact format:
{
  "name": "Brief plan name",
  "steps": ["Step 1 description", "Step 2 description", "..."]
}

Guidelines for steps:
- Each step should be self-contained and actionable
- Steps should be ordered logically (prerequisites first)
- Aim for 3-8 steps total
- Use clear, specific language

Example:
Goal: "Refactor the logging system to use logrus"
{
  "name": "Logging System Refactoring",
  "steps": [
    "Analyze current logging implementation in pkg/logger",
    "Create new logrus-based logger with required features",
    "Update all log calls to use new logger",
    "Verify all tests pass"
  ]
}`

	userPrompt := fmt.Sprintf("Goal: %s\nWorking directory: %s", req.Goal, req.PWD)

	stream := p.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
		Model: shared.ChatModel(p.model),
	})
	defer stream.Close()

	var fullContent strings.Builder
	for stream.Next() {
		event := stream.Current()
		if len(event.Choices) == 0 {
			continue
		}
		if delta := event.Choices[0].Delta.Content; delta != "" {
			fullContent.WriteString(delta)
		}
	}

	if stream.Err() != nil {
		return nil, fmt.Errorf("planning LLM call failed: %w", stream.Err())
	}

	// Parse the response to extract JSON
	response := fullContent.String()
	response = strings.TrimSpace(response)
	// 删去被<think></think>包裹的内容，因为它们不是JSON的一部分
	if strings.Contains(response, "<think>") && strings.Contains(response, "</think>") {
		start := strings.Index(response, "<think>") + len("<think>")
		end := strings.Index(response, "</think>")
		if end > start {
			response = response[:strings.Index(response, "<think>")] + response[end+len("</think>"):]
		}
	}

	// Try to extract JSON from markdown code blocks if present
	if strings.Contains(response, "```json") {
		start := strings.Index(response, "```json") + 7
		end := strings.LastIndex(response, "```")
		if end > start {
			response = response[start:end]
		}
	} else if strings.Contains(response, "```") {
		start := strings.Index(response, "```") + 3
		end := strings.LastIndex(response, "```")
		if end > start {
			response = response[start:end]
		}
	}

	response = strings.TrimSpace(response)

	var parsed PlanningResponse
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse plan JSON: %w\nResponse was: %s", err, response)
	}

	if len(parsed.Steps) == 0 {
		return nil, fmt.Errorf("LLM returned plan with no steps")
	}

	// Create the plan
	plan := CreatePlan(parsed.Name, req.Goal, parsed.Steps)
	return plan, nil
}
