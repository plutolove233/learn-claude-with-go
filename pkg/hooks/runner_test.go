package hooks

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSessionStartCommandReceivesJSONInput(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "input.json")
	runner := NewRunner(Config{
		EventSessionStart: []Group{
			{
				Matcher: "startup",
				Hooks: []Handler{{
					Type:    "command",
					Command: "cat > " + shellQuote(inputPath) + "; printf 'loaded context'",
				}},
			},
		},
	})

	result, err := runner.RunSessionStart(context.Background(), SessionStartInput{
		CommonInput: CommonInput{
			SessionID: "sess-1",
			CWD:       dir,
		},
		Source: "startup",
		Model:  "test-model",
	})
	if err != nil {
		t.Fatalf("RunSessionStart failed: %v", err)
	}
	if result.AdditionalContext != "loaded context" {
		t.Fatalf("unexpected additional context: %q", result.AdditionalContext)
	}

	raw, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read hook input: %v", err)
	}
	var input SessionStartInput
	if err := json.Unmarshal(raw, &input); err != nil {
		t.Fatalf("unmarshal hook input: %v", err)
	}
	if input.HookEventName != "SessionStart" || input.Source != "startup" || input.Model != "test-model" {
		t.Fatalf("unexpected hook input: %#v", input)
	}
}

func TestPreToolUseCanDenyWithStructuredOutput(t *testing.T) {
	runner := NewRunner(Config{
		EventPreToolUse: []Group{
			{
				Matcher: "bash",
				Hooks: []Handler{{
					Type:    "command",
					Command: "printf '%s' '{\"hookSpecificOutput\":{\"hookEventName\":\"PreToolUse\",\"permissionDecision\":\"deny\",\"permissionDecisionReason\":\"no rm\"}}'",
				}},
			},
		},
	})

	result, err := runner.RunPreToolUse(context.Background(), ToolInput{
		CommonInput: CommonInput{SessionID: "sess-1"},
		ToolName:    "bash",
		ToolInput:   map[string]any{"command": "rm -rf tmp"},
		ToolUseID:   "call-1",
	})
	if err != nil {
		t.Fatalf("RunPreToolUse failed: %v", err)
	}
	if result.PermissionDecision != PermissionDeny || result.PermissionDecisionReason != "no rm" {
		t.Fatalf("unexpected pre-tool decision: %#v", result)
	}
}

func TestPreToolUseExitCodeTwoDeniesTool(t *testing.T) {
	runner := NewRunner(Config{
		EventPreToolUse: []Group{
			{
				Matcher: "*",
				Hooks: []Handler{{
					Type:    "command",
					Command: "printf 'blocked by exit'; exit 2",
				}},
			},
		},
	})

	result, err := runner.RunPreToolUse(context.Background(), ToolInput{
		ToolName:  "bash",
		ToolInput: map[string]any{"command": "rm -rf tmp"},
	})
	if err != nil {
		t.Fatalf("RunPreToolUse failed: %v", err)
	}
	if result.PermissionDecision != PermissionDeny || !strings.Contains(result.PermissionDecisionReason, "blocked by exit") {
		t.Fatalf("expected exit code 2 denial, got %#v", result)
	}
}

func TestPostToolUseReceivesToolResponse(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "post.json")
	runner := NewRunner(Config{
		EventPostToolUse: []Group{
			{
				Matcher: "file_handler",
				Hooks: []Handler{{
					Type:    "command",
					Command: "cat > " + shellQuote(inputPath),
				}},
			},
		},
	})

	_, err := runner.RunPostToolUse(context.Background(), ToolInput{
		CommonInput:  CommonInput{SessionID: "sess-1", CWD: dir},
		ToolName:     "file_handler",
		ToolInput:    map[string]any{"action": "read", "path": "README.md"},
		ToolUseID:    "call-1",
		ToolResponse: map[string]any{"content": "ok"},
	})
	if err != nil {
		t.Fatalf("RunPostToolUse failed: %v", err)
	}

	raw, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read post input: %v", err)
	}
	var input ToolInput
	if err := json.Unmarshal(raw, &input); err != nil {
		t.Fatalf("unmarshal post input: %v", err)
	}
	if input.HookEventName != "PostToolUse" || input.ToolName != "file_handler" || input.ToolResponse["content"] != "ok" {
		t.Fatalf("unexpected post input: %#v", input)
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
