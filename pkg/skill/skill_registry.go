package skill

import (
	"claudego/pkg/types"
	"fmt"
	"strings"
	"sync"
)

// Registry holds all registered skills.
type Registry struct {
	mu     sync.RWMutex
	skills map[string]*types.Skill // skill name -> Skill
}

// defaultSkillRegistry is the global skill registry.
var defaultSkillRegistry = &Registry{
	skills: make(map[string]*types.Skill),
}

// NewRegistry creates a new registry.
func NewRegistry() *Registry {
	return &Registry{
		skills: make(map[string]*types.Skill),
	}
}

// DefaultRegistry returns the global skill registry.
func DefaultRegistry() *Registry {
	return defaultSkillRegistry
}

// Register adds a skill to the registry.
func (r *Registry) Register(s *types.Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if s.Name == "" {
		return fmt.Errorf("skill name cannot be empty")
	}
	if _, exists := r.skills[s.Name]; exists {
		return fmt.Errorf("skill %q is already registered", s.Name)
	}
	r.skills[s.Name] = s
	return nil
}

// Get returns a skill by name, or nil if not found.
func (r *Registry) Get(name string) (*types.Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.skills[name]
	return s, ok
}

// List returns all registered skills.
func (r *Registry) List() []*types.Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	skills := make([]*types.Skill, 0, len(r.skills))
	for _, s := range r.skills {
		skills = append(skills, s)
	}
	return skills
}

// LoadFromDir loads all skills from a directory and registers them.
func (r *Registry) LoadFromDir(dir string) error {
	loader := NewLoader(dir)
	skills, err := loader.LoadSkills()
	if err != nil {
		return err
	}

	for _, s := range skills {
		if err := r.Register(s); err != nil {
			// Skip duplicate skills (use first loaded)
			continue
		}
	}
	return nil
}

// LoadAndRegister is a convenience method that calls LoadFromDir on the default registry.
func LoadAndRegister(dir string) error {
	return defaultSkillRegistry.LoadFromDir(dir)
}

// Completions returns all skill names that start with the given prefix,
// each appended with a space for liner completion.
func (r *Registry) Completions(prefix string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var completions []string
	for name := range r.skills {
		if strings.HasPrefix(name, prefix) {
			completions = append(completions, "/"+name+" ")
		}
	}
	return completions
}