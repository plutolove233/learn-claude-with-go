package plan

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func generateID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type PlanStatus string
type StepStatus string

const (
	PlanStatusPending    PlanStatus = "pending"
	PlanStatusInProgress PlanStatus = "in_progress"
	PlanStatusCompleted  PlanStatus = "completed"
	PlanStatusFailed     PlanStatus = "failed"
	PlanStatusPaused     PlanStatus = "paused"
)

const (
	StepStatusPending    StepStatus = "pending"
	StepStatusInProgress StepStatus = "in_progress"
	StepStatusCompleted  StepStatus = "completed"
	StepStatusFailed     StepStatus = "failed"
)

type Step struct {
	ID          string     `json:"id"`
	Task        string     `json:"task"`
	Status      StepStatus `json:"status"`
	Result      string     `json:"result,omitempty"`
	ToolCalls   []string   `json:"tool_calls,omitempty"`
	Error       string     `json:"error,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type Plan struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Goal        string     `json:"goal"`
	Steps       []Step     `json:"steps"`
	Status      PlanStatus `json:"status"`
	CurrentStep int        `json:"current_step"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func DefaultPlansDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".claudego/plans"
	}
	return filepath.Join(home, ".claudego", "plans")
}

func (p *Plan) Save() error {
	dir := DefaultPlansDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create plans dir: %w", err)
	}

	p.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	path := filepath.Join(dir, p.ID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write plan file: %w", err)
	}
	return nil
}

func LoadPlan(id string) (*Plan, error) {
	path := filepath.Join(DefaultPlansDir(), id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	var p Plan
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}
	return &p, nil
}

func ListPlans() ([]*Plan, error) {
	dir := DefaultPlansDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read plans dir: %w", err)
	}

	var plans []*Plan
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := entry.Name()[:len(entry.Name())-5]
		p, err := LoadPlan(id)
		if err != nil {
			continue
		}
		plans = append(plans, p)
	}
	return plans, nil
}

func CreatePlan(name, goal string, steps []string) *Plan {
	now := time.Now()
	plan := &Plan{
		ID:        generateID(),
		Name:      name,
		Goal:      goal,
		Status:    PlanStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	for _, task := range steps {
		plan.Steps = append(plan.Steps, Step{
			ID:     generateID(),
			Task:   task,
			Status: StepStatusPending,
		})
	}
	return plan
}

func (p *Plan) MarkStepStarted(stepIdx int) {
	if stepIdx < 0 || stepIdx >= len(p.Steps) {
		return
	}
	now := time.Now()
	p.Steps[stepIdx].Status = StepStatusInProgress
	p.Steps[stepIdx].StartedAt = &now
	p.CurrentStep = stepIdx
}

func (p *Plan) MarkStepCompleted(stepIdx int, result string, toolCalls []string) {
	if stepIdx < 0 || stepIdx >= len(p.Steps) {
		return
	}
	now := time.Now()
	p.Steps[stepIdx].Status = StepStatusCompleted
	p.Steps[stepIdx].Result = result
	p.Steps[stepIdx].ToolCalls = toolCalls
	p.Steps[stepIdx].CompletedAt = &now
}

func (p *Plan) MarkStepFailed(stepIdx int, errMsg string) {
	if stepIdx < 0 || stepIdx >= len(p.Steps) {
		return
	}
	now := time.Now()
	p.Steps[stepIdx].Status = StepStatusFailed
	p.Steps[stepIdx].Error = errMsg
	p.Steps[stepIdx].CompletedAt = &now
}

func (p *Plan) IsCompleted() bool {
	for _, step := range p.Steps {
		if step.Status != StepStatusCompleted {
			return false
		}
	}
	return true
}

func (p *Plan) HasFailedSteps() bool {
	for _, step := range p.Steps {
		if step.Status == StepStatusFailed {
			return true
		}
	}
	return false
}

func (p *Plan) Summary() string {
	total := len(p.Steps)
	completed := 0
	for _, s := range p.Steps {
		if s.Status == StepStatusCompleted {
			completed++
		}
	}
	return fmt.Sprintf("Plan '%s': %d/%d steps completed", p.Name, completed, total)
}
