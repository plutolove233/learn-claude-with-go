package tools

import (
	"fmt"
	"sync"
)

// Registry holds all registered tools and their enabled state
type Registry struct {
	mu      sync.RWMutex
	tools   map[string]Tool   // tool name -> Tool
	enabled map[string]bool   // tool name -> enabled (default true)
}

// Global default registry
var defaultRegistry = &Registry{
	tools:   make(map[string]Tool),
	enabled: make(map[string]bool),
}

// GetRegistry returns the global tool registry
func GetRegistry() *Registry {
	return defaultRegistry
}

// Register adds a tool to the registry by its Name()
// Panics if a tool with the same name is already registered
func (r *Registry) Register(t Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := t.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %q is already registered", name)
	}
	r.tools[name] = t
	r.enabled[name] = true
	return nil
}

// Unregister removes a tool from the registry by name
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
	delete(r.enabled, name)
}

// Get returns a tool by name, or nil if not found
func (r *Registry) Get(name string) Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

// List returns all registered tools
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// ListByCategory returns tools filtered by category
func (r *Registry) ListByCategory(category ToolCategory) []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		if t.Metadata().Category == category {
			tools = append(tools, t)
		}
	}
	return tools
}

// Names returns the names of all registered tools
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered tools
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// Clear removes all tools from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools = make(map[string]Tool)
	r.enabled = make(map[string]bool)
}

// --- Enabled/Disabled management ---

// Enable allows a tool by name. Returns false if tool not found.
func (r *Registry) Enable(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[name]; !exists {
		return false
	}
	r.enabled[name] = true
	return true
}

// Disable prevents a tool from being used. Returns false if tool not found.
func (r *Registry) Disable(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[name]; !exists {
		return false
	}
	r.enabled[name] = false
	return true
}

// IsEnabled returns whether a tool is currently enabled
func (r *Registry) IsEnabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if enabled, ok := r.enabled[name]; ok {
		return enabled
	}
	return true
}

// EnableAll enables all tools currently in the registry
func (r *Registry) EnableAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.tools {
		r.enabled[t.Name()] = true
	}
}

// DisableAll disables all tools
func (r *Registry) DisableAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.tools {
		r.enabled[t.Name()] = false
	}
}

// EnabledTools returns the list of currently enabled tools
func (r *Registry) EnabledTools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	enabled := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		if r.enabled[t.Name()] {
			enabled = append(enabled, t)
		}
	}
	return enabled
}

// EnableByCategory enables all tools in a given category
func (r *Registry) EnableByCategory(category ToolCategory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.tools {
		if t.Metadata().Category == category {
			r.enabled[t.Name()] = true
		}
	}
}

// DisableByCategory disables all tools in a given category
func (r *Registry) DisableByCategory(category ToolCategory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.tools {
		if t.Metadata().Category == category {
			r.enabled[t.Name()] = false
		}
	}
}

// RegisterDefaults registers the built-in tools (bash, file_handler)
func RegisterDefaults() {
	registry := GetRegistry()
	if err := registry.Register(NewBashTool()); err != nil {
		panic("failed to register bash tool: " + err.Error())
	}
	if err := registry.Register(NewFileHandler()); err != nil {
		panic("failed to register file handler: " + err.Error())
	}
}
