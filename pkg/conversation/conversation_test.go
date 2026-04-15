package conversation

import (
	"sync"
	"testing"

	"claudego/pkg/types"
)

// TestNew verifies that New() creates a conversation with empty message list
func TestNew(t *testing.T) {
	conv := New()

	if conv == nil {
		t.Fatal("New() returned nil")
	}

	if conv.messages == nil {
		t.Error("messages slice should be initialized")
	}

	if len(conv.messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(conv.messages))
	}
}

// TestAddUserMessage verifies that user messages are added correctly
func TestAddUserMessage(t *testing.T) {
	conv := New()

	conv.AddUserMessage("Hello, Claude!")

	messages := conv.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	if messages[0].Role != "user" {
		t.Errorf("expected role 'user', got '%s'", messages[0].Role)
	}

	if messages[0].Content != "Hello, Claude!" {
		t.Errorf("expected content 'Hello, Claude!', got '%v'", messages[0].Content)
	}
}

// TestAddMultipleUserMessages verifies that multiple user messages are added in order
func TestAddMultipleUserMessages(t *testing.T) {
	conv := New()

	conv.AddUserMessage("First message")
	conv.AddUserMessage("Second message")
	conv.AddUserMessage("Third message")

	messages := conv.GetMessages()
	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}

	expected := []string{"First message", "Second message", "Third message"}
	for i, msg := range messages {
		if msg.Content != expected[i] {
			t.Errorf("message %d: expected '%s', got '%v'", i, expected[i], msg.Content)
		}
	}
}

// TestAddToolResults verifies that tool results are added correctly
func TestAddToolResults(t *testing.T) {
	conv := New()

	results := []types.ToolCallResult{
		{
			ToolCallID: "call_123",
			Name:       "get_weather",
			Content:    "Sunny, 72°F",
		},
	}

	conv.AddToolResults(results)

	messages := conv.GetMessages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	if messages[0].Role != "user" {
		t.Errorf("expected role 'user', got '%s'", messages[0].Role)
	}

	// Content should be the tool results slice
	toolResults, ok := messages[0].Content.([]types.ToolCallResult)
	if !ok {
		t.Fatalf("expected Content to be []types.ToolCallResult, got %T", messages[0].Content)
	}

	if len(toolResults) != 1 {
		t.Fatalf("expected 1 tool result, got %d", len(toolResults))
	}

	if toolResults[0].ToolCallID != "call_123" {
		t.Errorf("expected ToolCallID 'call_123', got '%s'", toolResults[0].ToolCallID)
	}
}

// TestCheckpoint verifies that Checkpoint returns the current message count
func TestCheckpoint(t *testing.T) {
	conv := New()

	checkpoint0 := conv.Checkpoint()
	if checkpoint0 != 0 {
		t.Errorf("expected checkpoint 0, got %d", checkpoint0)
	}

	conv.AddUserMessage("Message 1")
	checkpoint1 := conv.Checkpoint()
	if checkpoint1 != 1 {
		t.Errorf("expected checkpoint 1, got %d", checkpoint1)
	}

	conv.AddUserMessage("Message 2")
	conv.AddUserMessage("Message 3")
	checkpoint3 := conv.Checkpoint()
	if checkpoint3 != 3 {
		t.Errorf("expected checkpoint 3, got %d", checkpoint3)
	}
}

// TestRollback verifies that Rollback removes messages added after checkpoint
func TestRollback(t *testing.T) {
	conv := New()

	conv.AddUserMessage("Message 1")
	conv.AddUserMessage("Message 2")
	checkpoint := conv.Checkpoint()

	conv.AddUserMessage("Message 3")
	conv.AddUserMessage("Message 4")

	if conv.MessageCount() != 4 {
		t.Fatalf("expected 4 messages before rollback, got %d", conv.MessageCount())
	}

	conv.Rollback(checkpoint)

	if conv.MessageCount() != 2 {
		t.Errorf("expected 2 messages after rollback, got %d", conv.MessageCount())
	}

	messages := conv.GetMessages()
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	if messages[0].Content != "Message 1" || messages[1].Content != "Message 2" {
		t.Error("rollback did not preserve correct messages")
	}
}

// TestRollbackToZero verifies that rollback to 0 clears all messages
func TestRollbackToZero(t *testing.T) {
	conv := New()

	conv.AddUserMessage("Message 1")
	conv.AddUserMessage("Message 2")
	conv.AddUserMessage("Message 3")

	conv.Rollback(0)

	if conv.MessageCount() != 0 {
		t.Errorf("expected 0 messages after rollback to 0, got %d", conv.MessageCount())
	}
}

// TestRollbackNegativeCheckpoint verifies that negative checkpoint is treated as 0
func TestRollbackNegativeCheckpoint(t *testing.T) {
	conv := New()

	conv.AddUserMessage("Message 1")
	conv.AddUserMessage("Message 2")

	conv.Rollback(-5)

	if conv.MessageCount() != 0 {
		t.Errorf("expected 0 messages after rollback to negative, got %d", conv.MessageCount())
	}
}

// TestRollbackBeyondLength verifies that rollback beyond message count is safe
func TestRollbackBeyondLength(t *testing.T) {
	conv := New()

	conv.AddUserMessage("Message 1")
	conv.AddUserMessage("Message 2")

	conv.Rollback(100)

	if conv.MessageCount() != 2 {
		t.Errorf("expected 2 messages (no change), got %d", conv.MessageCount())
	}
}

// TestMultipleCheckpointsAndRollbacks verifies complex checkpoint/rollback scenarios
func TestMultipleCheckpointsAndRollbacks(t *testing.T) {
	conv := New()

	conv.AddUserMessage("Message 1")
	checkpoint1 := conv.Checkpoint()

	conv.AddUserMessage("Message 2")
	conv.AddUserMessage("Message 3")
	checkpoint2 := conv.Checkpoint()

	conv.AddUserMessage("Message 4")
	conv.AddUserMessage("Message 5")

	// Rollback to checkpoint2 (should have 3 messages)
	conv.Rollback(checkpoint2)
	if conv.MessageCount() != 3 {
		t.Errorf("expected 3 messages after first rollback, got %d", conv.MessageCount())
	}

	// Rollback to checkpoint1 (should have 1 message)
	conv.Rollback(checkpoint1)
	if conv.MessageCount() != 1 {
		t.Errorf("expected 1 message after second rollback, got %d", conv.MessageCount())
	}

	messages := conv.GetMessages()
	if len(messages) != 1 || messages[0].Content != "Message 1" {
		t.Error("final state should only contain Message 1")
	}
}

// TestGetMessages verifies that GetMessages returns a copy of messages
func TestGetMessages(t *testing.T) {
	conv := New()

	conv.AddUserMessage("Original message")

	messages1 := conv.GetMessages()
	messages2 := conv.GetMessages()

	// Verify we got copies, not the same slice
	if &messages1[0] == &messages2[0] {
		t.Error("GetMessages should return a copy, not the original slice")
	}

	// Modifying returned slice should not affect conversation
	messages1[0].Content = "Modified"

	messages3 := conv.GetMessages()
	if messages3[0].Content != "Original message" {
		t.Error("modifying returned messages should not affect conversation state")
	}
}

// TestClear verifies that Clear removes all messages
func TestClear(t *testing.T) {
	conv := New()

	conv.AddUserMessage("Message 1")
	conv.AddUserMessage("Message 2")
	conv.AddUserMessage("Message 3")

	conv.Clear()

	if conv.MessageCount() != 0 {
		t.Errorf("expected 0 messages after Clear, got %d", conv.MessageCount())
	}

	messages := conv.GetMessages()
	if len(messages) != 0 {
		t.Errorf("expected empty messages slice, got %d messages", len(messages))
	}
}

// TestMessageCount verifies that MessageCount returns correct count
func TestMessageCount(t *testing.T) {
	conv := New()

	if conv.MessageCount() != 0 {
		t.Errorf("expected 0 messages initially, got %d", conv.MessageCount())
	}

	conv.AddUserMessage("Message 1")
	if conv.MessageCount() != 1 {
		t.Errorf("expected 1 message, got %d", conv.MessageCount())
	}

	conv.AddUserMessage("Message 2")
	conv.AddUserMessage("Message 3")
	if conv.MessageCount() != 3 {
		t.Errorf("expected 3 messages, got %d", conv.MessageCount())
	}
}

// TestLastN verifies that LastN returns the last N messages
func TestLastN(t *testing.T) {
	conv := New()

	conv.AddUserMessage("Message 1")
	conv.AddUserMessage("Message 2")
	conv.AddUserMessage("Message 3")
	conv.AddUserMessage("Message 4")
	conv.AddUserMessage("Message 5")

	// Get last 2 messages
	last2 := conv.LastN(2)
	if len(last2) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(last2))
	}

	if last2[0].Content != "Message 4" || last2[1].Content != "Message 5" {
		t.Error("LastN(2) should return Message 4 and Message 5")
	}
}

// TestLastNZero verifies that LastN(0) returns all messages
func TestLastNZero(t *testing.T) {
	conv := New()

	conv.AddUserMessage("Message 1")
	conv.AddUserMessage("Message 2")
	conv.AddUserMessage("Message 3")

	lastN := conv.LastN(0)
	if len(lastN) != 3 {
		t.Errorf("expected 3 messages for LastN(0), got %d", len(lastN))
	}
}

// TestLastNNegative verifies that LastN with negative value returns all messages
func TestLastNNegative(t *testing.T) {
	conv := New()

	conv.AddUserMessage("Message 1")
	conv.AddUserMessage("Message 2")

	lastN := conv.LastN(-5)
	if len(lastN) != 2 {
		t.Errorf("expected 2 messages for LastN(-5), got %d", len(lastN))
	}
}

// TestLastNExceedsLength verifies that LastN greater than message count returns all
func TestLastNExceedsLength(t *testing.T) {
	conv := New()

	conv.AddUserMessage("Message 1")
	conv.AddUserMessage("Message 2")

	lastN := conv.LastN(100)
	if len(lastN) != 2 {
		t.Errorf("expected 2 messages for LastN(100), got %d", len(lastN))
	}
}

// TestConcurrentAccess verifies thread-safety of conversation operations
func TestConcurrentAccess(t *testing.T) {
	conv := New()
	var wg sync.WaitGroup

	// Spawn multiple goroutines adding messages concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			conv.AddUserMessage("Concurrent message")
		}(i)
	}

	wg.Wait()

	if conv.MessageCount() != 10 {
		t.Errorf("expected 10 messages after concurrent adds, got %d", conv.MessageCount())
	}
}

// TestCheckpointRollbackWorkflow simulates a realistic conversation workflow
func TestCheckpointRollbackWorkflow(t *testing.T) {
	conv := New()

	// Initial conversation
	conv.AddUserMessage("What's the weather?")
	checkpoint1 := conv.Checkpoint()

	// Simulate assistant response with tool call
	results := []types.ToolCallResult{
		{
			ToolCallID: "call_weather",
			Name:       "get_weather",
			Content:    "Sunny, 72°F",
		},
	}
	conv.AddToolResults(results)

	// Continue conversation
	conv.AddUserMessage("Thanks! What about tomorrow?")
	checkpoint2 := conv.Checkpoint()

	// Another tool call
	results2 := []types.ToolCallResult{
		{
			ToolCallID: "call_forecast",
			Name:       "get_forecast",
			Content:    "Rainy, 65°F",
		},
	}
	conv.AddToolResults(results2)

	// Verify we have 4 messages
	if conv.MessageCount() != 4 {
		t.Fatalf("expected 4 messages, got %d", conv.MessageCount())
	}

	// Rollback to checkpoint2 (undo last tool call)
	conv.Rollback(checkpoint2)
	if conv.MessageCount() != 3 {
		t.Errorf("expected 3 messages after rollback, got %d", conv.MessageCount())
	}

	// Rollback to checkpoint1 (undo entire second exchange)
	conv.Rollback(checkpoint1)
	if conv.MessageCount() != 1 {
		t.Errorf("expected 1 message after second rollback, got %d", conv.MessageCount())
	}

	messages := conv.GetMessages()
	if messages[0].Content != "What's the weather?" {
		t.Error("should only have initial user message after rollback")
	}
}
