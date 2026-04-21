package tools

import (
	"claudego/pkg/types"
	"fmt"
	"strings"
)

const INTERVAL int = 3

type PlanItem struct {
	Content    string `json:"content" validate:"required"`
	Status     string `json:"status" validate:"required,oneof=pending in_progress completed"`
	ActiveForm string `json:"active_form,omitempty"` //Optional present-continuous label.
}

type TodoManagerInput struct {
	PlanItems []PlanItem `json:"plan_items" validate:"required"`
}

type TodoManager struct {
	BaseTool[TodoManagerInput]
	state            []PlanItem
	roundSinceUpdate int
	waitTodo         int
}

func NewTodoManager() *TodoManager {
	t := &TodoManager{
		BaseTool: BaseTool[TodoManagerInput]{
			name:          "todo_manager",
			description:   "Manage a list of to-do items with their status. Each item has content and a status (pending, in_progress, completed). Optionally, an active_form can be included for items that require user input.",
			extraValidate: nil,
		},
		roundSinceUpdate: 0,
		waitTodo:         0,
	}
	t.fn = t.todoManagerExecute
	return t
}

func (t *TodoManager) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"plan_items": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"content": map[string]any{
							"type":        "string",
							"description": "The content of the to-do item.",
						},
						"status": map[string]any{
							"type":        "string",
							"enum":        []string{"pending", "in_progress", "completed"},
							"description": "The status of the to-do item.",
						},
						"active_form": map[string]any{
							"type":        "string",
							"description": "Optional present-continuous label.",
						},
					},
					"required": []string{"content", "status"},
				},
			},
		},
	}
}

func (t *TodoManager) Metadata() types.ToolMetadata {
	return types.ToolMetadata{
		Category:   types.CategoryProcess,
		SafeToSkip: false,
		MaxRetries: 0,
	}
}

func (t *TodoManager) todoManagerExecute(input TodoManagerInput) (string, error) {
	t.roundSinceUpdate = 0
	t.waitTodo = 0
	if len(input.PlanItems) > 12 {
		return "", fmt.Errorf("too many plan items provided, maximum is 12")
	}
	in_progress_count := 0

	for _, item := range input.PlanItems {
		if item.Status == "in_progress" {
			in_progress_count++
		}
		if item.Status == "pending" {
			t.waitTodo++
		}
	}

	if in_progress_count > 1 {
		return "", fmt.Errorf("only one plan item can be in_progress")
	}

	t.state = input.PlanItems
	t.roundSinceUpdate = 0

	return t.render(), nil
}

func (t *TodoManager) render() string {
	planItems := t.state
	if len(planItems) == 0 {
		return "No session plan yet."
	}
	var dashboard strings.Builder
	for _, item := range planItems {
		marker := map[string]string{
			"pending":     "- [ ]",
			"in_progress": "- [>]",
			"completed":   "- [✓]",
		}[item.Status]
		line := fmt.Sprintf("%s %s", marker, item.Content)
		if item.Status == "in_progress" && item.ActiveForm != "" {
			line += fmt.Sprintf(" (%s)", item.ActiveForm)
		}
		fmt.Fprintf(&dashboard, "%s\r\n", line)
	}
	completedCount := 0
	for _, item := range planItems {
		if item.Status == "completed" {
			completedCount++
		}
	}
	fmt.Fprintf(&dashboard, "\n(%d/%d completed)", completedCount, len(planItems))
	return dashboard.String()
}

func (t *TodoManager) NoteRoundWithoutUpdate() string {
	t.roundSinceUpdate++
	if len(t.state) == 0 {
		return ""
	}
	if t.waitTodo <= 0 {
		return ""
	}
	if t.roundSinceUpdate < INTERVAL {
		return ""
	}
	return "<reminder>Refresh your current plan before continuing.</reminder>"
}
