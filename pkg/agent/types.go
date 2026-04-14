package agent

// AgentTemplate represents a configurable agent loaded from a markdown file.
type AgentTemplate struct {
	Path        string   `json:"path"`        // file path of the agent template
	Name        string   `json:"name"`       // unique name for the agent template
	Description string   `json:"description"` // description of the agent template
	Tools       []string `json:"tools"`       // available tools for this agent
	Instructions string  `json:"instructions"` // agent instructions (markdown body)
}
