package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mattn/go-runewidth"
)

type Renderer struct {
	Theme Theme
	Width int
}

var defaultRenderer = Renderer{
	Theme: DefaultLightTheme(),
	Width: 88,
}

func NewRenderer(theme Theme, width int) Renderer {
	if width < 48 {
		width = 48
	}
	return Renderer{Theme: theme, Width: width}
}

func (r Renderer) Section(title string) string {
	line := strings.Repeat("─", max(2, r.Width-visibleWidth(title)-4))
	return paint("┌─ "+title+" "+line, r.Theme.UserBorder, "1")
}

func (r Renderer) RenderTranscript(items []TranscriptItem) string {
	out := make([]string, 0, len(items)*8)
	for i, item := range items {
		if i > 0 {
			out = append(out, "")
		}
		out = append(out, r.RenderMessage(item.Role, item.Body)...)
	}
	return strings.Join(out, "\n")
}

func (r Renderer) RenderMessage(role, body string) []string {
	switch strings.ToLower(role) {
	case "user":
		prefix := paint("● User", r.Theme.UserBorder, "1")
		lines := wrapPlain(body, r.Width-4)
		if len(lines) == 0 {
			lines = []string{""}
		}
		out := []string{prefix}
		for _, line := range lines {
			out = append(out, "  "+line)
		}
		return out
	case "assistant":
		head := paint("● Claude", r.Theme.Claude, "1")
		out := []string{head}
		for _, line := range r.RenderMarkdown(body, r.Width-6) {
			if line == "" {
				out = append(out, "")
				continue
			}
			out = append(out, "  "+paint("⎿", r.Theme.Inactive, "")+"  "+line)
		}
		return out
	default:
		lines := wrapPlain(body, r.Width-4)
		out := []string{paint("● System", r.Theme.Warning, "1")}
		for _, line := range lines {
			out = append(out, "  "+paint(line, r.Theme.Inactive, "2"))
		}
		return out
	}
}

func (r Renderer) RenderDiff(filename, oldText, newText string) string {
	oldLines := splitKeepEmpty(oldText)
	newLines := splitKeepEmpty(newText)
	ops := diffLines(oldLines, newLines)

	out := []string{
		paint("╭─ "+filename, r.Theme.UserBorder, "1"),
		paint("│ - removed", r.Theme.DiffRemoved, ""),
		paint("│ + added", r.Theme.DiffAdded, ""),
		paint("├"+strings.Repeat("─", max(8, r.Width-1)), r.Theme.UserBorder, ""),
	}

	for i := 0; i < len(ops); i++ {
		op := ops[i]
		switch op.Kind {
		case "equal":
			out = append(out, r.diffLine(" ", op.OldNo, op.Text, r.Theme.Inactive))
		case "delete":
			if i+1 < len(ops) && ops[i+1].Kind == "insert" {
				left, right := highlightWordChange(op.Text, ops[i+1].Text, r.Theme)
				out = append(out, r.diffLine("-", op.OldNo, left, ""))
				out = append(out, r.diffLine("+", ops[i+1].NewNo, right, ""))
				i++
				continue
			}
			out = append(out, r.diffLine("-", op.OldNo, paint(op.Text, r.Theme.DiffRemoved, ""), ""))
		case "insert":
			out = append(out, r.diffLine("+", op.NewNo, paint(op.Text, r.Theme.DiffAdded, ""), ""))
		}
	}

	return strings.Join(out, "\n")
}

func (r Renderer) diffLine(marker string, lineNo int, text, lineColor string) string {
	number := "    "
	if lineNo > 0 {
		number = fmt.Sprintf("%4d", lineNo)
	}
	gutter := fmt.Sprintf("│ %s %s ", marker, number)
	full := gutter + text
	if lineColor != "" {
		return paint(full, lineColor, "")
	}
	if marker == "-" || marker == "+" {
		color := r.Theme.DiffRemoved
		if marker == "+" {
			color = r.Theme.DiffAdded
		}
		return paint(gutter, color, "1") + text
	}
	return paint(gutter, r.Theme.Subtle, "") + text
}

func (r Renderer) renderTable(lines []string) []string {
	rows := make([][]string, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, parseTableRow(line))
	}
	if len(rows) < 2 {
		return []string{strings.Join(lines, "\n")}
	}

	header := rows[0]
	body := make([][]string, 0, len(rows)-2)
	for _, row := range rows[2:] {
		body = append(body, row)
	}

	widths := make([]int, len(header))
	for i, cell := range header {
		widths[i] = visibleWidth(cell)
	}
	for _, row := range body {
		for i, cell := range row {
			if i < len(widths) && visibleWidth(cell) > widths[i] {
				widths[i] = visibleWidth(cell)
			}
		}
	}

	var out []string
	out = append(out, renderTableBorder("┌", "┬", "┐", widths))
	out = append(out, renderTableRow(header, widths, true, r.Theme))
	out = append(out, renderTableBorder("├", "┼", "┤", widths))
	for _, row := range body {
		out = append(out, renderTableRow(row, widths, false, r.Theme))
	}
	out = append(out, renderTableBorder("└", "┴", "┘", widths))
	return out
}

func (r Renderer) renderToolOutput(output string) []string {
	return r.RenderMarkdown(output, r.Width-4)
}

// ----------------------------------------------------------------

type TranscriptItem struct {
	Role string
	Body string
}

type Spinner struct {
	Frames  []string
	Message string
	Theme   Theme
}

func NewSpinner(theme Theme, message string) Spinner {
	return Spinner{
		Frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		Message: message,
		Theme:   theme,
	}
}

func (s Spinner) Frame(step int) string {
	frame := s.Frames[step%len(s.Frames)]
	return paint(frame, s.Theme.ClaudeShimmer, "1") + " " + paint(s.Message+"...", s.Theme.Claude, "")
}

// ----------------------------------------------------------------
// Table helpers
// ----------------------------------------------------------------

func isTableStart(lines []string, idx int) bool {
	if idx+1 >= len(lines) {
		return false
	}
	return strings.Contains(lines[idx], "|") && isTableSeparator(strings.TrimSpace(lines[idx+1]))
}

func collectTable(lines []string, idx int) ([]string, int) {
	start := idx
	for idx < len(lines) && strings.Contains(lines[idx], "|") && strings.TrimSpace(lines[idx]) != "" {
		idx++
	}
	return lines[start:idx], idx
}

func isTableSeparator(line string) bool {
	if !strings.Contains(line, "|") {
		return false
	}
	trimmed := strings.ReplaceAll(line, "|", "")
	trimmed = strings.ReplaceAll(trimmed, "-", "")
	trimmed = strings.ReplaceAll(trimmed, ":", "")
	trimmed = strings.ReplaceAll(trimmed, " ", "")
	return trimmed == ""
}

func parseTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	row := make([]string, 0, len(parts))
	for _, part := range parts {
		row = append(row, strings.TrimSpace(part))
	}
	return row
}

func renderTableBorder(left, mid, right string, widths []int) string {
	parts := make([]string, len(widths))
	for i, w := range widths {
		parts[i] = strings.Repeat("─", w+2)
	}
	return left + strings.Join(parts, mid) + right
}

func renderTableRow(row []string, widths []int, header bool, theme Theme) string {
	cells := make([]string, 0, len(widths))
	for i, w := range widths {
		cell := ""
		if i < len(row) {
			cell = row[i]
		}
		cell = padRight(cell, w)
		if header {
			cell = paint(cell, theme.Text, "1")
		}
		cells = append(cells, " "+cell+" ")
	}
	return "│" + strings.Join(cells, "│") + "│"
}

// ----------------------------------------------------------------
// Diff
// ----------------------------------------------------------------

type diffOp struct {
	Kind  string
	Text  string
	OldNo int
	NewNo int
}

func diffLines(oldLines, newLines []string) []diffOp {
	n, m := len(oldLines), len(newLines)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}

	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if oldLines[i] == newLines[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	var ops []diffOp
	i, j := 0, 0
	oldNo, newNo := 1, 1
	for i < n && j < m {
		if oldLines[i] == newLines[j] {
			ops = append(ops, diffOp{Kind: "equal", Text: oldLines[i], OldNo: oldNo, NewNo: newNo})
			i++
			j++
			oldNo++
			newNo++
			continue
		}
		if dp[i+1][j] >= dp[i][j+1] {
			ops = append(ops, diffOp{Kind: "delete", Text: oldLines[i], OldNo: oldNo})
			i++
			oldNo++
			continue
		}
		ops = append(ops, diffOp{Kind: "insert", Text: newLines[j], NewNo: newNo})
		j++
		newNo++
	}
	for i < n {
		ops = append(ops, diffOp{Kind: "delete", Text: oldLines[i], OldNo: oldNo})
		i++
		oldNo++
	}
	for j < m {
		ops = append(ops, diffOp{Kind: "insert", Text: newLines[j], NewNo: newNo})
		j++
		newNo++
	}
	return ops
}

func highlightWordChange(left, right string, theme Theme) (string, string) {
	lr := []rune(left)
	rr := []rune(right)

	prefix := 0
	for prefix < len(lr) && prefix < len(rr) && lr[prefix] == rr[prefix] {
		prefix++
	}

	suffix := 0
	for suffix < len(lr)-prefix && suffix < len(rr)-prefix && lr[len(lr)-1-suffix] == rr[len(rr)-1-suffix] {
		suffix++
	}

	leftMid := string(lr[prefix : len(lr)-suffix])
	rightMid := string(rr[prefix : len(rr)-suffix])
	leftText := string(lr[:prefix]) + paint(leftMid, theme.DiffRemovedWord, "") + string(lr[len(lr)-suffix:])
	rightText := string(rr[:prefix]) + paint(rightMid, theme.DiffAddedWord, "") + string(rr[len(rr)-suffix:])
	return paint(leftText, theme.DiffRemoved, ""), paint(rightText, theme.DiffAdded, "")
}

// ----------------------------------------------------------------
// Shared utilities
// ----------------------------------------------------------------

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func visibleWidth(s string) int {
	return runewidth.StringWidth(stripANSI(s))
}

func splitKeepEmpty(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return []string{""}
	}
	return strings.Split(s, "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func padRight(s string, width int) string {
	if n := visibleWidth(s); n < width {
		return s + strings.Repeat(" ", width-n)
	}
	return s
}

func paint(text, colorCode, extra string) string {
	if text == "" {
		return ""
	}
	codes := make([]string, 0, 2)
	if extra != "" {
		codes = append(codes, extra)
	}
	if colorCode != "" {
		codes = append(codes, colorCode)
	}
	if len(codes) == 0 {
		return text
	}
	return "\x1b[" + strings.Join(codes, ";") + "m" + text + "\x1b[0m"
}
