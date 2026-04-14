package tools

import (
	"claudego/pkg/types"
	"context"
	"errors"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type BashInput struct {
	Command string `json:"command" validate:"required"`
}

type BashTool struct {
	BaseTool[BashInput]
}

func NewBashTool() *BashTool {
	return &BashTool{
		BaseTool[BashInput]{
			name:        "bash",
			description: "Run a shell command in the terminal",
			parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "The shell command to execute",
					},
				},
				"required": []string{"command"},
			},
			metadata: types.ToolMetadata{
				Category:   types.CategoryProcess,
				SafeToSkip: false,
				MaxRetries: 0,
			},
			fn:            bashExecute,
			extraValidate: isDangerousCommand,
		},
	}
}

func bashExecute(input BashInput) (string, error) {
	cmdStr := input.Command
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", errors.New("timeout (120s)")
	}
	return string(out), err
}

// dangerousPatterns holds compiled regexes that describe commands too risky
// to execute. Each entry pairs a pattern with a human-readable reason so
// error messages are actionable.
//
// Patterns are matched against a normalized form of the input (lowercase,
// quotes stripped, whitespace collapsed) to reduce trivial bypass attempts.
var dangerousPatterns = []struct {
	re     *regexp.Regexp
	reason string
}{
	// Recursive delete of root or all files
	{regexp.MustCompile(`rm\s+-[a-z]*r[a-z]*f\s+/`), "recursive delete from root"},
	{regexp.MustCompile(`rm\s+-[a-z]*f[a-z]*r\s+/`), "recursive delete from root"},
	{regexp.MustCompile(`rm\s+.*-rf\s+/\*`), "recursive delete of /*"},

	// Privilege escalation
	{regexp.MustCompile(`\bsudo\b`), "privilege escalation via sudo"},
	{regexp.MustCompile(`\bsu\b`), "privilege escalation via su"},

	// System control
	{regexp.MustCompile(`\b(shutdown|reboot|halt|poweroff|init\s+[06])\b`), "system shutdown or reboot"},

	// Disk and filesystem operations
	{regexp.MustCompile(`\bmkfs\b`), "filesystem format"},
	{regexp.MustCompile(`\bdd\b.*\bif=`), "raw disk write via dd"},
	{regexp.MustCompile(`\bfdisk\b`), "partition table modification"},
	{regexp.MustCompile(`\bparted\b`), "partition table modification"},

	// Fork bomb variants
	{regexp.MustCompile(`:\(\)\s*\{.*:\s*\|.*:.*&.*\}`), "fork bomb"},
	{regexp.MustCompile(`\bfork\s*bomb\b`), "fork bomb"},

	// Piping remote content directly into a shell
	{regexp.MustCompile(`(curl|wget)\s+.*\|\s*(ba)?sh`), "remote code execution via pipe"},
	{regexp.MustCompile(`(curl|wget)\s+.*\|\s*bash`), "remote code execution via pipe"},

	// Overwriting critical system files
	{regexp.MustCompile(`>\s*/etc/(passwd|shadow|sudoers|hosts)`), "overwrite of critical system file"},

	// Loading kernel modules
	{regexp.MustCompile(`\b(insmod|modprobe|rmmod)\b`), "kernel module manipulation"},

	// Changing file ownership of system paths
	{regexp.MustCompile(`\bchown\b.*\s+/`), "ownership change on system path"},
	{regexp.MustCompile(`\bchmod\b.*\s+/`), "permission change on system path"},
}

// normalize reduces the command string to a canonical form that makes
// pattern matching harder to bypass with trivial obfuscation (extra spaces,
// mixed case, extraneous quotes).
func normalize(cmd string) string {
	s := strings.ToLower(cmd)
	s = strings.NewReplacer(`"`, ``, `'`, ``, "`", ``).Replace(s)
	// Collapse runs of whitespace to a single space
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// isDangerousCommand returns an error describing the first dangerous pattern
// found in input.Command, or nil if the command appears safe.
func isDangerousCommand(input BashInput) error {
	normalized := normalize(input.Command)
	for _, p := range dangerousPatterns {
		if p.re.MatchString(normalized) {
			return errors.New("dangerous command detected: " + p.reason)
		}
	}
	return nil
}
