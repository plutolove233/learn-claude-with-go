package graph

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

const (
	// Start is the virtual graph entry node.
	Start = "__start__"
	// End is the virtual graph exit node.
	End = "__end__"
)

const defaultMaxSteps = 64

var ErrMaxStepsExceeded = errors.New("graph exceeded max steps")

// NodeFunc is a graph node that can inspect and update shared state.
type NodeFunc func(ctx context.Context, state *State) (*Command, error)

// RouterFunc chooses the next edge label after a node completes.
type RouterFunc func(ctx context.Context, state *State) (string, error)

// Command lets a node apply state updates and jump to a specific next node.
type Command struct {
	Update map[string]any
	Goto   string
}

// StepTrace records one node execution.
type StepTrace struct {
	Node        string    `json:"node"`
	Next        string    `json:"next,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Error       string    `json:"error,omitempty"`
}

// Result captures graph execution output.
type Result struct {
	State map[string]any `json:"state"`
	Trace []StepTrace    `json:"trace"`
	Steps int            `json:"steps"`
}

// Hooks expose runtime callbacks for observability.
type Hooks struct {
	BeforeNode func(ctx context.Context, node string, state map[string]any)
	AfterNode  func(ctx context.Context, trace StepTrace, state map[string]any)
}

// Option customizes graph behavior.
type Option func(*Graph)

// WithMaxSteps limits the number of node executions in one run.
func WithMaxSteps(maxSteps int) Option {
	return func(g *Graph) {
		if maxSteps > 0 {
			g.maxSteps = maxSteps
		}
	}
}

// WithHooks installs execution hooks.
func WithHooks(hooks Hooks) Option {
	return func(g *Graph) {
		g.hooks = hooks
	}
}

type conditionalEdge struct {
	router RouterFunc
	routes map[string]string
}

// Graph is a minimal LangGraph-style workflow runtime.
type Graph struct {
	nodes      map[string]NodeFunc
	edges      map[string]string
	conditions map[string]conditionalEdge
	maxSteps   int
	hooks      Hooks
}

// New creates a graph runtime with sensible defaults.
func New(options ...Option) *Graph {
	graph := &Graph{
		nodes:      make(map[string]NodeFunc),
		edges:      make(map[string]string),
		conditions: make(map[string]conditionalEdge),
		maxSteps:   defaultMaxSteps,
	}

	for _, option := range options {
		option(graph)
	}

	return graph
}

// AddNode registers a named node.
func (g *Graph) AddNode(name string, node NodeFunc) error {
	if name == "" {
		return errors.New("node name cannot be empty")
	}
	if name == Start || name == End {
		return fmt.Errorf("%q is reserved", name)
	}
	if node == nil {
		return fmt.Errorf("node %q cannot be nil", name)
	}
	if _, exists := g.nodes[name]; exists {
		return fmt.Errorf("node %q already exists", name)
	}

	g.nodes[name] = node
	return nil
}

// SetEntryPoint sets the first real node to execute.
func (g *Graph) SetEntryPoint(name string) error {
	return g.AddEdge(Start, name)
}

// AddEdge registers a deterministic edge between two nodes.
func (g *Graph) AddEdge(from, to string) error {
	if from == "" || to == "" {
		return errors.New("edge endpoints cannot be empty")
	}
	if from == End {
		return errors.New("cannot add outgoing edge from end")
	}
	if existing, ok := g.conditions[from]; ok && existing.router != nil {
		return fmt.Errorf("node %q already has conditional edges", from)
	}
	if _, exists := g.edges[from]; exists {
		return fmt.Errorf("node %q already has a direct edge", from)
	}

	g.edges[from] = to
	return nil
}

// AddConditionalEdges registers a router and label-to-node mapping for a node.
func (g *Graph) AddConditionalEdges(from string, router RouterFunc, routes map[string]string) error {
	if from == "" {
		return errors.New("conditional edge source cannot be empty")
	}
	if from == Start || from == End {
		return fmt.Errorf("conditional edges are not supported from %q", from)
	}
	if router == nil {
		return fmt.Errorf("router for node %q cannot be nil", from)
	}
	if len(routes) == 0 {
		return fmt.Errorf("node %q must define at least one conditional route", from)
	}
	if _, exists := g.edges[from]; exists {
		return fmt.Errorf("node %q already has a direct edge", from)
	}
	if _, exists := g.conditions[from]; exists {
		return fmt.Errorf("node %q already has conditional edges", from)
	}

	copied := make(map[string]string, len(routes))
	for label, target := range routes {
		if label == "" {
			return fmt.Errorf("node %q has an empty route label", from)
		}
		if target == "" {
			return fmt.Errorf("node %q route %q has an empty target", from, label)
		}
		copied[label] = target
	}

	g.conditions[from] = conditionalEdge{
		router: router,
		routes: copied,
	}
	return nil
}

// Validate checks that all referenced nodes exist and the graph has an entry point.
// If checkCycles is true, it also detects cycles in the graph.
func (g *Graph) Validate() error {
	return g.ValidateWithOptions(false)
}

// ValidateWithOptions checks graph validity with optional cycle detection.
func (g *Graph) ValidateWithOptions(checkCycles bool) error {
	startTarget, ok := g.edges[Start]
	if !ok {
		return errors.New("graph entry point is not set")
	}
	if err := g.validateTarget(startTarget); err != nil {
		return fmt.Errorf("invalid entry point: %w", err)
	}

	for from, to := range g.edges {
		if from != Start {
			if _, ok := g.nodes[from]; !ok {
				return fmt.Errorf("direct edge source %q is not a registered node", from)
			}
		}
		if err := g.validateTarget(to); err != nil {
			return fmt.Errorf("invalid direct edge %q -> %q: %w", from, to, err)
		}
	}

	for from, condition := range g.conditions {
		if _, ok := g.nodes[from]; !ok {
			return fmt.Errorf("conditional edge source %q is not a registered node", from)
		}
		for label, target := range condition.routes {
			if err := g.validateTarget(target); err != nil {
				return fmt.Errorf("invalid conditional edge %q[%s] -> %q: %w", from, label, target, err)
			}
		}
	}

	if checkCycles {
		if err := g.detectCycles(); err != nil {
			return err
		}
	}

	return nil
}

// detectCycles uses DFS to detect cycles in the graph.
func (g *Graph) detectCycles() error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(node string) error
	dfs = func(node string) error {
		if node == End {
			return nil
		}

		if recStack[node] {
			return fmt.Errorf("cycle detected involving node %q", node)
		}
		if visited[node] {
			return nil
		}

		visited[node] = true
		recStack[node] = true

		// Check direct edges
		if next, ok := g.edges[node]; ok {
			if err := dfs(next); err != nil {
				return err
			}
		}

		// Check conditional edges
		if condition, ok := g.conditions[node]; ok {
			for _, target := range condition.routes {
				if err := dfs(target); err != nil {
					return err
				}
			}
		}

		recStack[node] = false
		return nil
	}

	// Start DFS from entry point
	if entryPoint, ok := g.edges[Start]; ok {
		if err := dfs(entryPoint); err != nil {
			return err
		}
	}

	return nil
}

// Run executes the graph from entry point to end using an initial state snapshot.
func (g *Graph) Run(ctx context.Context, initial map[string]any) (*Result, error) {
	return g.RunState(ctx, NewState(initial))
}

// RunState executes the graph using a caller-provided state object.
func (g *Graph) RunState(ctx context.Context, state *State) (*Result, error) {
	if state == nil {
		state = NewState(nil)
	}
	if err := g.Validate(); err != nil {
		return &Result{State: state.Snapshot()}, err
	}

	current := g.edges[Start]
	result := &Result{
		State: state.Snapshot(),
		Trace: make([]StepTrace, 0),
	}

	for current != End {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			result.State = state.Snapshot()
			return result, fmt.Errorf("graph execution cancelled: %w", err)
		}

		if result.Steps >= g.maxSteps {
			result.State = state.Snapshot()
			return result, fmt.Errorf("%w: limit=%d", ErrMaxStepsExceeded, g.maxSteps)
		}

		node, ok := g.nodes[current]
		if !ok {
			result.State = state.Snapshot()
			return result, fmt.Errorf("node %q is not registered", current)
		}

		if g.hooks.BeforeNode != nil {
			g.safeCallBeforeHook(ctx, current, state.Snapshot())
		}

		startedAt := time.Now()
		command, err := node(ctx, state)
		finishedAt := time.Now()

		trace := StepTrace{
			Node:        current,
			StartedAt:   startedAt,
			CompletedAt: finishedAt,
		}

		if command != nil && len(command.Update) > 0 {
			state.Merge(command.Update)
		}

		if err != nil {
			trace.Error = err.Error()
			result.Trace = append(result.Trace, trace)
			result.Steps++
			result.State = state.Snapshot()
			if g.hooks.AfterNode != nil {
				g.safeCallAfterHook(ctx, trace, state.Snapshot())
			}
			return result, fmt.Errorf("node %q failed: %w", current, err)
		}

		next, resolveErr := g.resolveNext(ctx, current, state, command)
		if resolveErr != nil {
			trace.Error = resolveErr.Error()
			result.Trace = append(result.Trace, trace)
			result.Steps++
			result.State = state.Snapshot()
			if g.hooks.AfterNode != nil {
				g.safeCallAfterHook(ctx, trace, state.Snapshot())
			}
			return result, resolveErr
		}

		trace.Next = next
		result.Trace = append(result.Trace, trace)
		result.Steps++
		result.State = state.Snapshot()

		if g.hooks.AfterNode != nil {
			g.safeCallAfterHook(ctx, trace, state.Snapshot())
		}

		current = next
	}

	result.State = state.Snapshot()
	return result, nil
}

func (g *Graph) resolveNext(ctx context.Context, current string, state *State, command *Command) (string, error) {
	if command != nil && command.Goto != "" {
		if err := g.validateTarget(command.Goto); err != nil {
			return "", fmt.Errorf("node %q returned invalid goto %q: %w", current, command.Goto, err)
		}
		return command.Goto, nil
	}

	if condition, ok := g.conditions[current]; ok {
		label, err := condition.router(ctx, state)
		if err != nil {
			return "", fmt.Errorf("router for node %q failed: %w", current, err)
		}

		next, ok := condition.routes[label]
		if !ok {
			return "", fmt.Errorf("router for node %q returned unknown route %q", current, label)
		}
		return next, nil
	}

	next, ok := g.edges[current]
	if !ok {
		return "", fmt.Errorf("node %q has no outgoing edge", current)
	}
	if err := g.validateTarget(next); err != nil {
		return "", fmt.Errorf("node %q points to invalid target %q: %w", current, next, err)
	}
	return next, nil
}

func (g *Graph) validateTarget(target string) error {
	switch target {
	case "":
		return errors.New("target cannot be empty")
	case End:
		return nil
	case Start:
		return errors.New("target cannot be start")
	}

	if _, ok := g.nodes[target]; !ok {
		return fmt.Errorf("node %q is not registered", target)
	}
	return nil
}

func (g *Graph) safeCallBeforeHook(ctx context.Context, node string, state map[string]any) {
	defer func() {
		if r := recover(); r != nil {
			// Hook panicked, log but don't crash the graph
			fmt.Fprintf(os.Stderr, "BeforeNode hook panicked for node %q: %v\n", node, r)
		}
	}()
	g.hooks.BeforeNode(ctx, node, state)
}

func (g *Graph) safeCallAfterHook(ctx context.Context, trace StepTrace, state map[string]any) {
	defer func() {
		if r := recover(); r != nil {
			// Hook panicked, log but don't crash the graph
			fmt.Fprintf(os.Stderr, "AfterNode hook panicked for node %q: %v\n", trace.Node, r)
		}
	}()
	g.hooks.AfterNode(ctx, trace, state)
}
