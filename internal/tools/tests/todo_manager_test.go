package tests

import (
	"claudego/internal/tools"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func executeTodoManager(t *testing.T, tool *tools.TodoManager, input tools.TodoManagerInput) (string, error) {
	t.Helper()
	inputBytes, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal input: %v", err)
	}
	return tool.Execute(context.Background(), inputBytes)
}

func TestTodoManagerExecuteSuccess(t *testing.T) {
	tool := tools.NewTodoManager()
	output, err := executeTodoManager(t, tool, tools.TodoManagerInput{
		PlanItems: []tools.PlanItem{
			{Content: "prepare", Status: "pending"},
			{Content: "coding", Status: "in_progress", ActiveForm: "coding"},
			{Content: "done", Status: "completed"},
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	checks := []string{
		"[ ] prepare",
		"[>] coding (coding)",
		"[✓] done",
		"(1/3 completed)",
	}
	for _, expected := range checks {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got: %s", expected, output)
		}
	}
}

func TestTodoManagerRejectsTooManyItems(t *testing.T) {
	tool := tools.NewTodoManager()
	items := make([]tools.PlanItem, 13)
	for i := range items {
		items[i] = tools.PlanItem{Content: "task", Status: "pending"}
	}

	_, err := executeTodoManager(t, tool, tools.TodoManagerInput{PlanItems: items})
	if err == nil {
		t.Fatal("expected too many items error")
	}
	if !strings.Contains(err.Error(), "maximum is 12") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTodoManagerRejectsMultipleInProgress(t *testing.T) {
	tool := tools.NewTodoManager()
	_, err := executeTodoManager(t, tool, tools.TodoManagerInput{
		PlanItems: []tools.PlanItem{
			{Content: "a", Status: "in_progress"},
			{Content: "b", Status: "in_progress"},
		},
	})
	if err == nil {
		t.Fatal("expected multiple in_progress error")
	}
	if !strings.Contains(err.Error(), "only one plan item can be in_progress") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTodoManagerReminderInterval(t *testing.T) {
	tool := tools.NewTodoManager()
	_, err := executeTodoManager(t, tool, tools.TodoManagerInput{
		PlanItems: []tools.PlanItem{
			{Content: "task", Status: "in_progress"},
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	for i := 0; i < tools.INTERVAL-1; i++ {
		if got := tool.NoteRoundWithoutUpdate(); got != "" {
			t.Fatalf("reminder appeared too early at round %d: %q", i+1, got)
		}
	}

	got := tool.NoteRoundWithoutUpdate()
	if !strings.Contains(got, "Refresh your current plan before continuing.") {
		t.Fatalf("expected reminder after interval, got: %q", got)
	}
}

func TestTodoManagerReminderWithoutState(t *testing.T) {
	tool := tools.NewTodoManager()
	if got := tool.NoteRoundWithoutUpdate(); got != "" {
		t.Fatalf("expected empty reminder without state, got: %q", got)
	}
}
