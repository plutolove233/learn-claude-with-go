package tools

import (
	"testing"
)

func TestBashTool_Name(t *testing.T) {
	b := NewBashTool()
	if b.Name() != "bash" {
		t.Errorf("Name() = %q, want %q", b.Name(), "bash")
	}
}

func TestBashTool_Description(t *testing.T) {
	b := NewBashTool()
	if b.Description() == "" {
		t.Error("Description() returned empty string")
	}
}

func TestBashTool_Execute_Simple(t *testing.T) {
	b := NewBashTool()
	out, err := b.Execute(map[string]interface{}{"command": "echo hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty output")
	}
}

func TestBashTool_Execute_MissingCommand(t *testing.T) {
	b := NewBashTool()
	_, err := b.Execute(map[string]interface{}{})
	if err == nil {
		t.Error("expected error for missing command")
	}
}

func TestBashTool_Dangerous(t *testing.T) {
	b := NewBashTool()
	_, err := b.Execute(map[string]interface{}{"command": "rm -rf /"})
	if err == nil {
		t.Error("expected error for dangerous command")
	}
}

func TestBashTool_Dangerous_Sudo(t *testing.T) {
	b := NewBashTool()
	_, err := b.Execute(map[string]interface{}{"command": "sudo echo hello"})
	if err == nil {
		t.Error("expected error for sudo command")
	}
}
