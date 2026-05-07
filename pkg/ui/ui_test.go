package ui

import (
	"strings"
	"testing"
	"unicode"
)

func TestPermissionChoicePromptIsLinerSafe(t *testing.T) {
	prompt := permissionChoicePrompt()
	if prompt == "" {
		t.Fatal("expected permission prompt to be non-empty")
	}
	for _, r := range prompt {
		if unicode.Is(unicode.C, r) {
			t.Fatalf("permission prompt contains control character %q in %q", r, prompt)
		}
	}
	if strings.Contains(prompt, "\x1b") {
		t.Fatalf("permission prompt must not contain ANSI escape sequences: %q", prompt)
	}
}
