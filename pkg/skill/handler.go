package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"claudego/pkg/interfaces"
	"claudego/pkg/ui"
)

// MatchAndExecute checks if input is a slash command and loads the matching skill through the skill tool.
// Returns (matched, output, error):
//   - (true, output, nil) if a skill was matched and loaded successfully
//   - (true, "", error) if a skill was matched but loading failed
//   - (false, "", nil) if input is not a skill slash command
func MatchAndExecute(ctx context.Context, input string, registry *Registry, toolRegistry interfaces.ToolRegistry) (bool, string, error) {
	if !strings.HasPrefix(input, "/") {
		return false, "", nil
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
		return false, "", nil // Not a skill, let caller handle as regular command
	}

	tool, ok := toolRegistry.Get("load_skill")
	if !ok {
		return true, "", fmt.Errorf("load_skill tool is not registered")
	}

	ui.Info(fmt.Sprintf("Loading skill: %s", s.Name))

	payload, err := json.Marshal(struct {
		Skill   string `json:"skill"`
		Context string `json:"context"`
	}{
		Skill:   s.Name,
		Context: args,
	})
	if err != nil {
		return true, "", fmt.Errorf("failed to build skill payload: %w", err)
	}

	output, err := tool.Execute(ctx, payload)
	if err != nil {
		return true, "", err
	}

	ui.Info(fmt.Sprintf("Loaded skill: %s", s.Name))
	return true, output, nil
}
