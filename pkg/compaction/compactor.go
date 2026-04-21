package compaction

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"claudego/pkg/logger"
	"claudego/pkg/types"
)

// Compactor coordinates L1/L2/L3 compaction strategies.
type Compactor struct {
	storage      *Storage
	tokenCounter *TokenCounter
	summarizer   *Summarizer
	config       *types.CompactionConfig
	logger       *logger.Logger
	mu           sync.Mutex
}

func NewCompactor(config *types.CompactionConfig, llmClient LLMCompleter, log *logger.Logger) (*Compactor, error) {
	if config == nil {
		config = types.DefaultCompactionConfig()
	}
	storage, err := NewStorage(config.SessionStoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	return &Compactor{
		storage:      storage,
		tokenCounter: NewTokenCounter(),
		summarizer:   NewSummarizer(llmClient, config),
		config:       config,
		logger:       log,
	}, nil
}

// ProcessToolResults applies L1 compaction by storing large tool outputs on disk.
func (c *Compactor) ProcessToolResults(sessionID string, results []types.ToolCallResult) ([]types.ToolCallResult, error) {
	if c == nil || c.config == nil || !c.config.Enabled {
		return results, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	processed := make([]types.ToolCallResult, 0, len(results))
	for _, result := range results {
		contentSize := len([]byte(result.Content))
		if contentSize <= c.config.LargeResultThreshold {
			processed = append(processed, result)
			continue
		}

		storedResult := types.StoredToolResult{
			ToolCallID: result.ToolCallID,
			ToolName:   result.Name,
			Arguments:  "",
			Content:    result.Content,
			Timestamp:  time.Now(),
		}
		filePath, err := c.storage.SaveToolResult(sessionID, storedResult)
		if err != nil {
			if c.logger != nil {
				c.logger.Warning("failed to save tool result to disk: %v", err)
			}
			processed = append(processed, result)
			continue
		}

		runes := []rune(result.Content)
		summary := result.Content
		if len(runes) > 200 {
			summary = string(runes[:200]) + "..."
		}

		reference := types.ToolResultReference{
			Type:     "tool_result_reference",
			FilePath: filePath,
			Summary:  summary,
		}
		refJSON, _ := json.Marshal(reference)
		processed = append(processed, types.ToolCallResult{
			ToolCallID: result.ToolCallID,
			Name:       result.Name,
			Content:    string(refJSON),
		})

		if c.logger != nil {
			c.logger.Info("L1 compression saved tool=%s size=%d path=%s", result.Name, contentSize, filePath)
		}
	}

	return processed, nil
}

// CompactOldMessages applies L2 compaction to old tool results.
func (c *Compactor) CompactOldMessages(messages []types.Message, currentRound int) error {
	if c == nil || c.config == nil || !c.config.Enabled || currentRound <= c.config.OldMessageThreshold {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	targetRound := currentRound - c.config.OldMessageThreshold
	assistantCount := 0
	targetIndex := -1
	for i, msg := range messages {
		if msg.Role == "assistant" {
			assistantCount++
			if assistantCount == targetRound {
				targetIndex = i
				break
			}
		}
	}
	if targetIndex == -1 {
		return nil
	}

	for i := 0; i < targetIndex; i++ {
		msg := &messages[i]
		if msg.Role != "user" {
			continue
		}
		results, ok := msg.Content.([]types.ToolCallResult)
		if !ok {
			continue
		}

		for j := range results {
			result := &results[j]
			if strings.HasPrefix(strings.TrimSpace(result.Content), "{") && strings.Contains(result.Content, "tool_result_reference") {
				continue
			}
			runes := []rune(result.Content)
			if len(runes) > 200 {
				result.Content = string(runes[:200]) + "\\n[...截断，完整内容见磁盘文件...]"
			}
		}
		msg.Content = results
	}

	return nil
}

func (c *Compactor) ShouldAutoCompact(messages []types.Message) (bool, int, error) {
	if c == nil || c.config == nil || !c.config.Enabled {
		return false, 0, nil
	}
	totalTokens := c.tokenCounter.CountMessages(messages)
	return totalTokens > c.config.AutoCompactTokens, totalTokens, nil
}

// FullCompact applies L3 compaction by replacing history with a summary plus latest user turn.
func (c *Compactor) FullCompact(ctx context.Context, messages []types.Message, sessionID string) ([]types.Message, error) {
	if c == nil || c.config == nil || !c.config.Enabled {
		return messages, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	tokensBefore := c.tokenCounter.CountMessages(messages)
	summary, err := c.summarizer.GenerateSummary(ctx, messages, sessionID)
	if err != nil {
		if c.logger != nil {
			c.logger.Warning("failed to generate summary, fallback to truncation: %v", err)
		}
		return c.FallbackTruncate(messages), nil
	}

	summary = c.summarizer.TruncateSummary(summary, c.config.SummaryMaxTokens)

	var lastUserMessage *types.Message
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			copyMsg := messages[i]
			lastUserMessage = &copyMsg
			break
		}
	}

	compactedMessages := []types.Message{{Role: "user", Content: "对话摘要：\n" + summary}}
	if lastUserMessage != nil {
		compactedMessages = append(compactedMessages, *lastUserMessage)
	}

	tokensAfter := c.tokenCounter.CountMessages(compactedMessages)
	history := types.CompactionHistory{
		SessionID:     sessionID,
		Timestamp:     time.Now(),
		TokensBefore:  tokensBefore,
		TokensAfter:   tokensAfter,
		MessagesCount: len(messages),
		SummaryLength: len([]rune(summary)),
		FilesStored:   0,
	}
	if err := c.storage.SaveCompactionHistory(sessionID, history); err != nil && c.logger != nil {
		c.logger.Warning("failed to save compaction history: %v", err)
	}

	if c.logger != nil {
		c.logger.Info("L3 compression completed before=%d after=%d", tokensBefore, tokensAfter)
	}

	return compactedMessages, nil
}

func (c *Compactor) FallbackTruncate(messages []types.Message) []types.Message {
	if len(messages) <= 5 {
		return messages
	}
	return messages[len(messages)-5:]
}

func (c *Compactor) TokenCounter() *TokenCounter {
	return c.tokenCounter
}

func (c *Compactor) Config() *types.CompactionConfig {
	return c.config
}
