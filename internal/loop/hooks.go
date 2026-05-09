package loop

import (
	"context"

	"claudego/internal/tools"
	"claudego/pkg/hooks"
)

type hookAdapter struct {
	runner *hooks.Runner
}

func (a hookAdapter) RunPreToolUse(ctx context.Context, input tools.ToolHookInput) (tools.HookDecision, error) {
	result, err := a.runner.RunPreToolUse(ctx, hooks.ToolInput{
		CommonInput: hooks.CommonInput{
			SessionID: input.SessionID,
			CWD:       input.CWD,
		},
		ToolName:  input.ToolName,
		ToolInput: input.ToolInput,
		ToolUseID: input.ToolUseID,
	})
	return toToolHookDecision(result), err
}

func (a hookAdapter) RunPostToolUse(ctx context.Context, input tools.ToolHookInput) (tools.HookDecision, error) {
	result, err := a.runner.RunPostToolUse(ctx, hooks.ToolInput{
		CommonInput: hooks.CommonInput{
			SessionID: input.SessionID,
			CWD:       input.CWD,
		},
		ToolName:     input.ToolName,
		ToolInput:    input.ToolInput,
		ToolUseID:    input.ToolUseID,
		ToolResponse: input.ToolResponse,
	})
	return toToolHookDecision(result), err
}

func toToolHookDecision(result hooks.Result) tools.HookDecision {
	return tools.HookDecision{
		PermissionDecision:       result.PermissionDecision,
		PermissionDecisionReason: result.PermissionDecisionReason,
		AdditionalContext:        result.AdditionalContext,
	}
}
