package loop

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"claudego/internal/commands"
	"claudego/internal/config"
	"claudego/internal/tools"
	"claudego/pkg/compaction"
	"claudego/pkg/interfaces"
	"claudego/pkg/llm"
	"claudego/pkg/logger"
	"claudego/pkg/types"
)

type Agent struct {
	cfg       	*config.Config
	logger    	*logger.Logger
	registry  	interfaces.ToolRegistry
	llmClient 	*llm.Client

	compactor     	*compaction.Compactor
	sessionID     	string
	sessionTokens 	int

	todo 			*tools.TodoManager
}

func New(cfg *config.Config, l *logger.Logger, r interfaces.ToolRegistry) (*Agent, error) {
	if cfg.CompactionConfig == nil {
		cfg.CompactionConfig = types.DefaultCompactionConfig()
	}

	llmClient := llm.NewClient(cfg)
	sessionID := fmt.Sprintf("%d-%s", time.Now().Unix(), randomString(8))
	compactor, err := compaction.NewCompactor(cfg.CompactionConfig, llmClient, l)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize compactor: %w", err)
	}

	todo := tools.NewTodoManager()
	err = r.Register(todo)
	if err != nil {
		return nil, fmt.Errorf("failed to register tool: %w", err)
	}


	return &Agent{
		cfg:           cfg,
		logger:        l,
		registry:      r,
		llmClient:     llmClient,
		compactor:     compactor,
		sessionID:     sessionID,
		sessionTokens: 0,
		todo:          todo,
	}, nil
}

// LLMClient returns the LLM client for skill execution.
func (a *Agent) LLMClient() *llm.Client {
	return a.llmClient
}

func (a *Agent) Run(ctx context.Context, messages []types.Message) error {
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	systemPrompt := fmt.Sprintf("You are a coding agent at %s. Use bash to solve tasks.", pwd)

	for {
		if len(messages) > 0 {
			lastMsg := messages[len(messages)-1]
			if userMsg, ok := lastMsg.Content.(string); ok && strings.HasPrefix(userMsg, "/compact") {
				switch strings.TrimSpace(userMsg) {
				case "/compact":
					compactedMessages, compactErr := commands.HandleCompactCommand(ctx, a.compactor, messages, a.sessionID, a.logger)
					if compactErr != nil {
						a.logger.Error("failed to handle compact command: %v", compactErr)
						return compactErr
					}
					messages = compactedMessages
					return nil
				case "/compact-status":
					if statusErr := commands.HandleCompactStatusCommand(a.compactor, messages, a.sessionID, a.sessionTokens, a.logger); statusErr != nil {
						a.logger.Error("failed to handle compact status command: %v", statusErr)
						return statusErr
					}
					return nil
				}
			}
		}

		shouldCompact, tokenCount, countErr := a.compactor.ShouldAutoCompact(messages)
		if countErr != nil {
			a.logger.Warning("token counting failed: %v", countErr)
		}
		if shouldCompact {
			a.logger.Info("auto compacting context tokens=%d", tokenCount)
			compactedMessages, compactErr := a.compactor.FullCompact(ctx, messages, a.sessionID)
			if compactErr != nil {
				a.logger.Warning("L3 compression failed, fallback to truncation: %v", compactErr)
				compactedMessages = a.compactor.FallbackTruncate(messages)
			}
			messages = compactedMessages
		}

		result, err := a.llmClient.Complete(ctx, messages, systemPrompt, a.registry)
		if err != nil {
			return fmt.Errorf("LLM call failed: %w", err)
		}
		if result.Usage != nil {
			a.sessionTokens += result.Usage.TotalTokens
			if a.cfg.CompactionConfig.ShowTokenUsage {
				a.logger.Info(
					"token usage prompt=%d completion=%d total=%d session=%d",
					result.Usage.PromptTokens,
					result.Usage.CompletionTokens,
					result.Usage.TotalTokens,
					a.sessionTokens,
				)
			}
		}

		a.logger.Info("LLM response: %s", result.Content)
		a.logger.Info("Stop reason: %s", result.FinishReason)
		for _, tc := range result.ToolCalls {
			a.logger.Info("Tool call - ID: %s, Name: %s, Arguments: %s", tc.ID, tc.Function.Name, tc.Function.Arguments)
		}

		// Persist ToolCalls in the assistant message so BuildMessages can
		// reconstruct a well-formed history (API requires tool_calls before tool results).
		messages = append(messages, types.Message{
			Role:      "assistant",
			Content:   result.Content,
			ToolCalls: result.ToolCalls,
		})

		if result.FinishReason == "stop" {
			break
		}

		used_todo := false
		for _, tc := range result.ToolCalls {
			if tc.Function.Name == "todo_manager" {
				used_todo = true
				break
			}
		}

		if len(result.ToolCalls) > 0 {
			results := a.llmClient.ExecuteTools(ctx, result.ToolCalls, a.registry)
			results, err = a.compactor.ProcessToolResults(a.sessionID, results)
			if err != nil {
				a.logger.Warning("L1 compression failed: %v", err)
			}

			a.logger.Info("Tool execution results: %+v", results)

			// Pass []ToolCallResult directly so BuildMessages emits proper
			// ToolMessage entries instead of a freeform user string.
			messages = append(messages, types.Message{Role: "user", Content: results})
			if used_todo == false {
				reminder := a.todo.NoteRoundWithoutUpdate()
				if reminder != "" {
					results = append(results, types.ToolCallResult{
						Content: reminder,
					})
				}
			}
			currentRound := countAssistantMessages(messages)
			if err := a.compactor.CompactOldMessages(messages, currentRound); err != nil {
				a.logger.Warning("L2 compression failed: %v", err)
			}
			continue
		}
		break
	}
	return nil
}

func countAssistantMessages(messages []types.Message) int {
	count := 0
	for _, msg := range messages {
		if msg.Role == "assistant" {
			count++
		}
	}
	return count
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
