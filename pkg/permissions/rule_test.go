package permissions

import "testing"

func TestParseRuleToolWildcard(t *testing.T) {
	rule, err := parseRule("bash(*)")
	if err != nil {
		t.Fatalf("parseRule failed: %v", err)
	}
	if rule.tool != "bash" || rule.kind != ruleKindToolWildcard || rule.pattern != "*" {
		t.Fatalf("unexpected rule: %#v", rule)
	}
}

func TestParseRuleFileHandler(t *testing.T) {
	rule, err := parseRule("file_handler(write:*.go)")
	if err != nil {
		t.Fatalf("parseRule failed: %v", err)
	}
	if rule.tool != "file_handler" || rule.action != "write" || rule.pattern != "*.go" {
		t.Fatalf("unexpected rule: %#v", rule)
	}
}

func TestParseRuleRejectsInvalidShape(t *testing.T) {
	if _, err := parseRule("bash"); err == nil {
		t.Fatal("expected invalid rule to fail")
	}
}

func TestBashRuleMatchesCommand(t *testing.T) {
	rule, err := parseRule("bash(git status*)")
	if err != nil {
		t.Fatalf("parseRule failed: %v", err)
	}
	if !rule.matches(Request{ToolName: "bash", Arguments: []byte(`{"command":"git status --short"}`)}) {
		t.Fatal("expected rule to match git status command")
	}
	if rule.matches(Request{ToolName: "bash", Arguments: []byte(`{"command":"go test ./..."}`)}) {
		t.Fatal("did not expect rule to match go test command")
	}
}

func TestFileHandlerRuleMatchesActionAndPath(t *testing.T) {
	rule, err := parseRule("file_handler(write:*.go)")
	if err != nil {
		t.Fatalf("parseRule failed: %v", err)
	}
	if !rule.matches(Request{ToolName: "file_handler", Arguments: []byte(`{"action":"write","path":"main.go"}`)}) {
		t.Fatal("expected write to main.go to match")
	}
	if rule.matches(Request{ToolName: "file_handler", Arguments: []byte(`{"action":"read","path":"main.go"}`)}) {
		t.Fatal("did not expect read action to match write rule")
	}
}

func TestRuleWildcardSubstringCompatibility(t *testing.T) {
	rule, err := parseRule("file_handler(*:.env*)")
	if err != nil {
		t.Fatalf("parseRule failed: %v", err)
	}
	if !rule.matches(Request{ToolName: "file_handler", Arguments: []byte(`{"action":"read","path":".env.local"}`)}) {
		t.Fatal("expected .env.local to match .env*")
	}
}
