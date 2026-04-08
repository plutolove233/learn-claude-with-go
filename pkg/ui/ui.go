package ui

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
)

// Color constants (Claude Code style)
const (
	ColorTitle   = "\033[36m" // Cyan - headers, important labels
	ColorDefault = "\033[37m" // White - regular text
	ColorWarning = "\033[33m" // Orange - warnings
	ColorSuccess = "\033[32m" // Green - success messages
	ColorError   = "\033[31m" // Red - error messages
	ColorMuted   = "\033[90m" // Gray - timestamps, hints
	Reset        = "\033[0m"
)

// Border characters (simple lines)
const (
	BorderTopLeft     = "┌"
	BorderTopRight    = "┐"
	BorderBottomLeft  = "└"
	BorderBottomRight = "┘"
	BorderHorizontal  = "─"
	BorderVertical    = "│"
	BorderCross       = "├"
	BorderT           = "┬"
	BorderUpT         = "┴"
)

// DefaultWidth is the default box width
const DefaultWidth = 45

// Box prints a titled box with content
func Box(title, content string) {
	width := DefaultWidth

	titleWidth := runewidth.StringWidth(title)
	if titleWidth > width-4 {
		width = titleWidth + 4
	}

	for _, line := range wrapText(content, width-4) {
		lineWidth := runewidth.StringWidth(line)
		if lineWidth > width-4 {
			width = lineWidth + 4
		}
	}

	topBorder := BorderTopLeft + strings.Repeat(BorderHorizontal, width-2) + BorderTopRight
	bottomBorder := BorderBottomLeft + strings.Repeat(BorderHorizontal, width-2) + BorderBottomRight

	fmt.Println(ColorTitle + topBorder + Reset)
	fmt.Printf("%s %s%s%s\n",
		ColorTitle+BorderVertical+Reset,
		ColorTitle,
		padRight(title, width-4),
		ColorTitle+BorderVertical+Reset,
	)
	fmt.Printf("%s%s %s%s\n", ColorMuted+BorderCross+Reset, ColorMuted, strings.Repeat(BorderHorizontal, width-2), Reset)

	// Print content lines (wrap if needed)
	lines := wrapText(content, width-4)
	for _, line := range lines {
		fmt.Printf("%s %s%s%s\n",
			ColorDefault+BorderVertical+Reset,
			ColorDefault,
			padRight(line, width-4),
			ColorTitle+BorderVertical+Reset,
		)
	}

	fmt.Println(ColorTitle + bottomBorder + Reset)
}

// Step prints a step indicator: "Step 1/4: description"
func Step(num, total int, text string) {
	label := fmt.Sprintf("Step %d/%d:", num, total)
	fmt.Printf("%s[%s%s %s%s]%s %s\n",
		ColorTitle, ColorDefault, label, ColorTitle, ColorMuted, Reset, text)
}

// Success prints a success message
func Success(msg string) {
	fmt.Printf("%s✓ %s%s\n", ColorSuccess, Reset, msg)
}

// Error prints an error message
func Error(msg string) {
	fmt.Printf("%s✗ %s%s\n", ColorError, Reset, msg)
}

// Warning prints a warning message
func Warning(msg string) {
	fmt.Printf("%s⚠ %s%s\n", ColorWarning, Reset, msg)
}

// Info prints an info message
func Info(msg string) {
	fmt.Printf("%s%s%s\n", ColorTitle, msg, Reset)
}

// Divider prints a horizontal divider line
func Divider() {
	fmt.Printf("%s%s%s\n", ColorMuted, strings.Repeat("─", 45), Reset)
}

// ToolOutput prints tool output in a muted block. Long output is summarized as:
// first 2 lines + "... +N lines" + last 2 lines.
func ToolOutput(output string) {
	if output == "" {
		return
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) == 0 {
		return
	}

	const headLines = 2
	const tailLines = 2

	var display []string
	if len(lines) <= headLines+tailLines+1 {
		display = lines
	} else {
		display = append(display, lines[:headLines]...)
		display = append(display, fmt.Sprintf("... +%d lines", len(lines)-headLines-tailLines))
		display = append(display, lines[len(lines)-tailLines:]...)
	}

	indent := "  "
	for i, line := range display {
		color := ColorMuted
		if i == headLines && len(lines) > headLines+tailLines+1 && strings.HasPrefix(line, "... +") {
			fmt.Printf("%s%s%c %s%s\n", indent, color, '⎿', line, Reset)
			continue
		}
		if i == 0 {
			fmt.Printf("%s%s%c %s%s\n", indent, color, '⎿', line, Reset)
			continue
		}
		fmt.Printf("%s%s  %s%s\n", indent, color, line, Reset)
	}
}

// wrapText wraps text to fit within maxWidth
func wrapText(text string, maxWidth int) []string {
	if text == "" {
		return []string{""}
	}

	var lines []string
	for _, rawLine := range strings.Split(text, "\n") {
		if runewidth.StringWidth(rawLine) <= maxWidth {
			lines = append(lines, rawLine)
			continue
		}

		var currentLine strings.Builder
		currentWidth := 0
		for _, r := range rawLine {
			rw := runewidth.RuneWidth(r)
			if currentWidth+rw > maxWidth && currentLine.Len() > 0 {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
				currentWidth = 0
			}
			currentLine.WriteRune(r)
			currentWidth += rw
		}
		lines = append(lines, currentLine.String())
	}
	return lines
}

func padRight(text string, width int) string {
	padding := width - runewidth.StringWidth(text)
	if padding <= 0 {
		return text
	}
	return text + strings.Repeat(" ", padding)
}
