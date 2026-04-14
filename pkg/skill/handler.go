package skill

import (
	"context"
	"fmt"
	"strings"

	"claudego/pkg/interfaces"
	"claudego/pkg/llm"
	"claudego/pkg/ui"
)

// MatchAndExecute checks if input is a slash command and executes it if found.
// Returns (matched, error):
//   - (true, nil) if a skill was matched and executed successfully
//   - (true, error) if a skill was matched but execution failed
//   - (false, nil) if input is not a slash command
func MatchAndExecute(ctx context.Context, input string, registry *Registry, llmClient *llm.Client, toolRegistry interfaces.ToolRegistry) (bool, error) {
	if !strings.HasPrefix(input, "/") {
		return false, nil
	}

	// Parse slash command: /skill-name [args]
	parts := strings.SplitN(strings.TrimPrefix(input, "/"), " ", 2)
	skillName := parts[0]
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	// Look up skill
	s, ok := registry.Get(skillName)
	if !ok {
		return false, nil // Not a skill, let caller handle as regular command
	}

	// Execute skill
	ui.Info(fmt.Sprintf("Executing skill: %s", s.Name))
	if err := Execute(ctx, s, args, llmClient, toolRegistry); err != nil {
		return true, err
	}

	return true, nil
}
