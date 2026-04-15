package plan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestGenerateID verifies that generateID produces unique identifiers
func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Error("generateID returned empty string")
	}
	if id2 == "" {
		t.Error("generateID returned empty string")
	}
	if id1 == id2 {
		t.Errorf("generateID produced duplicate IDs: %s", id1)
	}
}

// TestCreatePlan verifies plan creation with correct initial state
func TestCreatePlan(t *testing.T) {
	name := "Test Plan"
	goal := "Test Goal"
	steps := []string{"Step 1", "Step 2", "Step 3"}

	plan := CreatePlan(name, goal, steps)

	if plan.ID == "" {
		t.Error("CreatePlan did not generate ID")
	}
	if plan.Name != name {
		t.Errorf("Name = %q, want %q", plan.Name, name)
	}
	if plan.Goal != goal {
		t.Errorf("Goal = %q, want %q", plan.Goal, goal)
	}
	if plan.Status != PlanStatusPending {
		t.Errorf("Status = %q, want %q", plan.Status, PlanStatusPending)
	}
	if len(plan.Steps) != len(steps) {
		t.Errorf("Steps count = %d, want %d", len(plan.Steps), len(steps))
	}
	for i, step := range plan.Steps {
		if step.ID == "" {
			t.Errorf("Step %d has no ID", i)
		}
		if step.Task != steps[i] {
			t.Errorf("Step %d Task = %q, want %q", i, step.Task, steps[i])
		}
		if step.Status != StepStatusPending {
			t.Errorf("Step %d Status = %q, want %q", i, step.Status, StepStatusPending)
		}
	}
	if plan.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
	if plan.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero")
	}
}

// TestPlanSaveAndLoad verifies persistence operations
func TestPlanSaveAndLoad(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	originalDir := DefaultPlansDir()

	// Override DefaultPlansDir for testing
	t.Setenv("HOME", tmpDir)

	plan := CreatePlan("Save Test", "Test saving", []string{"Task 1"})

	// Save the plan
	if err := plan.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	expectedPath := filepath.Join(DefaultPlansDir(), plan.ID+".json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Plan file not created at %s", expectedPath)
	}

	// Load the plan
	loaded, err := LoadPlan(plan.ID)
	if err != nil {
		t.Fatalf("LoadPlan failed: %v", err)
	}

	// Verify loaded plan matches original
	if loaded.ID != plan.ID {
		t.Errorf("Loaded ID = %q, want %q", loaded.ID, plan.ID)
	}
	if loaded.Name != plan.Name {
		t.Errorf("Loaded Name = %q, want %q", loaded.Name, plan.Name)
	}
	if loaded.Goal != plan.Goal {
		t.Errorf("Loaded Goal = %q, want %q", loaded.Goal, plan.Goal)
	}
	if loaded.Status != plan.Status {
		t.Errorf("Loaded Status = %q, want %q", loaded.Status, plan.Status)
	}
	if len(loaded.Steps) != len(plan.Steps) {
		t.Errorf("Loaded Steps count = %d, want %d", len(loaded.Steps), len(plan.Steps))
	}

	// Restore original directory
	_ = originalDir
}

// TestLoadPlan_NonExistent verifies error handling for missing files
func TestLoadPlan_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	_, err := LoadPlan("nonexistent-id")
	if err == nil {
		t.Error("LoadPlan should fail for non-existent plan")
	}
}

// TestListPlans verifies listing all plans
func TestListPlans(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create multiple plans
	plan1 := CreatePlan("Plan 1", "Goal 1", []string{"Task 1"})
	plan2 := CreatePlan("Plan 2", "Goal 2", []string{"Task 2"})
	plan3 := CreatePlan("Plan 3", "Goal 3", []string{"Task 3"})

	if err := plan1.Save(); err != nil {
		t.Fatalf("Failed to save plan1: %v", err)
	}
	if err := plan2.Save(); err != nil {
		t.Fatalf("Failed to save plan2: %v", err)
	}
	if err := plan3.Save(); err != nil {
		t.Fatalf("Failed to save plan3: %v", err)
	}

	// List plans
	plans, err := ListPlans()
	if err != nil {
		t.Fatalf("ListPlans failed: %v", err)
	}

	if len(plans) != 3 {
		t.Errorf("ListPlans returned %d plans, want 3", len(plans))
	}

	// Verify plan IDs are present
	ids := make(map[string]bool)
	for _, p := range plans {
		ids[p.ID] = true
	}
	if !ids[plan1.ID] {
		t.Errorf("Plan1 ID %s not found in list", plan1.ID)
	}
	if !ids[plan2.ID] {
		t.Errorf("Plan2 ID %s not found in list", plan2.ID)
	}
	if !ids[plan3.ID] {
		t.Errorf("Plan3 ID %s not found in list", plan3.ID)
	}
}

// TestListPlans_EmptyDirectory verifies behavior with no plans
func TestListPlans_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	plans, err := ListPlans()
	if err != nil {
		t.Fatalf("ListPlans failed: %v", err)
	}
	if plans != nil {
		t.Errorf("ListPlans returned %v, want nil for empty directory", plans)
	}
}

// TestMarkStepStarted verifies step status transition to in_progress
func TestMarkStepStarted(t *testing.T) {
	plan := CreatePlan("Test", "Goal", []string{"Step 1", "Step 2"})

	beforeMark := time.Now()
	plan.MarkStepStarted(0)
	afterMark := time.Now()

	if plan.Steps[0].Status != StepStatusInProgress {
		t.Errorf("Step 0 Status = %q, want %q", plan.Steps[0].Status, StepStatusInProgress)
	}
	if plan.Steps[0].StartedAt == nil {
		t.Error("Step 0 StartedAt is nil")
	} else {
		if plan.Steps[0].StartedAt.Before(beforeMark) || plan.Steps[0].StartedAt.After(afterMark) {
			t.Errorf("Step 0 StartedAt = %v, want between %v and %v",
				plan.Steps[0].StartedAt, beforeMark, afterMark)
		}
	}
	if plan.CurrentStep != 0 {
		t.Errorf("CurrentStep = %d, want 0", plan.CurrentStep)
	}
	// Verify other steps unchanged
	if plan.Steps[1].Status != StepStatusPending {
		t.Errorf("Step 1 Status = %q, want %q", plan.Steps[1].Status, StepStatusPending)
	}
}

// TestMarkStepStarted_InvalidIndex verifies bounds checking
func TestMarkStepStarted_InvalidIndex(t *testing.T) {
	plan := CreatePlan("Test", "Goal", []string{"Step 1"})

	// Test negative index
	plan.MarkStepStarted(-1)
	if plan.Steps[0].Status != StepStatusPending {
		t.Error("MarkStepStarted with negative index should not modify step")
	}

	// Test out of bounds index
	plan.MarkStepStarted(10)
	if plan.Steps[0].Status != StepStatusPending {
		t.Error("MarkStepStarted with out of bounds index should not modify step")
	}
}

// TestMarkStepCompleted verifies step completion with result and tool calls
func TestMarkStepCompleted(t *testing.T) {
	plan := CreatePlan("Test", "Goal", []string{"Step 1"})
	result := "Task completed successfully"
	toolCalls := []string{"tool1", "tool2"}

	beforeMark := time.Now()
	plan.MarkStepCompleted(0, result, toolCalls)
	afterMark := time.Now()

	if plan.Steps[0].Status != StepStatusCompleted {
		t.Errorf("Step 0 Status = %q, want %q", plan.Steps[0].Status, StepStatusCompleted)
	}
	if plan.Steps[0].Result != result {
		t.Errorf("Step 0 Result = %q, want %q", plan.Steps[0].Result, result)
	}
	if len(plan.Steps[0].ToolCalls) != len(toolCalls) {
		t.Errorf("Step 0 ToolCalls count = %d, want %d", len(plan.Steps[0].ToolCalls), len(toolCalls))
	}
	for i, tc := range plan.Steps[0].ToolCalls {
		if tc != toolCalls[i] {
			t.Errorf("Step 0 ToolCalls[%d] = %q, want %q", i, tc, toolCalls[i])
		}
	}
	if plan.Steps[0].CompletedAt == nil {
		t.Error("Step 0 CompletedAt is nil")
	} else {
		if plan.Steps[0].CompletedAt.Before(beforeMark) || plan.Steps[0].CompletedAt.After(afterMark) {
			t.Errorf("Step 0 CompletedAt = %v, want between %v and %v",
				plan.Steps[0].CompletedAt, beforeMark, afterMark)
		}
	}
}

// TestMarkStepCompleted_InvalidIndex verifies bounds checking
func TestMarkStepCompleted_InvalidIndex(t *testing.T) {
	plan := CreatePlan("Test", "Goal", []string{"Step 1"})

	plan.MarkStepCompleted(-1, "result", nil)
	if plan.Steps[0].Status != StepStatusPending {
		t.Error("MarkStepCompleted with negative index should not modify step")
	}

	plan.MarkStepCompleted(10, "result", nil)
	if plan.Steps[0].Status != StepStatusPending {
		t.Error("MarkStepCompleted with out of bounds index should not modify step")
	}
}

// TestMarkStepFailed verifies step failure with error message
func TestMarkStepFailed(t *testing.T) {
	plan := CreatePlan("Test", "Goal", []string{"Step 1"})
	errMsg := "Task failed due to error"

	beforeMark := time.Now()
	plan.MarkStepFailed(0, errMsg)
	afterMark := time.Now()

	if plan.Steps[0].Status != StepStatusFailed {
		t.Errorf("Step 0 Status = %q, want %q", plan.Steps[0].Status, StepStatusFailed)
	}
	if plan.Steps[0].Error != errMsg {
		t.Errorf("Step 0 Error = %q, want %q", plan.Steps[0].Error, errMsg)
	}
	if plan.Steps[0].CompletedAt == nil {
		t.Error("Step 0 CompletedAt is nil")
	} else {
		if plan.Steps[0].CompletedAt.Before(beforeMark) || plan.Steps[0].CompletedAt.After(afterMark) {
			t.Errorf("Step 0 CompletedAt = %v, want between %v and %v",
				plan.Steps[0].CompletedAt, beforeMark, afterMark)
		}
	}
}

// TestMarkStepFailed_InvalidIndex verifies bounds checking
func TestMarkStepFailed_InvalidIndex(t *testing.T) {
	plan := CreatePlan("Test", "Goal", []string{"Step 1"})

	plan.MarkStepFailed(-1, "error")
	if plan.Steps[0].Status != StepStatusPending {
		t.Error("MarkStepFailed with negative index should not modify step")
	}

	plan.MarkStepFailed(10, "error")
	if plan.Steps[0].Status != StepStatusPending {
		t.Error("MarkStepFailed with out of bounds index should not modify step")
	}
}

// TestIsCompleted verifies completion detection
func TestIsCompleted(t *testing.T) {
	plan := CreatePlan("Test", "Goal", []string{"Step 1", "Step 2", "Step 3"})

	// Initially not completed
	if plan.IsCompleted() {
		t.Error("IsCompleted = true for pending plan, want false")
	}

	// Complete first two steps
	plan.MarkStepCompleted(0, "done", nil)
	plan.MarkStepCompleted(1, "done", nil)

	if plan.IsCompleted() {
		t.Error("IsCompleted = true with incomplete steps, want false")
	}

	// Complete last step
	plan.MarkStepCompleted(2, "done", nil)

	if !plan.IsCompleted() {
		t.Error("IsCompleted = false with all steps completed, want true")
	}
}

// TestIsCompleted_WithFailedStep verifies failed steps prevent completion
func TestIsCompleted_WithFailedStep(t *testing.T) {
	plan := CreatePlan("Test", "Goal", []string{"Step 1", "Step 2"})

	plan.MarkStepCompleted(0, "done", nil)
	plan.MarkStepFailed(1, "error")

	if plan.IsCompleted() {
		t.Error("IsCompleted = true with failed step, want false")
	}
}

// TestHasFailedSteps verifies failure detection
func TestHasFailedSteps(t *testing.T) {
	plan := CreatePlan("Test", "Goal", []string{"Step 1", "Step 2", "Step 3"})

	// Initially no failures
	if plan.HasFailedSteps() {
		t.Error("HasFailedSteps = true for new plan, want false")
	}

	// Complete first step
	plan.MarkStepCompleted(0, "done", nil)
	if plan.HasFailedSteps() {
		t.Error("HasFailedSteps = true with only completed steps, want false")
	}

	// Fail second step
	plan.MarkStepFailed(1, "error")
	if !plan.HasFailedSteps() {
		t.Error("HasFailedSteps = false with failed step, want true")
	}
}

// TestSummary verifies summary string generation
func TestSummary(t *testing.T) {
	plan := CreatePlan("My Plan", "Goal", []string{"Step 1", "Step 2", "Step 3"})

	summary := plan.Summary()
	expected := "Plan 'My Plan': 0/3 steps completed"
	if summary != expected {
		t.Errorf("Summary = %q, want %q", summary, expected)
	}

	// Complete one step
	plan.MarkStepCompleted(0, "done", nil)
	summary = plan.Summary()
	expected = "Plan 'My Plan': 1/3 steps completed"
	if summary != expected {
		t.Errorf("Summary = %q, want %q", summary, expected)
	}

	// Complete all steps
	plan.MarkStepCompleted(1, "done", nil)
	plan.MarkStepCompleted(2, "done", nil)
	summary = plan.Summary()
	expected = "Plan 'My Plan': 3/3 steps completed"
	if summary != expected {
		t.Errorf("Summary = %q, want %q", summary, expected)
	}
}

// TestPlanJSONSerialization verifies JSON marshaling/unmarshaling
func TestPlanJSONSerialization(t *testing.T) {
	plan := CreatePlan("Test Plan", "Test Goal", []string{"Task 1", "Task 2"})
	plan.MarkStepStarted(0)
	plan.MarkStepCompleted(0, "result", []string{"tool1"})

	// Marshal to JSON
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Unmarshal from JSON
	var loaded Plan
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Verify fields
	if loaded.ID != plan.ID {
		t.Errorf("Loaded ID = %q, want %q", loaded.ID, plan.ID)
	}
	if loaded.Name != plan.Name {
		t.Errorf("Loaded Name = %q, want %q", loaded.Name, plan.Name)
	}
	if loaded.Steps[0].Status != plan.Steps[0].Status {
		t.Errorf("Loaded Step 0 Status = %q, want %q", loaded.Steps[0].Status, plan.Steps[0].Status)
	}
	if loaded.Steps[0].Result != plan.Steps[0].Result {
		t.Errorf("Loaded Step 0 Result = %q, want %q", loaded.Steps[0].Result, plan.Steps[0].Result)
	}
}

// TestDefaultPlansDir verifies directory path generation
func TestDefaultPlansDir(t *testing.T) {
	dir := DefaultPlansDir()
	if dir == "" {
		t.Error("DefaultPlansDir returned empty string")
	}
	if !filepath.IsAbs(dir) && dir != ".claudego/plans" {
		t.Errorf("DefaultPlansDir = %q, want absolute path or .claudego/plans", dir)
	}
}

// TestPlanSave_UpdatesTimestamp verifies UpdatedAt is set on save
func TestPlanSave_UpdatesTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	plan := CreatePlan("Test", "Goal", []string{"Task 1"})
	originalUpdatedAt := plan.UpdatedAt

	// Wait a bit to ensure timestamp difference
	time.Sleep(10 * time.Millisecond)

	if err := plan.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if !plan.UpdatedAt.After(originalUpdatedAt) {
		t.Errorf("UpdatedAt not updated: original=%v, current=%v", originalUpdatedAt, plan.UpdatedAt)
	}
}
