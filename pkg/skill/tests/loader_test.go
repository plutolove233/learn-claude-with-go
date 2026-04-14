package tests

import (
	"claudego/pkg/skill"
	"os"
	"path/filepath"
	"testing"
)

func TestParseMarkdown(t *testing.T) {
	markdown := `---
name: review
description: Review code changes for issues
---
You are a code reviewer. Analyze the git diff.`

	s, err := skill.ParseMarkdown(markdown, "review.md")
	if err != nil {
		t.Fatalf("ParseMarkdown failed: %v", err)
	}

	if s.Name != "review" {
		t.Errorf("expected name 'review', got '%s'", s.Name)
	}
	if s.Description != "Review code changes for issues" {
		t.Errorf("unexpected description: %s", s.Description)
	}
	if s.Instructions != "You are a code reviewer. Analyze the git diff." {
		t.Errorf("unexpected instructions: %s", s.Instructions)
	}
}

func TestParseMarkdownNoFrontmatter(t *testing.T) {
	markdown := `No frontmatter here`
	_, err := skill.ParseMarkdown(markdown, "test.md")
	if err == nil {
		t.Error("expected error for missing frontmatter")
	}
}

func TestParseMarkdownNoName(t *testing.T) {
	markdown := `---
description: No name field
---
Content`
	_, err := skill.ParseMarkdown(markdown, "test.md")
	if err == nil {
		t.Error("expected error for missing name")
	}
}

func TestLoader(t *testing.T) {
	tmpDir := t.TempDir()

	skill1 := `---
name: skill1
description: First test skill
---
Skill 1 content`
	skill2 := `---
name: skill2
description: Second test skill
---
Skill 2 content`

	// New structure: ~/.claude/go/skills/{skill_name}/SKILL.md
	if err := os.MkdirAll(filepath.Join(tmpDir, "skill1"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "skill2"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "skill1", "SKILL.md"), []byte(skill1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "skill2", "SKILL.md"), []byte(skill2), 0644); err != nil {
		t.Fatal(err)
	}

	loader := skill.NewLoader(tmpDir)
	skills, err := loader.LoadSkills()
	if err != nil {
		t.Fatalf("LoadSkills failed: %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(skills))
	}
}
