package tests

import (
	"claudego/internal/tools"
	"claudego/pkg/skill"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillToolExecuteLoadsContext(t *testing.T) {
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

	registry := skill.NewRegistry()
	if err := registry.LoadFromDir(tmpDir); err != nil {
		t.Fatalf("LoadFromDir failed: %v", err)
	}

	tool := tools.NewLoadSkillTool(registry)
	if tool.Name() != "load_skill" {
		t.Fatalf("unexpected tool name: %s", tool.Name())
	}
	if tool.Description() != "Load a registered skill into the current context." {
		t.Fatalf("unexpected description: %s", tool.Description())
	}
	inputBytes, err := json.Marshal(tools.SkillInput{
		Skill:   "review",
		Context: "focus on reliability",
	})
	if err != nil {
		t.Fatalf("marshal input failed: %v", err)
	}

	output, err := tool.Execute(context.Background(), inputBytes)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	checks := []string{
		"Skill loaded: review",
		"This skill is now active for the rest of the conversation.",
		"Description: Review code changes for issues",
		"Context:",
		"focus on reliability",
		"Instructions:",
		"Check for bugs, regressions, and missing tests.",
	}
	for _, expected := range checks {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got: %s", expected, output)
		}
	}
}

func TestSkillToolParameters(t *testing.T) {
	tmpDir := t.TempDir()

	skillA := `---
name: refactor
description: Refactor code safely
---
Improve structure without changing behavior.`
	skillB := `---
name: review
description: Review code changes for issues
---
Check for bugs, regressions, and missing tests.`

	for name, content := range map[string]string{"review": skillB, "refactor": skillA} {
		if err := os.MkdirAll(filepath.Join(tmpDir, name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, name, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	registry := skill.NewRegistry()
	if err := registry.LoadFromDir(tmpDir); err != nil {
		t.Fatalf("LoadFromDir failed: %v", err)
	}

	tool := tools.NewLoadSkillTool(registry)
	params := tool.Parameters()
	if params["type"] != "object" {
		t.Fatalf("unexpected type: %v", params["type"])
	}
	properties := params["properties"].(map[string]any)
	if _, ok := properties["skill"]; !ok {
		t.Fatal("expected skill property to exist")
	}
	if _, ok := properties["context"]; !ok {
		t.Fatal("expected context property to exist")
	}
	skillProp := properties["skill"].(map[string]any)
	if skillProp["type"] != "string" {
		t.Fatalf("unexpected skill type: %v", skillProp["type"])
	}
	if skillProp["description"] != "Name of the skill to load" {
		t.Fatalf("unexpected skill description: %v", skillProp["description"])
	}
	contextProp := properties["context"].(map[string]any)
	if contextProp["type"] != "string" {
		t.Fatalf("unexpected context type: %v", contextProp["type"])
	}
	if contextProp["description"] != "Optional request or task context to append to the loaded skill" {
		t.Fatalf("unexpected context description: %v", contextProp["description"])
	}
	required, ok := params["required"].([]string)
	if !ok {
		t.Fatalf("expected required to be []string, got %T", params["required"])
	}
	if len(required) != 1 || required[0] != "skill" {
		t.Fatalf("unexpected required fields: %#v", required)
	}
}
