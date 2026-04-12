package subagent

import "claudego/pkg/llm"

// TaskInput defines the input for the task tool (JSON schema for LLM)
type TaskInput struct {
	Task    string        `json:"task"`              // Task description
	Context []llm.Message `json:"context,omitempty"` // Forked context fragments
}

// TaskOutput defines the output of the task tool (JSON schema for LLM)
type TaskOutput struct {
	Summary string `json:"summary"` // Plain text summary from sub-agent
	Success bool   `json:"success"` // Whether completed normally (not max rounds)
}

// SubAgentConfig configures a sub-agent instance
type SubAgentConfig struct {
	Name      string      // Sub-agent name (for logging)
	MaxRounds int         // Max turns before forced stop
	Tools     []string    // Tool names allowed (whitelist)
	LLMClient *llm.Client // LLM client to use
}

// SubAgentResult is what a sub-agent returns after execution
type SubAgentResult struct {
	Summary string // Plain text summary
	Success bool   // true if normal completion, false if max rounds reached
}
