package types

import "time"

// StoredToolResult is a tool output persisted on disk to shrink in-memory context.
type StoredToolResult struct {
	ToolCallID   string     `json:"tool_call_id"`
	ToolName     string     `json:"tool_name"`
	Arguments    string     `json:"arguments"`
	Content      string     `json:"content"`
	Timestamp    time.Time  `json:"timestamp"`
	CompressedAt *time.Time `json:"compressed_at,omitempty"`
}

// ToolResultReference is a lightweight pointer replacing large tool outputs.
type ToolResultReference struct {
	Type     string `json:"type"`
	FilePath string `json:"file_path"`
	Summary  string `json:"summary"`
}

// CompactionHistory tracks one full-compaction event for auditing and recovery.
type CompactionHistory struct {
	SessionID     string    `json:"session_id"`
	Timestamp     time.Time `json:"timestamp"`
	TokensBefore  int       `json:"tokens_before"`
	TokensAfter   int       `json:"tokens_after"`
	MessagesCount int       `json:"messages_count"`
	SummaryLength int       `json:"summary_length"`
	FilesStored   int       `json:"files_stored"`
}

// CompactionConfig controls all compaction behavior.
type CompactionConfig struct {
	Enabled              bool   `json:"enabled"`
	LargeResultThreshold int    `json:"large_result_threshold"`
	OldMessageThreshold  int    `json:"old_message_threshold"`
	AutoCompactTokens    int    `json:"auto_compact_tokens"`
	SummaryMinTokens     int    `json:"summary_min_tokens"`
	SummaryMaxTokens     int    `json:"summary_max_tokens"`
	SessionStoragePath   string `json:"session_storage_path"`
	ShowTokenUsage       bool   `json:"show_token_usage"`
}

// DefaultCompactionConfig returns default settings for context compaction.
func DefaultCompactionConfig() *CompactionConfig {
	return &CompactionConfig{
		Enabled:              true,
		LargeResultThreshold: 1024,
		OldMessageThreshold:  20,
		AutoCompactTokens:    8000,
		SummaryMinTokens:     500,
		SummaryMaxTokens:     1000,
		SessionStoragePath:   "~/.claudego/sessions",
		ShowTokenUsage:       true,
	}
}
