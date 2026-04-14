package skill

import (
	"context"
	"fmt"

	"claudego/pkg/interfaces"
	"claudego/pkg/llm"
	"claudego/pkg/types"
)

// Execute runs a skill with the given args and LLM client.
func Execute(ctx context.Context, s *types.Skill, args string, llmClient *llm.Client, registry interfaces.ToolRegistry) error {
	// Build system prompt: skill instructions + user args as context
	system := s.Instructions
	if args != "" {
		system = system + "\n\nUser request: " + args
	}

	// Build initial user message with the skill context
	messages := []types.Message{
		{
			Role:    "user",
			Content: "Please help with the task described above.",
		},
	}

	// Execute with LLM
	result, err := llmClient.Complete(ctx, messages, system, registry)
	if err != nil {
		return fmt.Errorf("skill execution failed: %w", err)
	}

	// Output the result
	if result.Content != "" {
		fmt.Println(result.Content)
	}

	return nil
}
