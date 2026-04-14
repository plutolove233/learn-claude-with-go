package tests

import (
	"claudego/pkg/skill"
	"claudego/pkg/types"
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryRegister(t *testing.T) {
	r := skill.NewRegistry()
	s := &types.Skill{Name: "test", Description: "Test skill"}

	if err := r.Register(s); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if _, ok := r.Get("test"); !ok {
		t.Error("Get failed to retrieve registered skill")
	}
}

func TestRegistryDuplicate(t *testing.T) {
	r := skill.NewRegistry()
	s := &types.Skill{Name: "test", Description: "Test skill"}

	r.Register(s)
	err := r.Register(s)
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestRegistryCompletions(t *testing.T) {
	r := skill.NewRegistry()
	r.Register(&types.Skill{Name: "review"})
	r.Register(&types.Skill{Name: "refactor"})
	r.Register(&types.Skill{Name: "tdd"})

	completions := r.Completions("re")
	if len(completions) != 2 {
		t.Errorf("expected 2 completions, got %d", len(completions))
	}
}

func TestRegistryLoadFromDir(t *testing.T) {
	tmpDir := t.TempDir()

	skill1 := `---
name: skill1
description: First skill
---
Content 1`

	// New structure: ~/.claude/go/skills/{skill_name}/SKILL.md
	if err := os.MkdirAll(filepath.Join(tmpDir, "skill1"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "skill1", "SKILL.md"), []byte(skill1), 0644); err != nil {
		t.Fatal(err)
	}

	r := skill.NewRegistry()
	if err := r.LoadFromDir(tmpDir); err != nil {
		t.Fatalf("LoadFromDir failed: %v", err)
	}

	if _, ok := r.Get("skill1"); !ok {
		t.Error("skill1 not loaded from directory")
	}
}
