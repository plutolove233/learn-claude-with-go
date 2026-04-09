package ui

import (
	"strconv"
	"strings"

	"github.com/mattn/go-runewidth"
)

func (r Renderer) RenderMarkdown(src string, width int) []string {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	lines := strings.Split(src, "\n")
	var out []string

	for i := 0; i < len(lines); {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			out = append(out, "")
			i++
			continue
		}

		if strings.HasPrefix(trimmed, "```") {
			lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			var block []string
			i++
			for i < len(lines) && strings.TrimSpace(lines[i]) != "```" {
				block = append(block, lines[i])
				i++
			}
			if i < len(lines) {
				i++
			}
			out = append(out, r.renderCodeBlock(lang, block, width)...)
			continue
		}

		if isTableStart(lines, i) {
			tableLines, next := collectTable(lines, i)
			out = append(out, r.renderTable(tableLines)...)
			i = next
			continue
		}

		if strings.HasPrefix(trimmed, ">") {
			var quote []string
			for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), ">") {
				quote = append(quote, strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i]), ">")))
				i++
			}
			for _, ql := range wrapPlain(strings.Join(quote, " "), max(12, width-4)) {
				out = append(out, paint("│", r.Theme.QuoteBar, "2")+" "+paint(r.renderInline(ql), "", "3"))
			}
			continue
		}

		if h, ok := heading(line); ok {
			out = append(out, paint(r.renderInline(h), r.Theme.Text, "1;4"))
			i++
			continue
		}

		if bullet, content, ok := listItem(line, r.Theme); ok {
			for idx, wl := range wrapPlain(content, max(12, width-4)) {
				if idx == 0 {
					out = append(out, bullet+" "+r.renderInline(wl))
				} else {
					out = append(out, strings.Repeat(" ", visibleWidth(stripANSI(bullet))+1)+r.renderInline(wl))
				}
			}
			i++
			continue
		}

		var para []string
		for i < len(lines) {
			t := strings.TrimSpace(lines[i])
			if t == "" || strings.HasPrefix(t, "```") || strings.HasPrefix(t, ">") || isTableStart(lines, i) {
				break
			}
			if _, ok := heading(lines[i]); ok {
				break
			}
			if _, _, ok := listItem(lines[i], r.Theme); ok {
				break
			}
			para = append(para, t)
			i++
		}
		for _, pl := range wrapPlain(strings.Join(para, " "), width) {
			out = append(out, r.renderInline(pl))
		}
	}

	return trimTrailingBlankLines(out)
}

func (r Renderer) renderCodeBlock(lang string, block []string, width int) []string {
	border := paint("╭─ code", r.Theme.Suggestion, "1")
	if lang != "" {
		border += paint(" ["+lang+"]", r.Theme.Inactive, "")
	}
	out := []string{border}
	for _, line := range block {
		if line == "" {
			out = append(out, "│")
			continue
		}
		for _, wrapped := range wrapPlain(line, max(12, width-2)) {
			out = append(out, "│ "+highlightCode(wrapped, lang, r.Theme))
		}
	}
	out = append(out, paint("╰"+strings.Repeat("─", 14), r.Theme.Suggestion, ""))
	return out
}

func (r Renderer) renderInline(src string) string {
	var b strings.Builder
	for i := 0; i < len(src); {
		if src[i] == '[' {
			closeText := strings.IndexByte(src[i:], ']')
			if closeText > 0 && strings.HasPrefix(src[i+closeText:], "](") {
				closeURL := strings.IndexByte(src[i+closeText+2:], ')')
				if closeURL > 0 {
					text := src[i+1 : i+closeText]
					url := src[i+closeText+2 : i+closeText+2+closeURL]
					b.WriteString(paint(text, r.Theme.Suggestion, "4"))
					b.WriteString(paint(" ("+url+")", r.Theme.Inactive, "2"))
					i += closeText + 3 + closeURL
					continue
				}
			}
		}
		if src[i] == '`' {
			if end := strings.IndexByte(src[i+1:], '`'); end >= 0 {
				code := src[i+1 : i+1+end]
				b.WriteString(paint(code, r.Theme.InlineCode, ""))
				i += end + 2
				continue
			}
		}
		if strings.HasPrefix(src[i:], "**") {
			if end := strings.Index(src[i+2:], "**"); end >= 0 {
				b.WriteString(paint(src[i+2:i+2+end], "", "1"))
				i += end + 4
				continue
			}
		}
		if src[i] == '*' {
			if end := strings.IndexByte(src[i+1:], '*'); end >= 0 {
				b.WriteString(paint(src[i+1:i+1+end], "", "3"))
				i += end + 2
				continue
			}
		}
		b.WriteByte(src[i])
		i++
	}
	return b.String()
}

func heading(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	for level := 1; level <= 6; level++ {
		prefix := strings.Repeat("#", level) + " "
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix)), true
		}
	}
	return "", false
}

func listItem(line string, theme Theme) (bullet, content string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		return paint("•", theme.Claude, "1"), strings.TrimSpace(trimmed[2:]), true
	}
	if idx := orderedListIndex(trimmed); idx > 0 {
		parts := strings.SplitN(trimmed, ". ", 2)
		if len(parts) == 2 {
			return paint(parts[0]+".", theme.Suggestion, "1"), parts[1], true
		}
	}
	return "", "", false
}

func orderedListIndex(s string) int {
	for i, r := range s {
		if r == '.' {
			if i > 0 {
				if _, err := strconv.Atoi(s[:i]); err == nil {
					return i
				}
			}
			return -1
		}
		if r < '0' || r > '9' {
			return -1
		}
	}
	return -1
}

func wrapPlain(text string, width int) []string {
	if width < 8 {
		width = 8
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	words := strings.Fields(text)
	lines := make([]string, 0, len(words)/2+1)
	current := words[0]
	for _, word := range words[1:] {
		if runewidth.StringWidth(current+" "+word) <= width {
			current += " " + word
			continue
		}
		lines = append(lines, current)
		current = word
	}
	lines = append(lines, current)
	return lines
}

func highlightCode(line, lang string, theme Theme) string {
	keywords := map[string]map[string]struct{}{
		"go": {"func": {}, "return": {}, "type": {}, "struct": {}, "if": {}, "else": {}, "for": {}, "range": {}, "package": {}, "import": {}},
		"sh": {"if": {}, "then": {}, "fi": {}, "for": {}, "do": {}, "done": {}, "case": {}, "esac": {}},
	}

	set := keywords[strings.ToLower(lang)]
	var b strings.Builder
	for i := 0; i < len(line); {
		if strings.HasPrefix(line[i:], "//") || strings.HasPrefix(line[i:], "#") {
			b.WriteString(paint(line[i:], theme.Inactive, "2"))
			break
		}
		if line[i] == '"' || line[i] == '\'' {
			quote := line[i]
			j := i + 1
			for j < len(line) {
				if line[j] == '\\' && j+1 < len(line) {
					j += 2
					continue
				}
				if line[j] == quote {
					j++
					break
				}
				j++
			}
			b.WriteString(paint(line[i:j], theme.Suggestion, ""))
			i = j
			continue
		}
		if isIdentStart(line[i]) {
			j := i + 1
			for j < len(line) && isIdentPart(line[j]) {
				j++
			}
			word := line[i:j]
			if _, ok := set[word]; ok {
				b.WriteString(paint(word, theme.Claude, "1"))
			} else {
				b.WriteString(word)
			}
			i = j
			continue
		}
		b.WriteByte(line[i])
		i++
	}
	return b.String()
}

func isIdentStart(b byte) bool {
	return b == '_' || ('a' <= b && b <= 'z') || ('A' <= b && b <= 'Z')
}

func isIdentPart(b byte) bool {
	return isIdentStart(b) || ('0' <= b && b <= '9')
}

func trimTrailingBlankLines(lines []string) []string {
	end := len(lines)
	for end > 0 && strings.TrimSpace(stripANSI(lines[end-1])) == "" {
		end--
	}
	return lines[:end]
}

