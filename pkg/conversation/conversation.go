package conversation

import (
	"sync"

	"claudego/pkg/llm"
)

type Conversation struct {
	mu       sync.Mutex
	messages []llm.Message
}

func New() *Conversation {
	return &Conversation{
		messages: make([]llm.Message, 0),
	}
}

// Checkpoint returns the current message count.
// Call Rollback(checkpoint) later to restore to this point.
func (c *Conversation) Checkpoint() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.messages)
}

// Rollback removes all messages added after the checkpoint was taken.
// This effectively undoes an entire round-trip (user message + assistant response + tool calls).
func (c *Conversation) Rollback(checkpoint int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if checkpoint < 0 {
		checkpoint = 0
	}
	if checkpoint < len(c.messages) {
		c.messages = c.messages[:checkpoint]
	}
}

func (c *Conversation) AddUserMessage(content string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, llm.Message{
		Role:    "user",
		Content: content,
	})
}

func (c *Conversation) AddToolResults(results []llm.ToolCallResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, llm.Message{
		Role:    "user",
		Content: results,
	})
}

func (c *Conversation) GetMessages() []llm.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]llm.Message, len(c.messages))
	copy(result, c.messages)
	return result
}

func (c *Conversation) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = make([]llm.Message, 0)
}

func (c *Conversation) MessageCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.messages)
}

func (c *Conversation) LastN(n int) []llm.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	if n <= 0 || n >= len(c.messages) {
		result := make([]llm.Message, len(c.messages))
		copy(result, c.messages)
		return result
	}
	result := make([]llm.Message, n)
	copy(result, c.messages[len(c.messages)-n:])
	return result
}
