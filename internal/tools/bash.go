package tools

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

var dangerousCommands = []string{
	"rm -rf /",
	"sudo",
	"shutdown",
	"reboot",
	"mkfs",
	"dd if=",
	":(){:|:&};:",   // fork bomb
	"curl.*|.*sh",   // pipe curl to sh
	"wget.*|.*sh",   // pipe wget to sh
}

func isDangerousCommand(cmdStr string) bool {
	// Normalize: remove quotes, collapse whitespace, convert to lowercase for comparison
	normalized := strings.ToLower(cmdStr)
	// Remove various quote styles
	normalized = strings.ReplaceAll(normalized, "\"", "")
	normalized = strings.ReplaceAll(normalized, "'", "")
	normalized = strings.ReplaceAll(normalized, "`", "")
	// Collapse multiple spaces into one
	for strings.Contains(normalized, "  ") {
		normalized = strings.ReplaceAll(normalized, "  ", " ")
	}
	normalized = strings.TrimSpace(normalized)

	for _, d := range dangerousCommands {
		if strings.Contains(normalized, d) {
			return true
		}
	}
	// Additional check: rm -rf /* or rm -rf //*
	if strings.HasPrefix(normalized, "rm -rf") &&
		(strings.Contains(normalized, "/*") || strings.Contains(normalized, "//*")) {
		return true
	}
	return false
}

type BashTool struct {
	BaseTool
}

func NewBashTool() *BashTool {
	return &BashTool{}
}

func (b *BashTool) Name() string {
	return "bash"
}

func (b *BashTool) Description() string {
	return "Run a shell command in the terminal"
}

func (b *BashTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Category:   CategoryProcess,
		SafeToSkip: false,
		MaxRetries: 0,
	}
}

func (b *BashTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The shell command to execute",
			},
		},
		"required": []string{"command"},
	}
}

func (b *BashTool) Execute(input map[string]any) (string, error) {
	raw, ok := input["command"]
	if !ok {
		return "", errors.New("command field required")
	}
	cmdStr, ok := raw.(string)
	if !ok {
		return "", errors.New("command must be string")
	}
	if isDangerousCommand(cmdStr) {
		return "", errors.New("dangerous command blocked")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", errors.New("timeout (120s)")
	}
	return string(out), err
}
