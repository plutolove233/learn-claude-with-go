package tools

import (
	"context"
	"encoding/json"
	"testing"

	"claudego/pkg/permissions"
	"claudego/pkg/types"

	"github.com/sashabaranov/go-openai"
)

type hookTestTool struct {
	BaseTool[hookTestInput]
	calls int
}

type hookTestInput struct {
	Value string `json:"value"`
}

func newHookTestTool() *hookTestTool {
	t := &hookTestTool{}
	t.BaseTool = BaseTool[hookTestInput]{
		name:        "sample",
		description: "sample test tool",
		fn: func(input hookTestInput) (string, error) {
			t.calls++
			return "tool output", nil
		},
	}
	return t
}

type fakeHookRunner struct {
	preDecision HookDecision
	preCalls    int
	postCalls   int
	postInput   ToolHookInput
}

type denyingDecider struct{}

func (d denyingDecider) Decide(ctx context.Context, req permissions.Request) permissions.Decision {
	return permissions.Decision{Action: permissions.DecisionDeny, Reason: "permission denied"}
}

func (r *fakeHookRunner) RunPreToolUse(ctx context.Context, input ToolHookInput) (HookDecision, error) {
	r.preCalls++
	return r.preDecision, nil
}

func (r *fakeHookRunner) RunPostToolUse(ctx context.Context, input ToolHookInput) (HookDecision, error) {
	r.postCalls++
	r.postInput = input
	return HookDecision{}, nil
}

func TestExecuteToolsSkipsToolWhenPreHookDenies(t *testing.T) {
	registry := NewRegistry()
	tool := newHookTestTool()
	if err := registry.Register(tool); err != nil {
		t.Fatalf("register tool: %v", err)
	}
	runner := &fakeHookRunner{
		preDecision: HookDecision{
			PermissionDecision:       HookPermissionDeny,
			PermissionDecisionReason: "blocked",
		},
	}

	results := ExecuteTools(context.Background(), []openai.ToolCall{toolCall("call-1", "sample", `{"value":"x"}`)}, registry, ToolExecutionOptions{
		Hooks:     runner,
		SessionID: "sess-1",
		CWD:       "/tmp/project",
	})

	if tool.calls != 0 {
		t.Fatalf("expected denied tool not to execute")
	}
	if runner.preCalls != 1 || runner.postCalls != 0 {
		t.Fatalf("unexpected hook calls: pre=%d post=%d", runner.preCalls, runner.postCalls)
	}
	if len(results) != 1 || results[0].Content != "Error: hook denied sample: blocked" {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func TestExecuteToolsRunsPostHookWithResult(t *testing.T) {
	registry := NewRegistry()
	tool := newHookTestTool()
	if err := registry.Register(tool); err != nil {
		t.Fatalf("register tool: %v", err)
	}
	runner := &fakeHookRunner{}

	results := ExecuteTools(context.Background(), []openai.ToolCall{toolCall("call-1", "sample", `{"value":"x"}`)}, registry, ToolExecutionOptions{
		Hooks:     runner,
		SessionID: "sess-1",
		CWD:       "/tmp/project",
	})

	if len(results) != 1 || results[0].Content != "tool output" {
		t.Fatalf("unexpected results: %#v", results)
	}
	if runner.postCalls != 1 {
		t.Fatalf("expected post hook to run once, got %d", runner.postCalls)
	}
	if runner.postInput.Event != "PostToolUse" || runner.postInput.ToolName != "sample" || runner.postInput.ToolResponse["content"] != "tool output" {
		t.Fatalf("unexpected post hook input: %#v", runner.postInput)
	}
}

func TestExecuteToolsLetsPreHookAllowBypassPermissionPrompt(t *testing.T) {
	registry := NewRegistry()
	tool := newHookTestTool()
	if err := registry.Register(tool); err != nil {
		t.Fatalf("register tool: %v", err)
	}
	runner := &fakeHookRunner{
		preDecision: HookDecision{PermissionDecision: HookPermissionAllow},
	}

	results := ExecuteTools(context.Background(), []openai.ToolCall{toolCall("call-1", "sample", `{"value":"x"}`)}, registry, ToolExecutionOptions{
		Permissions: denyingDecider{},
		Hooks:       runner,
	})

	if len(results) != 1 || results[0].Content != "tool output" {
		t.Fatalf("expected hook allow to bypass permissions, got %#v", results)
	}
}

func toolCall(id, name, args string) openai.ToolCall {
	return openai.ToolCall{
		ID:   id,
		Type: openai.ToolTypeFunction,
		Function: openai.FunctionCall{
			Name:      name,
			Arguments: args,
		},
	}
}

func TestDecodeToolArgumentsForHooksFallsBackToRawJSON(t *testing.T) {
	raw := json.RawMessage(`{"value":`)
	input := decodeToolArgumentsForHooks([]byte(raw))
	if input["raw"] != string(raw) {
		t.Fatalf("unexpected decoded input: %#v", input)
	}
}

var _ types.ToolCallResult
