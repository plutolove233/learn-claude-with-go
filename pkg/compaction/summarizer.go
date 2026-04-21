package compaction

import (
	"context"
	"fmt"
	"strings"

	"claudego/pkg/interfaces"
	"claudego/pkg/types"
)

// LLMCompleter abstracts the completion call used for summary generation.
type LLMCompleter interface {
	Complete(ctx context.Context, messages []types.Message, system string, registry interfaces.ToolRegistry) (*types.CompleteResult, error)
}

// Summarizer uses the LLM client to generate compact conversation summaries.
type Summarizer struct {
	llmClient LLMCompleter
	config    *types.CompactionConfig
}

func NewSummarizer(llmClient LLMCompleter, config *types.CompactionConfig) *Summarizer {
	return &Summarizer{llmClient: llmClient, config: config}
}

func (s *Summarizer) GenerateSummary(ctx context.Context, messages []types.Message, sessionID string) (string, error) {
	if s.llmClient == nil {
		return "", fmt.Errorf("llm client is nil")
	}
	prompt := s.buildSummaryPrompt(messages, sessionID)
	summaryMessages := []types.Message{{Role: "user", Content: prompt}}

	result, err := s.llmClient.Complete(
		ctx,
		summaryMessages,
		"You are a helpful assistant that creates concise summaries of coding conversations.",
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}
	return result.Content, nil
}

func (s *Summarizer) buildSummaryPrompt(messages []types.Message, sessionID string) string {
	var b strings.Builder
	b.WriteString("请为以下对话生成一份简洁摘要，控制在 500-1000 token 内。\\n\\n")

	for i, msg := range messages {
		if i > 0 {
			b.WriteString("\\n")
		}
		b.WriteString("[")
		b.WriteString(strings.ToUpper(msg.Role))
		b.WriteString("]\\n")

		switch v := msg.Content.(type) {
		case string:
			if len(v) > 500 {
				b.WriteString(v[:500])
				b.WriteString("\\n[...内容已截断...]\\n")
			} else {
				b.WriteString(v)
				b.WriteString("\\n")
			}
		case []types.ToolCallResult:
			for _, result := range v {
				b.WriteString(fmt.Sprintf("工具: %s\\n", result.Name))
				if len(result.Content) > 200 {
					b.WriteString(result.Content[:200])
					b.WriteString("\\n[...内容已截断...]\\n")
				} else {
					b.WriteString(result.Content)
					b.WriteString("\\n")
				}
			}
		default:
			b.WriteString("[不可序列化消息类型]\\n")
		}
	}

	b.WriteString("\\n摘要需要覆盖：\\n")
	b.WriteString("1. 用户的主要目标\\n")
	b.WriteString("2. 已完成的关键步骤\\n")
	b.WriteString("3. 当前状态与进展\\n")
	b.WriteString("4. 关键决策与发现\\n")
	b.WriteString("5. 下一步计划\\n\\n")
	b.WriteString("会话 ID: ")
	b.WriteString(sessionID)
	b.WriteString("\\n")
	return b.String()
}

func (s *Summarizer) TruncateSummary(summary string, maxTokens int) string {
	if maxTokens <= 0 {
		return ""
	}
	maxChars := maxTokens * 4
	runes := []rune(summary)
	if len(runes) <= maxChars {
		return summary
	}
	return string(runes[:maxChars]) + "\\n[...摘要已截断...]"
}
