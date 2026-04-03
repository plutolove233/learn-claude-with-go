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
}

type BashTool struct{}

func NewBashTool() *BashTool {
	return &BashTool{}
}

func (b *BashTool) Name() string {
	return "bash"
}

func (b *BashTool) Description() string {
	return "Run a shell command in the terminal"
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

func (b *BashTool) Execute(input map[string]interface{}) (string, error) {
	raw, ok := input["command"]
	if !ok {
		return "", errors.New("command field required")
	}
	cmdStr, ok := raw.(string)
	if !ok {
		return "", errors.New("command must be string")
	}
	for _, d := range dangerousCommands {
		if strings.Contains(cmdStr, d) {
			return "", errors.New("dangerous command blocked")
		}
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
