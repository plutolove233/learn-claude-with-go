package skill

import (
	"fmt"
	"strings"

	"claudego/pkg/types"
)

// BuildContext renders a loaded skill into a context block that can be appended to the conversation.
func BuildContext(s *types.Skill, args string) string {
	if s == nil {
		return ""
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "Skill loaded: %s\n", s.Name)
	builder.WriteString("This skill is now active for the rest of the conversation.\n")

	if s.Description != "" {
		fmt.Fprintf(&builder, "Description: %s\n", s.Description)
	}

	if trimmedArgs := strings.TrimSpace(args); trimmedArgs != "" {
		builder.WriteString("\nContext:\n")
		builder.WriteString(trimmedArgs)
		builder.WriteString("\n")
	}

	if trimmedInstructions := strings.TrimSpace(s.Instructions); trimmedInstructions != "" {
		builder.WriteString("\nInstructions:\n")
		builder.WriteString(trimmedInstructions)
	}

	return strings.TrimSpace(builder.String())
}
