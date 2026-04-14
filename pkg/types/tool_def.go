package types

// ToolCategory classifies tools for organization and filtering
type ToolCategory string

const (
	CategorySystem   ToolCategory = "system"
	CategoryFile     ToolCategory = "file"
	CategoryNetwork  ToolCategory = "network"
	CategoryProcess  ToolCategory = "process"
	CategoryExternal ToolCategory = "external"
)

// ToolMetadata holds additional information about a tool
type ToolMetadata struct {
	Category   ToolCategory
	SafeToSkip bool // if true, agent can proceed without this tool
	MaxRetries int  // max retry attempts on failure (0 = no retries)
}