package permissions

import (
	"context"
	"fmt"
	"sync"
)

type Config struct {
	Allow []string `json:"allow,omitempty"`
	Ask   []string `json:"ask,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

type Request struct {
	ToolName  string
	Arguments []byte
}

type DecisionAction string

const (
	DecisionAllow DecisionAction = "allow"
	DecisionDeny  DecisionAction = "deny"
)

type Decision struct {
	Action DecisionAction
	Reason string
}

type PromptDecision string

const (
	PromptAllowOnce          PromptDecision = "allow_once"
	PromptAlwaysAllowSession PromptDecision = "always_allow_session"
	PromptDeny               PromptDecision = "deny"
)

type PromptRequest struct {
	ToolName  string
	Arguments []byte
	Reason    string
}

type Prompter interface {
	PromptToolPermission(ctx context.Context, req PromptRequest) (PromptDecision, error)
}

type Manager struct {
	allow    []rule
	ask      []rule
	deny     []rule
	prompter Prompter

	mu            sync.RWMutex
	sessionGrants []rule
}

func NewManager(cfg Config, prompter Prompter) *Manager {
	return &Manager{
		allow:    parseRuleList(cfg.Allow),
		ask:      parseRuleList(cfg.Ask),
		deny:     parseRuleList(cfg.Deny),
		prompter: prompter,
	}
}

func (m *Manager) Decide(ctx context.Context, req Request) Decision {
	if matched := firstMatch(m.deny, req); matched != "" {
		return Decision{Action: DecisionDeny, Reason: "matched deny rule " + matched}
	}
	if matched := firstMatch(builtInDenyRules(), req); matched != "" {
		return Decision{Action: DecisionDeny, Reason: "matched built-in deny rule " + matched}
	}
	if matched := firstMatch(m.sessionGrantRules(), req); matched != "" {
		return Decision{Action: DecisionAllow, Reason: "matched session allow rule " + matched}
	}
	if matched := firstMatch(m.ask, req); matched != "" {
		return m.askUser(ctx, req, "matched ask rule "+matched)
	}
	if matched := firstMatch(m.allow, req); matched != "" {
		return Decision{Action: DecisionAllow, Reason: "matched allow rule " + matched}
	}
	if matched := firstMatch(builtInAllowRules(), req); matched != "" {
		return Decision{Action: DecisionAllow, Reason: "matched built-in allow rule " + matched}
	}
	if matched := firstMatch(builtInAskRules(), req); matched != "" {
		return m.askUser(ctx, req, "matched built-in ask rule "+matched)
	}
	return m.askUser(ctx, req, "unknown tool requires permission")
}

func (m *Manager) askUser(ctx context.Context, req Request, reason string) Decision {
	if m.prompter == nil {
		return Decision{Action: DecisionDeny, Reason: reason + "; no permission prompter configured"}
	}
	choice, err := m.prompter.PromptToolPermission(ctx, PromptRequest{
		ToolName:  req.ToolName,
		Arguments: req.Arguments,
		Reason:    reason,
	})
	if err != nil {
		return Decision{Action: DecisionDeny, Reason: fmt.Sprintf("%s; prompt failed: %v", reason, err)}
	}
	switch choice {
	case PromptAllowOnce:
		return Decision{Action: DecisionAllow, Reason: reason + "; user allowed once"}
	case PromptAlwaysAllowSession:
		m.addSessionGrant(req)
		return Decision{Action: DecisionAllow, Reason: reason + "; user allowed for session"}
	default:
		return Decision{Action: DecisionDeny, Reason: reason + "; user denied request"}
	}
}

func (m *Manager) addSessionGrant(req Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessionGrants = append(m.sessionGrants, rule{
		raw:     req.ToolName + "(*)",
		tool:    req.ToolName,
		kind:    ruleKindToolWildcard,
		pattern: "*",
	})
}

func (m *Manager) sessionGrantRules() []rule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]rule, len(m.sessionGrants))
	copy(out, m.sessionGrants)
	return out
}

func parseRuleList(rawRules []string) []rule {
	rules := make([]rule, 0, len(rawRules))
	for _, raw := range rawRules {
		r, err := parseRule(raw)
		if err == nil {
			rules = append(rules, r)
		}
	}
	return rules
}

func firstMatch(rules []rule, req Request) string {
	for _, r := range rules {
		if r.matches(req) {
			return r.raw
		}
	}
	return ""
}

func builtInDenyRules() []rule {
	return nil
}

func builtInAllowRules() []rule {
	return []rule{
		mustParseRule("file_handler(read:*)"),
		mustParseRule("load_skill(*)"),
		mustParseRule("todo_manager(*)"),
	}
}

func builtInAskRules() []rule {
	return []rule{
		mustParseRule("file_handler(write:*)"),
		mustParseRule("bash(*)"),
	}
}
