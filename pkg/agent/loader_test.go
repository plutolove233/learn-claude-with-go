package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMarkdown(t *testing.T) {
	markdown := `---
name: test-agent
description: A test agent for unit testing
tools:
  - tool1
  - tool2
---
# Instructions

This is the agent content.
`

	agent, err := ParseMarkdown(markdown, "test.md")
	if err != nil {
		t.Fatalf("ParseMarkdown failed: %v", err)
	}

	if agent.Name != "test-agent" {
		t.Errorf("expected name 'test-agent', got '%s'", agent.Name)
	}
	if agent.Description != "A test agent for unit testing" {
		t.Errorf("unexpected description: %s", agent.Description)
	}
	if len(agent.Tools) != 2 || agent.Tools[0] != "tool1" || agent.Tools[1] != "tool2" {
		t.Errorf("unexpected tools: %v", agent.Tools)
	}
	if agent.Instructions != "# Instructions\n\nThis is the agent content." {
		t.Errorf("unexpected instructions: %s", agent.Instructions)
	}
}

func TestLoader(t *testing.T) {
	// Create temp directory with test agents
	tmpDir := t.TempDir()

	agent1 := `---
name: agent1
description: First test agent
---
Agent 1 content`
	agent2 := `---
name: agent2
description: Second test agent
---
Agent 2 content`

	if err := os.WriteFile(filepath.Join(tmpDir, "agent1.md"), []byte(agent1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "agent2.md"), []byte(agent2), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(tmpDir)
	agents, err := loader.LoadAgents()
	if err != nil {
		t.Fatalf("LoadAgents failed: %v", err)
	}

	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(agents))
	}
}
