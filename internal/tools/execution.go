package tools

import (
	"claudego/pkg/interfaces"
	"claudego/pkg/permissions"
	"claudego/pkg/types"
	"claudego/pkg/ui"
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

type ToolExecutionOptions struct {
	Permissions interfaces.PermissionDecider
}

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
		var output string
		var toolFound bool

		if options.Permissions != nil {
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

		results = append(results, types.ToolCallResult{
			Name:       fn.Name,
			ToolCallID: tc.ID,
			Content:    output,
		})
	}
	return results
}
