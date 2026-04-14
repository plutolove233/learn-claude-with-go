package skill

import (
	"claudego/pkg/types"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader loads skill templates from markdown files with frontmatter.
type Loader struct {
	skillsDir string
}

// NewLoader creates a new skill loader for the specified directory.
func NewLoader(skillsDir string) *Loader {
	return &Loader{skillsDir: skillsDir}
}

// LoadSkills loads all skills from the configured directory.
// Expected structure: ~/.claudego/skills/{skill_name}/SKILL.md
func (l *Loader) LoadSkills() ([]*types.Skill, error) {
	entries, err := os.ReadDir(l.skillsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read skills directory: %w", err)
	}

	var skills []*types.Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Each subdirectory is a skill named after the directory
		skillDir := filepath.Join(l.skillsDir, entry.Name())
		skillPath := filepath.Join(skillDir, "SKILL.md")

		s, err := l.LoadFromFile(skillPath)
		if err != nil {
			// Skip invalid skill directories
			continue
		}
		skills = append(skills, s)
	}

	return skills, nil
}

// LoadFromFile loads a single skill from a markdown file.
func (l *Loader) LoadFromFile(path string) (*types.Skill, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return ParseMarkdown(string(content), path)
}

// ParseMarkdown parses a markdown string with frontmatter into a Skill.
func ParseMarkdown(markdown string, path string) (*types.Skill, error) {
	content := strings.TrimPrefix(markdown, "---\n")
	if content == markdown {
		return nil, fmt.Errorf("no frontmatter found in %s", path)
	}

	parts := strings.SplitN(content, "---\n", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid frontmatter format in %s", path)
	}

	frontmatter := parts[0]
	skillContent := parts[1]

	var fm yamlFrontmatter
	if err := yaml.Unmarshal([]byte(frontmatter), &fm); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	if fm.Name == "" {
		return nil, fmt.Errorf("skill name is required in %s", path)
	}

	return &types.Skill{
		Path:         path,
		Name:         fm.Name,
		Description:  fm.Description,
		Instructions: strings.TrimSpace(skillContent),
	}, nil
}

// yamlFrontmatter represents the YAML frontmatter structure.
type yamlFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}
