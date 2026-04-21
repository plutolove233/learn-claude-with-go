package compaction

import (
	"encoding/json"
	"unicode/utf8"

	"claudego/pkg/types"
)

// TokenCounter estimates token usage from conversation messages.
type TokenCounter struct {
	lastCount int
}

func NewTokenCounter() *TokenCounter {
	return &TokenCounter{}
}

// CountMessage estimates token count with a simple heuristic: 1 token ~= 4 chars.
func (tc *TokenCounter) CountMessage(msg types.Message) int {
	var content string

	switch v := msg.Content.(type) {
	case string:
		content = v
	case []types.ToolCallResult:
		data, err := json.Marshal(v)
		if err != nil {
			for _, result := range v {
				content += result.ToolCallID + result.Name + result.Content
			}
		} else {
			content = string(data)
		}
	default:
		data, err := json.Marshal(v)
		if err == nil {
			content = string(data)
		}
	}

	charCount := utf8.RuneCountInString(content)
	tokens := (charCount + 3) / 4
	if tokens < 1 {
		tokens = 1
	}
	return tokens
}

func (tc *TokenCounter) CountMessages(messages []types.Message) int {
	total := 0
	for _, msg := range messages {
		total += tc.CountMessage(msg)
	}
	tc.lastCount = total
	return total
}

func (tc *TokenCounter) LastCount() int {
	return tc.lastCount
}

// EstimateCompression estimates post-compaction tokens with a summary and last user turn.
func (tc *TokenCounter) EstimateCompression(messages []types.Message, summaryTokens int) int {
	lastUserTokens := 0
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastUserTokens = tc.CountMessage(messages[i])
			break
		}
	}
	return summaryTokens + lastUserTokens
}
