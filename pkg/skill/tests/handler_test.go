package tests

import (
	"claudego/internal/tools"
	"claudego/pkg/skill"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMatchAndExecuteLoadsSkillThroughTool(t *testing.T) {
	tmpDir := t.TempDir()

	reviewSkill := `---
name: review
description: Review code changes for issues
---
Check for bugs, regressions, and missing tests.`

	if err := os.MkdirAll(filepath.Join(tmpDir, "review"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "review", "SKILL.md"), []byte(reviewSkill), 0o644); err != nil {
		t.Fatal(err)
	}

	skillRegistry := skill.NewRegistry()
	if err := skillRegistry.LoadFromDir(tmpDir); err != nil {
		t.Fatalf("LoadFromDir failed: %v", err)
	}

	toolRegistry := tools.NewRegistry()
	if err := toolRegistry.Register(tools.NewLoadSkillTool(skillRegistry)); err != nil {
		t.Fatalf("Register skill tool failed: %v", err)
	}

	matched, output, err := skill.MatchAndExecute(context.Background(), "/review focus on safety", skillRegistry, toolRegistry)
	if err != nil {
		t.Fatalf("MatchAndExecute failed: %v", err)
	}
	if !matched {
		t.Fatal("expected slash command to match skill")
	}
	if !strings.Contains(output, "Skill loaded: review") {
		t.Fatalf("expected skill context to be loaded, got: %s", output)
	}
	if !strings.Contains(output, "focus on safety") {
		t.Fatalf("expected user context to be preserved, got: %s", output)
	}
}

func TestMatchAndExecuteIgnoresNonSkillCommands(t *testing.T) {
	skillRegistry := skill.NewRegistry()
	toolRegistry := tools.NewRegistry()
	if err := toolRegistry.Register(tools.NewLoadSkillTool(skillRegistry)); err != nil {
		t.Fatalf("Register skill tool failed: %v", err)
	}

	matched, output, err := skill.MatchAndExecute(context.Background(), "hello world", skillRegistry, toolRegistry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matched {
		t.Fatal("expected non-skill input to be ignored")
	}
	if output != "" {
		t.Fatalf("expected empty output, got %q", output)
	}
}
