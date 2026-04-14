package types

// Skill represents a slash command skill loaded from a markdown file.
type Skill struct {
	Name         string // slash command name (e.g., "review")
	Description  string // short description shown in help
	Path         string // source file path
	Instructions string // markdown body: injected as system prompt fragment
}
