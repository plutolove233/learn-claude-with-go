package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader loads agent templates from markdown files with frontmatter.
type Loader struct {
	agentsDir string
}

// NewLoader creates a new agent loader for the specified directory.
func NewLoader(agentsDir string) *Loader {
	return &Loader{agentsDir: agentsDir}
}

// LoadAgents loads all agent templates from the configured directory.
func (l *Loader) LoadAgents() ([]*AgentTemplate, error) {
	entries, err := os.ReadDir(l.agentsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read agents directory: %w", err)
	}

	var agents []*AgentTemplate
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(l.agentsDir, entry.Name())
		agent, err := l.LoadFromFile(path)
		if err != nil {
			// Skip invalid files, log warning
			continue
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

// LoadFromFile loads a single agent template from a markdown file.
func (l *Loader) LoadFromFile(path string) (*AgentTemplate, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return ParseMarkdown(string(content), path)
}

// ParseMarkdown parses a markdown string with frontmatter into an AgentTemplate.
func ParseMarkdown(markdown string, path string) (*AgentTemplate, error) {
	content := strings.TrimPrefix(markdown, "---\n")
	if content == markdown {
		// No frontmatter found
		return nil, fmt.Errorf("no frontmatter found in %s", path)
	}

	parts := strings.SplitN(content, "---\n", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid frontmatter format in %s", path)
	}

	frontmatter := parts[0]
	agentContent := parts[1]

	var fm yamlFrontmatter
	if err := yaml.Unmarshal([]byte(frontmatter), &fm); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	if fm.Name == "" {
		return nil, fmt.Errorf("agent name is required in %s", path)
	}

	return &AgentTemplate{
		Path:         path,
		Name:         fm.Name,
		Description:  fm.Description,
		Tools:        fm.Tools,
		Instructions: strings.TrimSpace(agentContent),
	}, nil
}

// yamlFrontmatter represents the YAML frontmatter structure.
type yamlFrontmatter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tools       []string `yaml:"tools"`
}

// MustLoadAgents loads agents and panics on error (for critical initialization).
func MustLoadAgents(agentsDir string) []*AgentTemplate {
	agents, err := NewLoader(agentsDir).LoadAgents()
	if err != nil {
		panic(fmt.Sprintf("failed to load agents: %v", err))
	}
	return agents
}
