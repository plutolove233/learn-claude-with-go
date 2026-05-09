package tools

import (
	"claudego/pkg/interfaces"
	"claudego/pkg/permissions"
	"claudego/pkg/types"
	"claudego/pkg/ui"
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

type ToolExecutionOptions struct {
	Permissions interfaces.PermissionDecider
	Hooks       HookRunner
	SessionID   string
	CWD         string
}

type HookRunner interface {
	RunPreToolUse(ctx context.Context, input ToolHookInput) (HookDecision, error)
	RunPostToolUse(ctx context.Context, input ToolHookInput) (HookDecision, error)
}

type ToolHookInput struct {
	Event        string
	SessionID    string
	CWD          string
	ToolName     string
	ToolInput    map[string]any
	ToolUseID    string
	ToolResponse map[string]any
}

type HookDecision struct {
	PermissionDecision       string
	PermissionDecisionReason string
	AdditionalContext        string
}

const HookPermissionDeny = "deny"
const HookPermissionAllow = "allow"

func ExecuteTools(ctx context.Context, toolCalls []openai.ToolCall, registry interfaces.ToolRegistry, options ToolExecutionOptions) []types.ToolCallResult {
	var results []types.ToolCallResult
	enabledTools := registry.EnabledTools()

	for _, tc := range toolCalls {
		fn := tc.Function
		if fn.Name == "" {
			continue
		}

		ui.ToolCall(fn.Name, fn.Arguments)

		input := []byte(fn.Arguments)
		hookToolInput := decodeToolArgumentsForHooks(input)
		var output string
		var toolFound bool
		hookAllowed := false

		if options.Hooks != nil {
			decision, hookErr := options.Hooks.RunPreToolUse(ctx, ToolHookInput{
				Event:     "PreToolUse",
				SessionID: options.SessionID,
				CWD:       options.CWD,
				ToolName:  fn.Name,
				ToolInput: hookToolInput,
				ToolUseID: tc.ID,
			})
			if hookErr != nil {
				output = fmt.Sprintf("Error: pre-tool hook failed for %s: %v", fn.Name, hookErr)
				ui.ToolOutput(output)
				results = append(results, types.ToolCallResult{
					Name:       fn.Name,
					ToolCallID: tc.ID,
					Content:    output,
				})
				continue
			}
			if decision.PermissionDecision == HookPermissionDeny {
				reason := decision.PermissionDecisionReason
				if reason == "" {
					reason = "blocked by hook"
				}
				output = fmt.Sprintf("Error: hook denied %s: %s", fn.Name, reason)
				ui.ToolOutput(output)
				results = append(results, types.ToolCallResult{
					Name:       fn.Name,
					ToolCallID: tc.ID,
					Content:    output,
				})
				continue
			}
			if decision.PermissionDecision == HookPermissionAllow {
				hookAllowed = true
			}
		}

		if options.Permissions != nil && !hookAllowed {
			decision := options.Permissions.Decide(ctx, permissions.Request{
				ToolName:  fn.Name,
				Arguments: input,
			})
			if decision.Action == permissions.DecisionDeny {
				output = fmt.Sprintf("Error: permission denied for %s: %s", fn.Name, decision.Reason)
				ui.ToolOutput(output)
				results = append(results, types.ToolCallResult{
					Name:       fn.Name,
					ToolCallID: tc.ID,
					Content:    output,
				})
				continue
			}
		}

		for _, t := range enabledTools {
			if t.Name() == fn.Name {
				toolFound = true
				out, execErr := t.Execute(ctx, input)
				if execErr != nil {
					output = "Error: " + execErr.Error()
				} else {
					output = out
				}
				break
			}
		}
		if !toolFound {
			output = fmt.Sprintf("Error: tool %q not found or not enabled", fn.Name)
		}

		ui.ToolOutput(output)

		if options.Hooks != nil && toolFound {
			_, hookErr := options.Hooks.RunPostToolUse(ctx, ToolHookInput{
				Event:     "PostToolUse",
				SessionID: options.SessionID,
				CWD:       options.CWD,
				ToolName:  fn.Name,
				ToolInput: hookToolInput,
				ToolUseID: tc.ID,
				ToolResponse: map[string]any{
					"content": output,
				},
			})
			if hookErr != nil {
				output = output + "\nError: post-tool hook failed: " + hookErr.Error()
			}
		}

		results = append(results, types.ToolCallResult{
			Name:       fn.Name,
			ToolCallID: tc.ID,
			Content:    output,
		})
	}
	return results
}

func decodeToolArgumentsForHooks(input []byte) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	var decoded map[string]any
	if err := json.Unmarshal(input, &decoded); err != nil {
		return map[string]any{"raw": string(input)}
	}
	if decoded == nil {
		return map[string]any{}
	}
	return decoded
}
