package permissions

import (
	"context"
	"testing"
)

type fakePrompter struct {
	decision PromptDecision
	calls    int
}

func (p *fakePrompter) PromptToolPermission(ctx context.Context, req PromptRequest) (PromptDecision, error) {
	p.calls++
	return p.decision, nil
}

func TestDenyWinsOverAskAndAllow(t *testing.T) {
	mgr := NewManager(Config{
		Allow: []string{"bash(*)"},
		Ask:   []string{"bash(*)"},
		Deny:  []string{"bash(sudo *)"},
	}, nil)

	decision := mgr.Decide(context.Background(), Request{
		ToolName:  "bash",
		Arguments: []byte(`{"command":"sudo ls"}`),
	})
	if decision.Action != DecisionDeny {
		t.Fatalf("expected deny, got %#v", decision)
	}
}

func TestBuiltInDefaults(t *testing.T) {
	mgr := NewManager(Config{}, nil)

	read := mgr.Decide(context.Background(), Request{
		ToolName:  "file_handler",
		Arguments: []byte(`{"action":"read","path":"README.md"}`),
	})
	if read.Action != DecisionAllow {
		t.Fatalf("expected read allow, got %#v", read)
	}

	write := mgr.Decide(context.Background(), Request{
		ToolName:  "file_handler",
		Arguments: []byte(`{"action":"write","path":"README.md"}`),
	})
	if write.Action != DecisionDeny {
		t.Fatalf("ask without prompter should deny safely, got %#v", write)
	}

	bash := mgr.Decide(context.Background(), Request{
		ToolName:  "bash",
		Arguments: []byte(`{"command":"git status --short"}`),
	})
	if bash.Action != DecisionDeny {
		t.Fatalf("ask without prompter should deny safely, got %#v", bash)
	}
}

func TestAskApprovalAllowsOnce(t *testing.T) {
	prompter := &fakePrompter{decision: PromptAllowOnce}
	mgr := NewManager(Config{Ask: []string{"bash(*)"}}, prompter)

	decision := mgr.Decide(context.Background(), Request{
		ToolName:  "bash",
		Arguments: []byte(`{"command":"git status --short"}`),
	})
	if decision.Action != DecisionAllow {
		t.Fatalf("expected approval to allow, got %#v", decision)
	}
	if prompter.calls != 1 {
		t.Fatalf("expected one prompt, got %d", prompter.calls)
	}
}

func TestSessionGrantAvoidsSecondPrompt(t *testing.T) {
	prompter := &fakePrompter{decision: PromptAlwaysAllowSession}
	mgr := NewManager(Config{Ask: []string{"bash(*)"}}, prompter)
	req := Request{
		ToolName:  "bash",
		Arguments: []byte(`{"command":"git status --short"}`),
	}

	first := mgr.Decide(context.Background(), req)
	second := mgr.Decide(context.Background(), req)
	if first.Action != DecisionAllow || second.Action != DecisionAllow {
		t.Fatalf("expected both calls to allow, got %#v then %#v", first, second)
	}
	if prompter.calls != 1 {
		t.Fatalf("expected session grant to skip second prompt, got %d prompts", prompter.calls)
	}
}

func TestUnknownToolDefaultsToAsk(t *testing.T) {
	prompter := &fakePrompter{decision: PromptAllowOnce}
	mgr := NewManager(Config{}, prompter)

	decision := mgr.Decide(context.Background(), Request{
		ToolName:  "custom_tool",
		Arguments: []byte(`{"value":"x"}`),
	})
	if decision.Action != DecisionAllow {
		t.Fatalf("expected prompted unknown tool to allow, got %#v", decision)
	}
}
