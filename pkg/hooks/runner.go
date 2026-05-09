package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	EventSessionStart = "SessionStart"
	EventPreToolUse   = "PreToolUse"
	EventPostToolUse  = "PostToolUse"

	PermissionAllow = "allow"
	PermissionDeny  = "deny"
)

type Config map[string][]Group

type Group struct {
	Matcher string    `json:"matcher,omitempty"`
	Hooks   []Handler `json:"hooks"`
}

type Handler struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	If      string `json:"if,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

type CommonInput struct {
	HookEventName  string `json:"hook_event_name"`
	SessionID      string `json:"session_id,omitempty"`
	TranscriptPath string `json:"transcript_path,omitempty"`
	CWD            string `json:"cwd,omitempty"`
}

type SessionStartInput struct {
	CommonInput
	Source string `json:"source"`
	Model  string `json:"model,omitempty"`
}

type ToolInput struct {
	CommonInput
	ToolName     string         `json:"tool_name"`
	ToolInput    map[string]any `json:"tool_input"`
	ToolUseID    string         `json:"tool_use_id,omitempty"`
	ToolResponse map[string]any `json:"tool_response,omitempty"`
}

type Result struct {
	PermissionDecision       string
	PermissionDecisionReason string
	AdditionalContext        string
}

type Runner struct {
	config Config
}

func NewRunner(config Config) *Runner {
	return &Runner{config: config}
}

func (r *Runner) RunSessionStart(ctx context.Context, input SessionStartInput) (Result, error) {
	input.HookEventName = EventSessionStart
	return r.run(ctx, EventSessionStart, input.Source, input, nil)
}

func (r *Runner) RunPreToolUse(ctx context.Context, input ToolInput) (Result, error) {
	input.HookEventName = EventPreToolUse
	return r.run(ctx, EventPreToolUse, input.ToolName, input, &input)
}

func (r *Runner) RunPostToolUse(ctx context.Context, input ToolInput) (Result, error) {
	input.HookEventName = EventPostToolUse
	return r.run(ctx, EventPostToolUse, input.ToolName, input, &input)
}

func (r *Runner) run(ctx context.Context, eventName, matchValue string, input any, toolInput *ToolInput) (Result, error) {
	if r == nil {
		return Result{}, nil
	}

	var combined Result
	for _, group := range r.config[eventName] {
		if !matches(group.Matcher, matchValue) {
			continue
		}
		for _, handler := range group.Hooks {
			if handler.Type != "" && handler.Type != "command" {
				continue
			}
			if handler.Command == "" {
				continue
			}
			if toolInput != nil && handler.If != "" && !matchesToolCondition(handler.If, *toolInput) {
				continue
			}
			result, err := runCommandHook(ctx, handler, input)
			if err != nil {
				return combined, err
			}
			combined = combine(eventName, combined, result)
		}
	}
	return combined, nil
}

func combine(eventName string, current, next Result) Result {
	if next.AdditionalContext != "" {
		if current.AdditionalContext != "" {
			current.AdditionalContext += "\n"
		}
		current.AdditionalContext += next.AdditionalContext
	}
	if eventName == EventPreToolUse {
		if next.PermissionDecision == PermissionDeny {
			current.PermissionDecision = PermissionDeny
			current.PermissionDecisionReason = next.PermissionDecisionReason
			return current
		}
		if current.PermissionDecision == "" && next.PermissionDecision != "" {
			current.PermissionDecision = next.PermissionDecision
			current.PermissionDecisionReason = next.PermissionDecisionReason
		}
	}
	return current
}

func runCommandHook(ctx context.Context, handler Handler, input any) (Result, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return Result{}, fmt.Errorf("marshal hook input: %w", err)
	}

	if handler.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(handler.Timeout)*time.Second)
		defer cancel()
	}

	cmd := shellCommand(ctx, handler.Command)
	cmd.Stdin = bytes.NewReader(payload)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	output := strings.TrimSpace(stdout.String())
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 2 {
			reason := output
			if reason == "" {
				reason = strings.TrimSpace(stderr.String())
			}
			if reason == "" {
				reason = "hook exited with code 2"
			}
			return Result{PermissionDecision: PermissionDeny, PermissionDecisionReason: reason}, nil
		}
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
		return Result{}, fmt.Errorf("hook command failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	result := Result{AdditionalContext: output}
	if parsed, ok := parseHookOutput(output); ok {
		result = parsed
	}
	return result, nil
}

func shellCommand(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/C", command)
	}
	return exec.CommandContext(ctx, "sh", "-c", command)
}

func parseHookOutput(output string) (Result, bool) {
	if output == "" || !strings.HasPrefix(strings.TrimSpace(output), "{") {
		return Result{}, false
	}
	var raw struct {
		Decision           string `json:"decision"`
		Reason             string `json:"reason"`
		AdditionalContext  string `json:"additionalContext"`
		HookSpecificOutput struct {
			HookEventName            string `json:"hookEventName"`
			PermissionDecision       string `json:"permissionDecision"`
			PermissionDecisionReason string `json:"permissionDecisionReason"`
			AdditionalContext        string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(output), &raw); err != nil {
		return Result{}, false
	}

	result := Result{
		PermissionDecision:       raw.HookSpecificOutput.PermissionDecision,
		PermissionDecisionReason: raw.HookSpecificOutput.PermissionDecisionReason,
		AdditionalContext:        raw.AdditionalContext,
	}
	if result.AdditionalContext == "" {
		result.AdditionalContext = raw.HookSpecificOutput.AdditionalContext
	}
	if result.PermissionDecision == "" {
		switch raw.Decision {
		case "block":
			result.PermissionDecision = PermissionDeny
		case "approve":
			result.PermissionDecision = PermissionAllow
		default:
			result.PermissionDecision = raw.Decision
		}
		result.PermissionDecisionReason = raw.Reason
	}
	return result, true
}

func matches(pattern, value string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" || pattern == "*" {
		return true
	}
	if isExactMatcher(pattern) {
		for _, part := range strings.Split(pattern, "|") {
			if strings.TrimSpace(part) == value {
				return true
			}
		}
		return false
	}
	matched, err := regexp.MatchString(pattern, value)
	return err == nil && matched
}

func isExactMatcher(pattern string) bool {
	for _, r := range pattern {
		if !(r == '|' || r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func matchesToolCondition(condition string, input ToolInput) bool {
	open := strings.Index(condition, "(")
	if open < 1 || !strings.HasSuffix(condition, ")") {
		return false
	}
	toolName := strings.TrimSpace(condition[:open])
	if toolName != input.ToolName {
		return false
	}
	pattern := strings.TrimSpace(condition[open+1 : len(condition)-1])
	if pattern == "" || pattern == "*" {
		return true
	}
	raw, _ := json.Marshal(input.ToolInput)
	return wildcardMatch(pattern, string(raw))
}

func wildcardMatch(pattern, value string) bool {
	var b strings.Builder
	b.WriteString("(?s)")
	for _, r := range pattern {
		switch r {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteByte('.')
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	matched, err := regexp.MatchString(b.String(), value)
	return err == nil && matched
}
