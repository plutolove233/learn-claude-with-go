package llm

import (
	"context"
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"claudego/internal/config"
	"claudego/pkg/interfaces"
	"claudego/pkg/types"
)

// TestNewClient tests client creation with configuration
func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		APIKey:  "test-api-key",
		BaseURL: "https://api.test.com",
		Model:   "claude-3-5-sonnet-20241022",
	}

	client := NewClient(cfg)

	require.NotNil(t, client)
	assert.Equal(t, "claude-3-5-sonnet-20241022", client.model)
}

// TestClient_Model tests the Model getter
func TestClient_Model(t *testing.T) {
	cfg := &config.Config{
		APIKey:  "test-key",
		BaseURL: "https://api.test.com",
		Model:   "claude-3-opus-20240229",
	}

	client := NewClient(cfg)
	assert.Equal(t, "claude-3-opus-20240229", client.Model())
}

// TestBuildMessages_SystemMessage tests system message is always first
func TestBuildMessages_SystemMessage(t *testing.T) {
	client := &Client{model: "test-model"}
	messages := []types.Message{}
	system := "You are a helpful assistant"

	result := client.buildMessages(messages, system)

	require.Len(t, result, 1)
	// System message should be first
	assert.NotNil(t, result[0])
}

// TestBuildMessages_UserMessage tests user message conversion
func TestBuildMessages_UserMessage(t *testing.T) {
	client := &Client{model: "test-model"}
	messages := []types.Message{
		{
			Role:    "user",
			Content: "Hello, how are you?",
		},
	}
	system := "You are a helpful assistant"

	result := client.buildMessages(messages, system)

	// Should have system + user message
	require.Len(t, result, 2)
}

// TestBuildMessages_AssistantMessage tests assistant message conversion
func TestBuildMessages_AssistantMessage(t *testing.T) {
	client := &Client{model: "test-model"}
	messages := []types.Message{
		{
			Role:    "assistant",
			Content: "I'm doing well, thank you!",
		},
	}
	system := "You are a helpful assistant"

	result := client.buildMessages(messages, system)

	// Should have system + assistant message
	require.Len(t, result, 2)
}

// TestBuildMessages_AssistantWithToolCalls tests assistant message with tool calls
func TestBuildMessages_AssistantWithToolCalls(t *testing.T) {
	client := &Client{model: "test-model"}
	messages := []types.Message{
		{
			Role:    "assistant",
			Content: "Let me check that for you.",
			ToolCalls: []openai.ChatCompletionMessageToolCallUnion{
				{
					ID:   "call_123",
					Type: "function",
					Function: openai.ChatCompletionMessageFunctionToolCallFunction{
						Name:      "get_weather",
						Arguments: `{"location":"San Francisco"}`,
					},
				},
			},
		},
	}
	system := "You are a helpful assistant"

	result := client.buildMessages(messages, system)

	// Should have system + assistant message with tool calls
	require.Len(t, result, 2)
}

// TestBuildMessages_ToolResults tests tool result messages
func TestBuildMessages_ToolResults(t *testing.T) {
	client := &Client{model: "test-model"}
	messages := []types.Message{
		{
			Role: "user",
			Content: []types.ToolCallResult{
				{
					ToolCallID: "call_123",
					Name:       "get_weather",
					Content:    `{"temperature": 72, "condition": "sunny"}`,
				},
			},
		},
	}
	system := "You are a helpful assistant"

	result := client.buildMessages(messages, system)

	// Should have system + tool result message
	require.Len(t, result, 2)
}

// TestBuildMessages_MultipleMessages tests multiple message types
func TestBuildMessages_MultipleMessages(t *testing.T) {
	client := &Client{model: "test-model"}
	messages := []types.Message{
		{
			Role:    "user",
			Content: "What's the weather?",
		},
		{
			Role:    "assistant",
			Content: "Let me check.",
			ToolCalls: []openai.ChatCompletionMessageToolCallUnion{
				{
					ID:   "call_123",
					Type: "function",
					Function: openai.ChatCompletionMessageFunctionToolCallFunction{
						Name:      "get_weather",
						Arguments: `{"location":"SF"}`,
					},
				},
			},
		},
		{
			Role: "user",
			Content: []types.ToolCallResult{
				{
					ToolCallID: "call_123",
					Name:       "get_weather",
					Content:    `{"temp": 72}`,
				},
			},
		},
		{
			Role:    "assistant",
			Content: "It's 72 degrees and sunny!",
		},
	}
	system := "You are a helpful assistant"

	result := client.buildMessages(messages, system)

	// Should have system + 4 messages (user, assistant with tool, tool result, assistant)
	require.Len(t, result, 5)
}

// TestBuildMessages_EmptyContent tests handling of empty content
func TestBuildMessages_EmptyContent(t *testing.T) {
	client := &Client{model: "test-model"}
	messages := []types.Message{
		{
			Role:    "user",
			Content: "",
		},
	}
	system := "You are a helpful assistant"

	result := client.buildMessages(messages, system)

	// Should still process the message
	require.Len(t, result, 2)
}

// TestToToolCallParams tests conversion of tool calls to parameters
func TestToToolCallParams(t *testing.T) {
	toolCalls := []openai.ChatCompletionMessageToolCallUnion{
		{
			ID:   "call_123",
			Type: "function",
			Function: openai.ChatCompletionMessageFunctionToolCallFunction{
				Name:      "get_weather",
				Arguments: `{"location":"SF"}`,
			},
		},
		{
			ID:   "call_456",
			Type: "function",
			Function: openai.ChatCompletionMessageFunctionToolCallFunction{
				Name:      "get_time",
				Arguments: `{"timezone":"PST"}`,
			},
		},
	}

	result := toToolCallParams(toolCalls)

	require.Len(t, result, 2)
	assert.NotNil(t, result[0].OfFunction)
	assert.Equal(t, "call_123", result[0].OfFunction.ID)
	assert.Equal(t, "get_weather", result[0].OfFunction.Function.Name)
	assert.NotNil(t, result[1].OfFunction)
	assert.Equal(t, "call_456", result[1].OfFunction.ID)
	assert.Equal(t, "get_time", result[1].OfFunction.Function.Name)
}

// TestToToolCallParams_Empty tests empty tool calls
func TestToToolCallParams_Empty(t *testing.T) {
	toolCalls := []openai.ChatCompletionMessageToolCallUnion{}

	result := toToolCallParams(toolCalls)

	require.Len(t, result, 0)
}

// MockTool is a mock implementation of the Tool interface for testing
type MockTool struct {
	name        string
	description string
	parameters  map[string]any
	executeFunc func(ctx context.Context, input []byte) (string, error)
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return m.description
}

func (m *MockTool) Execute(ctx context.Context, input []byte) (string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	return "mock output", nil
}

func (m *MockTool) Parameters() map[string]any {
	return m.parameters
}

func (m *MockTool) Metadata() types.ToolMetadata {
	return types.ToolMetadata{}
}

// MockToolRegistry is a mock implementation of ToolRegistry for testing
type MockToolRegistry struct {
	tools map[string]*MockTool
}

func NewMockToolRegistry() *MockToolRegistry {
	return &MockToolRegistry{
		tools: make(map[string]*MockTool),
	}
}

func (m *MockToolRegistry) Get(name string) (interfaces.Tool, bool) {
	tool, ok := m.tools[name]
	return tool, ok
}

func (m *MockToolRegistry) EnabledTools() []interfaces.Tool {
	var tools []interfaces.Tool
	for _, t := range m.tools {
		tools = append(tools, t)
	}
	return tools
}

func (m *MockToolRegistry) Register(tool interfaces.Tool) error {
	m.tools[tool.Name()] = tool.(*MockTool)
	return nil
}

func (m *MockToolRegistry) RegisterMock(name string, tool *MockTool) {
	m.tools[name] = tool
}

func (m *MockToolRegistry) Unregister(name string) {
	delete(m.tools, name)
}

func (m *MockToolRegistry) Filter(allowed []string) interfaces.ToolRegistry {
	return m
}

func (m *MockToolRegistry) List() []interfaces.Tool {
	var tools []interfaces.Tool
	for _, t := range m.tools {
		tools = append(tools, t)
	}
	return tools
}

// TestBuildToolDefs tests building tool definitions
func TestBuildToolDefs_EmptyRegistry(t *testing.T) {
	client := &Client{model: "test-model"}
	registry := NewMockToolRegistry()

	result := client.buildToolDefs(registry)

	assert.Nil(t, result)
}

// TestBuildToolDefs_WithTools tests building tool definitions with tools
func TestBuildToolDefs_WithTools(t *testing.T) {
	client := &Client{model: "test-model"}
	registry := NewMockToolRegistry()

	registry.RegisterMock("get_weather", &MockTool{
		name:        "get_weather",
		description: "Get weather information",
		parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]any{
					"type":        "string",
					"description": "City name",
				},
			},
		},
	})

	result := client.buildToolDefs(registry)

	require.Len(t, result, 1)
}

// TestExecuteTools_ToolNotFound tests execution when tool is not found
func TestExecuteTools_ToolNotFound(t *testing.T) {
	client := &Client{model: "test-model"}
	registry := NewMockToolRegistry()

	toolCalls := []openai.ChatCompletionMessageToolCallUnion{
		{
			ID:   "call_123",
			Type: "function",
			Function: openai.ChatCompletionMessageFunctionToolCallFunction{
				Name:      "nonexistent_tool",
				Arguments: `{}`,
			},
		},
	}

	results := client.ExecuteTools(context.Background(), toolCalls, registry)

	require.Len(t, results, 1)
	assert.Contains(t, results[0].Content, "Error: tool \"nonexistent_tool\" not found")
	assert.Equal(t, "call_123", results[0].ToolCallID)
	assert.Equal(t, "nonexistent_tool", results[0].Name)
}

// TestExecuteTools_Success tests successful tool execution
func TestExecuteTools_Success(t *testing.T) {
	client := &Client{model: "test-model"}
	registry := NewMockToolRegistry()

	registry.RegisterMock("get_weather", &MockTool{
		name:        "get_weather",
		description: "Get weather",
		executeFunc: func(ctx context.Context, input []byte) (string, error) {
			return `{"temperature": 72, "condition": "sunny"}`, nil
		},
	})

	toolCalls := []openai.ChatCompletionMessageToolCallUnion{
		{
			ID:   "call_123",
			Type: "function",
			Function: openai.ChatCompletionMessageFunctionToolCallFunction{
				Name:      "get_weather",
				Arguments: `{"location":"SF"}`,
			},
		},
	}

	results := client.ExecuteTools(context.Background(), toolCalls, registry)

	require.Len(t, results, 1)
	assert.Equal(t, `{"temperature": 72, "condition": "sunny"}`, results[0].Content)
	assert.Equal(t, "call_123", results[0].ToolCallID)
	assert.Equal(t, "get_weather", results[0].Name)
}

// TestExecuteTools_ExecutionError tests tool execution error handling
func TestExecuteTools_ExecutionError(t *testing.T) {
	client := &Client{model: "test-model"}
	registry := NewMockToolRegistry()

	registry.RegisterMock("failing_tool", &MockTool{
		name:        "failing_tool",
		description: "A tool that fails",
		executeFunc: func(ctx context.Context, input []byte) (string, error) {
			return "", assert.AnError
		},
	})

	toolCalls := []openai.ChatCompletionMessageToolCallUnion{
		{
			ID:   "call_123",
			Type: "function",
			Function: openai.ChatCompletionMessageFunctionToolCallFunction{
				Name:      "failing_tool",
				Arguments: `{}`,
			},
		},
	}

	results := client.ExecuteTools(context.Background(), toolCalls, registry)

	require.Len(t, results, 1)
	assert.Contains(t, results[0].Content, "Error:")
	assert.Equal(t, "call_123", results[0].ToolCallID)
	assert.Equal(t, "failing_tool", results[0].Name)
}

// TestExecuteTools_EmptyToolName tests handling of empty tool name
func TestExecuteTools_EmptyToolName(t *testing.T) {
	client := &Client{model: "test-model"}
	registry := NewMockToolRegistry()

	toolCalls := []openai.ChatCompletionMessageToolCallUnion{
		{
			ID:   "call_123",
			Type: "function",
			Function: openai.ChatCompletionMessageFunctionToolCallFunction{
				Name:      "",
				Arguments: `{}`,
			},
		},
	}

	results := client.ExecuteTools(context.Background(), toolCalls, registry)

	// Empty tool name should be skipped
	require.Len(t, results, 0)
}

// TestExecuteTools_MultipleTools tests execution of multiple tools
func TestExecuteTools_MultipleTools(t *testing.T) {
	client := &Client{model: "test-model"}
	registry := NewMockToolRegistry()

	registry.RegisterMock("tool1", &MockTool{
		name: "tool1",
		executeFunc: func(ctx context.Context, input []byte) (string, error) {
			return "output1", nil
		},
	})

	registry.RegisterMock("tool2", &MockTool{
		name: "tool2",
		executeFunc: func(ctx context.Context, input []byte) (string, error) {
			return "output2", nil
		},
	})

	toolCalls := []openai.ChatCompletionMessageToolCallUnion{
		{
			ID:   "call_1",
			Type: "function",
			Function: openai.ChatCompletionMessageFunctionToolCallFunction{
				Name:      "tool1",
				Arguments: `{}`,
			},
		},
		{
			ID:   "call_2",
			Type: "function",
			Function: openai.ChatCompletionMessageFunctionToolCallFunction{
				Name:      "tool2",
				Arguments: `{}`,
			},
		},
	}

	results := client.ExecuteTools(context.Background(), toolCalls, registry)

	require.Len(t, results, 2)
	assert.Equal(t, "output1", results[0].Content)
	assert.Equal(t, "call_1", results[0].ToolCallID)
	assert.Equal(t, "output2", results[1].Content)
	assert.Equal(t, "call_2", results[1].ToolCallID)
}
