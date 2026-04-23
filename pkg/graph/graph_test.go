package graph

import (
	"context"
	"errors"
	"strings"
	"testing"

	"claudego/internal/tools"
	"claudego/pkg/interfaces"
	"claudego/pkg/types"

	"github.com/openai/openai-go/v3"
)

func TestGraphRunSequential(t *testing.T) {
	graph := New()

	if err := graph.AddNode("planner", func(ctx context.Context, state *State) (*Command, error) {
		state.Set("plan", "collect facts")
		return nil, nil
	}); err != nil {
		t.Fatalf("AddNode planner failed: %v", err)
	}
	if err := graph.AddNode("writer", func(ctx context.Context, state *State) (*Command, error) {
		plan, ok := Value[string](state, "plan")
		if !ok {
			t.Fatal("expected plan in state")
		}
		return &Command{
			Update: map[string]any{
				"answer": plan + " -> write result",
			},
		}, nil
	}); err != nil {
		t.Fatalf("AddNode writer failed: %v", err)
	}
	if err := graph.SetEntryPoint("planner"); err != nil {
		t.Fatalf("SetEntryPoint failed: %v", err)
	}
	if err := graph.AddEdge("planner", "writer"); err != nil {
		t.Fatalf("AddEdge planner->writer failed: %v", err)
	}
	if err := graph.AddEdge("writer", End); err != nil {
		t.Fatalf("AddEdge writer->end failed: %v", err)
	}

	result, err := graph.Run(context.Background(), map[string]any{"question": "how?"})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.Steps != 2 {
		t.Fatalf("Steps = %d, want 2", result.Steps)
	}
	if got := result.State["answer"]; got != "collect facts -> write result" {
		t.Fatalf("answer = %v, want final answer", got)
	}
	if len(result.Trace) != 2 {
		t.Fatalf("trace length = %d, want 2", len(result.Trace))
	}
	if result.Trace[0].Node != "planner" || result.Trace[0].Next != "writer" {
		t.Fatalf("unexpected first trace: %+v", result.Trace[0])
	}
}

func TestGraphConditionalEdges(t *testing.T) {
	graph := New()

	if err := graph.AddNode("router", func(ctx context.Context, state *State) (*Command, error) {
		return nil, nil
	}); err != nil {
		t.Fatalf("AddNode router failed: %v", err)
	}
	if err := graph.AddNode("medical", func(ctx context.Context, state *State) (*Command, error) {
		return &Command{Update: map[string]any{"expert": "medical"}}, nil
	}); err != nil {
		t.Fatalf("AddNode medical failed: %v", err)
	}
	if err := graph.AddNode("finance", func(ctx context.Context, state *State) (*Command, error) {
		return &Command{Update: map[string]any{"expert": "finance"}}, nil
	}); err != nil {
		t.Fatalf("AddNode finance failed: %v", err)
	}
	if err := graph.SetEntryPoint("router"); err != nil {
		t.Fatalf("SetEntryPoint failed: %v", err)
	}
	if err := graph.AddConditionalEdges("router", func(ctx context.Context, state *State) (string, error) {
		domain, _ := Value[string](state, "domain")
		return domain, nil
	}, map[string]string{
		"medical": "medical",
		"finance": "finance",
	}); err != nil {
		t.Fatalf("AddConditionalEdges failed: %v", err)
	}
	if err := graph.AddEdge("medical", End); err != nil {
		t.Fatalf("AddEdge medical->end failed: %v", err)
	}
	if err := graph.AddEdge("finance", End); err != nil {
		t.Fatalf("AddEdge finance->end failed: %v", err)
	}

	result, err := graph.Run(context.Background(), map[string]any{"domain": "medical"})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if got := result.State["expert"]; got != "medical" {
		t.Fatalf("expert = %v, want medical", got)
	}
	if result.Trace[0].Next != "medical" {
		t.Fatalf("router next = %q, want medical", result.Trace[0].Next)
	}
}

func TestGraphCommandGotoOverridesStaticEdge(t *testing.T) {
	graph := New()

	if err := graph.AddNode("supervisor", func(ctx context.Context, state *State) (*Command, error) {
		return &Command{Goto: "expert_b"}, nil
	}); err != nil {
		t.Fatalf("AddNode supervisor failed: %v", err)
	}
	if err := graph.AddNode("expert_a", func(ctx context.Context, state *State) (*Command, error) {
		state.Set("expert", "a")
		return nil, nil
	}); err != nil {
		t.Fatalf("AddNode expert_a failed: %v", err)
	}
	if err := graph.AddNode("expert_b", func(ctx context.Context, state *State) (*Command, error) {
		state.Set("expert", "b")
		return nil, nil
	}); err != nil {
		t.Fatalf("AddNode expert_b failed: %v", err)
	}
	if err := graph.SetEntryPoint("supervisor"); err != nil {
		t.Fatalf("SetEntryPoint failed: %v", err)
	}
	if err := graph.AddEdge("supervisor", "expert_a"); err != nil {
		t.Fatalf("AddEdge supervisor->expert_a failed: %v", err)
	}
	if err := graph.AddEdge("expert_a", End); err != nil {
		t.Fatalf("AddEdge expert_a->end failed: %v", err)
	}
	if err := graph.AddEdge("expert_b", End); err != nil {
		t.Fatalf("AddEdge expert_b->end failed: %v", err)
	}

	result, err := graph.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if got := result.State["expert"]; got != "b" {
		t.Fatalf("expert = %v, want b", got)
	}
	if result.Trace[0].Next != "expert_b" {
		t.Fatalf("first next = %q, want expert_b", result.Trace[0].Next)
	}
}

func TestGraphStopsAtMaxSteps(t *testing.T) {
	graph := New(WithMaxSteps(3))

	if err := graph.AddNode("loop", func(ctx context.Context, state *State) (*Command, error) {
		count, _ := Value[int](state, "count")
		state.Set("count", count+1)
		return nil, nil
	}); err != nil {
		t.Fatalf("AddNode loop failed: %v", err)
	}
	if err := graph.SetEntryPoint("loop"); err != nil {
		t.Fatalf("SetEntryPoint failed: %v", err)
	}
	if err := graph.AddEdge("loop", "loop"); err != nil {
		t.Fatalf("AddEdge loop->loop failed: %v", err)
	}

	result, err := graph.Run(context.Background(), nil)
	if err == nil {
		t.Fatal("expected max step error")
	}
	if !errors.Is(err, ErrMaxStepsExceeded) {
		t.Fatalf("error = %v, want ErrMaxStepsExceeded", err)
	}
	if result.Steps != 3 {
		t.Fatalf("Steps = %d, want 3", result.Steps)
	}
	if got := result.State["count"]; got != 3 {
		t.Fatalf("count = %v, want 3", got)
	}
}

func TestNewLLMNode(t *testing.T) {
	registry := tools.NewRegistry()
	if err := registry.Register(&mockTool{
		name: "search_docs",
		run: func(ctx context.Context, input []byte) (string, error) {
			return "knowledge-base-result", nil
		},
	}); err != nil {
		t.Fatalf("Register tool failed: %v", err)
	}

	client := &mockLLMClient{
		results: []*types.CompleteResult{
			{
				Content:      "I need to search",
				FinishReason: "tool_calls",
				ToolCalls: []openai.ChatCompletionMessageToolCallUnion{
					{
						ID:   "call_1",
						Type: "function",
						Function: openai.ChatCompletionMessageFunctionToolCallFunction{
							Name:      "search_docs",
							Arguments: `{"query":"golang graph agent"}`,
						},
					},
				},
			},
			{
				Content:      "Use graph runtime plus expert nodes",
				FinishReason: "stop",
			},
		},
	}

	node, err := NewLLMNode(LLMNodeConfig{
		Client:       client,
		Registry:     registry,
		SystemPrompt: "You are a domain expert supervisor.",
		BuildUserPrompt: func(state *State) string {
			task, _ := Value[string](state, "task")
			return task
		},
		ResponseKey:    "supervisor_output",
		ToolResultsKey: "tool_results",
	})
	if err != nil {
		t.Fatalf("NewLLMNode failed: %v", err)
	}

	state := NewState(map[string]any{"task": "Design a graph for a legal agent"})
	if _, err := node(context.Background(), state); err != nil {
		t.Fatalf("LLM node run failed: %v", err)
	}

	if got := state.Snapshot()["supervisor_output"]; got != "Use graph runtime plus expert nodes" {
		t.Fatalf("supervisor_output = %v, want final LLM response", got)
	}

	results, ok := state.Get("tool_results")
	if !ok {
		t.Fatal("expected tool_results in state")
	}
	typedResults, ok := results.([]types.ToolCallResult)
	if !ok {
		t.Fatalf("tool_results type = %T, want []types.ToolCallResult", results)
	}
	if len(typedResults) != 1 || typedResults[0].Content != "knowledge-base-result" {
		t.Fatalf("unexpected tool results: %+v", typedResults)
	}
	if client.calls != 2 {
		t.Fatalf("client calls = %d, want 2", client.calls)
	}
}

type mockLLMClient struct {
	results []*types.CompleteResult
	calls   int
}

func (m *mockLLMClient) Complete(ctx context.Context, messages []types.Message, system string, registry interfaces.ToolRegistry) (*types.CompleteResult, error) {
	if m.calls >= len(m.results) {
		return nil, errors.New("unexpected llm call")
	}
	result := m.results[m.calls]
	m.calls++
	return result, nil
}

func (m *mockLLMClient) Model() string {
	return "mock-model"
}

func (m *mockLLMClient) ExecuteTools(ctx context.Context, toolCalls []openai.ChatCompletionMessageToolCallUnion, registry interfaces.ToolRegistry) []types.ToolCallResult {
	results := make([]types.ToolCallResult, 0, len(toolCalls))
	for _, call := range toolCalls {
		tool, ok := registry.Get(call.Function.Name)
		if !ok {
			results = append(results, types.ToolCallResult{
				ToolCallID: call.ID,
				Name:       call.Function.Name,
				Content:    "Error: missing tool",
			})
			continue
		}
		output, err := tool.Execute(ctx, []byte(call.Function.Arguments))
		if err != nil {
			output = "Error: " + err.Error()
		}
		results = append(results, types.ToolCallResult{
			ToolCallID: call.ID,
			Name:       call.Function.Name,
			Content:    output,
		})
	}
	return results
}

type mockTool struct {
	name string
	run  func(ctx context.Context, input []byte) (string, error)
}

func (m *mockTool) Name() string { return m.name }

func (m *mockTool) Description() string { return "mock tool" }

func (m *mockTool) Execute(ctx context.Context, input []byte) (string, error) {
	return m.run(ctx, input)
}

func (m *mockTool) Parameters() map[string]any { return map[string]any{} }

func (m *mockTool) Metadata() types.ToolMetadata { return types.ToolMetadata{} }

func TestGraphContextCancellation(t *testing.T) {
	graph := New()

	blockingNode := func(ctx context.Context, state *State) (*Command, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	if err := graph.AddNode("blocker", blockingNode); err != nil {
		t.Fatalf("AddNode failed: %v", err)
	}
	if err := graph.SetEntryPoint("blocker"); err != nil {
		t.Fatalf("SetEntryPoint failed: %v", err)
	}
	if err := graph.AddEdge("blocker", End); err != nil {
		t.Fatalf("AddEdge failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := graph.Run(ctx, nil)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if result.Steps != 0 {
		t.Fatalf("Steps = %d, want 0", result.Steps)
	}
}

func TestGraphCycleDetection(t *testing.T) {
	graph := New()

	if err := graph.AddNode("A", func(ctx context.Context, state *State) (*Command, error) {
		return nil, nil
	}); err != nil {
		t.Fatalf("AddNode A failed: %v", err)
	}
	if err := graph.AddNode("B", func(ctx context.Context, state *State) (*Command, error) {
		return nil, nil
	}); err != nil {
		t.Fatalf("AddNode B failed: %v", err)
	}

	if err := graph.SetEntryPoint("A"); err != nil {
		t.Fatalf("SetEntryPoint failed: %v", err)
	}
	if err := graph.AddEdge("A", "B"); err != nil {
		t.Fatalf("AddEdge A->B failed: %v", err)
	}
	if err := graph.AddEdge("B", "A"); err != nil {
		t.Fatalf("AddEdge B->A failed: %v", err)
	}

	// Normal validation should pass
	if err := graph.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// Cycle detection should fail
	if err := graph.ValidateWithOptions(true); err == nil {
		t.Fatal("expected cycle detection error")
	}
}

func TestGraphHookPanicRecovery(t *testing.T) {
	var beforeCalled, afterCalled bool

	graph := New(WithHooks(Hooks{
		BeforeNode: func(ctx context.Context, node string, state map[string]any) {
			beforeCalled = true
			panic("before hook panic")
		},
		AfterNode: func(ctx context.Context, trace StepTrace, state map[string]any) {
			afterCalled = true
			panic("after hook panic")
		},
	}))

	if err := graph.AddNode("test", func(ctx context.Context, state *State) (*Command, error) {
		return nil, nil
	}); err != nil {
		t.Fatalf("AddNode failed: %v", err)
	}
	if err := graph.SetEntryPoint("test"); err != nil {
		t.Fatalf("SetEntryPoint failed: %v", err)
	}
	if err := graph.AddEdge("test", End); err != nil {
		t.Fatalf("AddEdge failed: %v", err)
	}

	// Graph should complete despite hook panics
	result, err := graph.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.Steps != 1 {
		t.Fatalf("Steps = %d, want 1", result.Steps)
	}
	if !beforeCalled || !afterCalled {
		t.Fatal("hooks were not called")
	}
}

func TestStateConcurrency(t *testing.T) {
	state := NewState(map[string]any{"counter": 0})

	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				count, _ := Value[int](state, "counter")
				state.Set("counter", count+1)
				_ = state.Snapshot()
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	// Should not panic with concurrent access
	snapshot := state.Snapshot()
	if _, ok := snapshot["counter"]; !ok {
		t.Fatal("counter not found in snapshot")
	}
}

func TestStateDeepCopy(t *testing.T) {
	original := []string{"a", "b", "c"}
	state := NewState(map[string]any{"items": original})

	snapshot := state.Snapshot()
	items := snapshot["items"].([]string)
	items[0] = "MODIFIED"

	// Original state should not be affected
	stateItems, _ := Value[[]string](state, "items")
	if stateItems[0] != "a" {
		t.Fatalf("state was modified: %v", stateItems)
	}
}

func TestLLMNodeMultiRoundToolCalls(t *testing.T) {
	registry := tools.NewRegistry()
	if err := registry.Register(&mockTool{
		name: "step1",
		run: func(ctx context.Context, input []byte) (string, error) {
			return "step1_result", nil
		},
	}); err != nil {
		t.Fatalf("Register step1 failed: %v", err)
	}
	if err := registry.Register(&mockTool{
		name: "step2",
		run: func(ctx context.Context, input []byte) (string, error) {
			return "step2_result", nil
		},
	}); err != nil {
		t.Fatalf("Register step2 failed: %v", err)
	}

	client := &mockLLMClient{
		results: []*types.CompleteResult{
			{
				Content:      "Calling step1",
				FinishReason: "tool_calls",
				ToolCalls: []openai.ChatCompletionMessageToolCallUnion{
					{
						ID:   "call_1",
						Type: "function",
						Function: openai.ChatCompletionMessageFunctionToolCallFunction{
							Name:      "step1",
							Arguments: `{}`,
						},
					},
				},
			},
			{
				Content:      "Calling step2",
				FinishReason: "tool_calls",
				ToolCalls: []openai.ChatCompletionMessageToolCallUnion{
					{
						ID:   "call_2",
						Type: "function",
						Function: openai.ChatCompletionMessageFunctionToolCallFunction{
							Name:      "step2",
							Arguments: `{}`,
						},
					},
				},
			},
			{
				Content:      "All done",
				FinishReason: "stop",
			},
		},
	}

	node, err := NewLLMNode(LLMNodeConfig{
		Client:        client,
		Registry:      registry,
		SystemPrompt:  "Test",
		MaxToolRounds: 5,
		BuildUserPrompt: func(state *State) string {
			return "test"
		},
	})
	if err != nil {
		t.Fatalf("NewLLMNode failed: %v", err)
	}

	state := NewState(nil)
	if _, err := node(context.Background(), state); err != nil {
		t.Fatalf("node execution failed: %v", err)
	}

	if client.calls != 3 {
		t.Fatalf("client calls = %d, want 3", client.calls)
	}

	response, _ := Value[string](state, "last_response")
	if response != "All done" {
		t.Fatalf("response = %q, want 'All done'", response)
	}
}

func TestLLMNodeFailOnToolError(t *testing.T) {
	registry := tools.NewRegistry()
	if err := registry.Register(&mockTool{
		name: "failing_tool",
		run: func(ctx context.Context, input []byte) (string, error) {
			return "", errors.New("tool failed")
		},
	}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	client := &mockLLMClient{
		results: []*types.CompleteResult{
			{
				Content:      "Calling tool",
				FinishReason: "tool_calls",
				ToolCalls: []openai.ChatCompletionMessageToolCallUnion{
					{
						ID:   "call_1",
						Type: "function",
						Function: openai.ChatCompletionMessageFunctionToolCallFunction{
							Name:      "failing_tool",
							Arguments: `{}`,
						},
					},
				},
			},
			// Second result won't be reached because FailOnToolError will stop execution
			{
				Content:      "Should not reach here",
				FinishReason: "stop",
			},
		},
	}

	node, err := NewLLMNode(LLMNodeConfig{
		Client:          client,
		Registry:        registry,
		SystemPrompt:    "Test",
		FailOnToolError: true,
		BuildUserPrompt: func(state *State) string {
			return "test"
		},
	})
	if err != nil {
		t.Fatalf("NewLLMNode failed: %v", err)
	}

	state := NewState(nil)
	_, err = node(context.Background(), state)
	if err == nil {
		t.Fatal("expected tool error to fail node")
	}
	if !strings.Contains(err.Error(), "tool execution failed") {
		t.Fatalf("error = %v, want tool execution error", err)
	}

	// Should only call LLM once before failing
	if client.calls != 1 {
		t.Fatalf("client calls = %d, want 1", client.calls)
	}
}
