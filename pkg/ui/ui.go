package ui

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
)

const (
	ColorTitle   = "\033[38;2;215;119;87m"
	ColorDefault = "\033[38;2;235;235;235m"
	ColorWarning = "\033[38;2;230;185;90m"
	ColorSuccess = "\033[38;2;105;219;124m"
	ColorError   = "\033[38;2;220;95;105m"
	ColorMuted   = "\033[38;2;150;150;150m"
	Reset        = "\033[0m"
)

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

const DefaultWidth = 45

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

func Step(num, total int, text string) {
	label := fmt.Sprintf("Step %d/%d:", num, total)
	fmt.Printf("%s[%s%s %s%s]%s %s\n",
		ColorTitle, ColorDefault, label, ColorTitle, ColorMuted, Reset, text)
}

func Success(msg string) {
	fmt.Println(paint("✓"+msg, ColorSuccess, ""))
}

func Error(msg string) {
	fmt.Println(paint("✗"+msg, ColorError, ""))
}

func Warning(msg string) {
	fmt.Println(paint("⚠ "+msg, ColorWarning, ""))
}

func Info(msg string) {
	fmt.Println(paint(msg, ColorTitle, ""))
}

func Divider() {
	fmt.Println(paint(strings.Repeat("─", 45), ColorMuted, ""))
}

func ToolOutput(output string) {
	for _, line := range defaultRenderer.renderToolOutput(output) {
		fmt.Println(line)
	}
}

func Blank() {
	fmt.Println()
}

func ToolCall(name, args string) {
	width := 50
	label := "─ " + name + " "
	lineWidth := width - 2 - runewidth.StringWidth(label)
	if lineWidth < 0 {
		lineWidth = 0
	}
	top := ColorMuted + "┌" + label + strings.Repeat("─", lineWidth) + "┐" + Reset
	bot := ColorMuted + "└" + strings.Repeat("─", width-2) + "┘" + Reset
	fmt.Println(top)
	for _, line := range wrapText(args, width-4) {
		fmt.Printf("%s %s%s%s\n", ColorMuted+BorderVertical+Reset, ColorDefault, padRight(line, width-4), ColorMuted+BorderVertical+Reset)
	}
	fmt.Println(bot)
}

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

// ui.go 中新增

func Welcome(name, version, model string, cwd string) {
	const innerWidth = 64 // 边框内宽度，可按需调整

	// cwd 过长时截断，保留末尾部分
	if runewidth.StringWidth(cwd) > innerWidth-2 {
		runes := []rune(cwd)
		for runewidth.StringWidth("…"+string(runes)) > innerWidth-2 {
			runes = runes[1:]
		}
		cwd = "…" + string(runes)
	}

	top := ColorMuted + "╭" + strings.Repeat("─", innerWidth) + "╮" + Reset
	bottom := ColorMuted + "╰" + strings.Repeat("─", innerWidth) + "╯" + Reset

	// 第一行：✻ 名称  版本
	icon := ColorTitle + "✻ " + name + Reset
	ver := ColorMuted + version + Reset
	nameVer := icon + "  " + ver
	row1 := borderRow(nameVer, innerWidth)

	// 第二行：空行
	row2 := borderRow("", innerWidth)

	// 第三行：模型名
	modelInfo := ColorMuted + "model: " + Reset + ColorDefault + model + Reset
	row3 := borderRow(modelInfo, innerWidth)

	// 第四行：工作目录
	work_path := ColorMuted + "directory: " + Reset + ColorDefault + cwd + Reset
	row4 := borderRow(work_path, innerWidth)

	fmt.Println()
	fmt.Println(top)
	fmt.Println(row1)
	fmt.Println(row2)
	fmt.Println(row3)
	fmt.Println(row4)
	fmt.Println(bottom)

	// 快捷键提示
	dot := ColorMuted + " · " + Reset
	hint := func(key, desc string) string {
		return ColorMuted + key + Reset + " " + ColorTitle + desc + Reset
	}
	fmt.Println()
	fmt.Println("  " +
		hint("/plan", "for plan mode") + dot +
		hint("q", "to quit") + dot +
		hint("?", "for help"),
	)
	fmt.Println()
}

// borderRow 把内容放在 │ │ 之间，右侧用空格补齐
func borderRow(content string, innerWidth int) string {
	visible := visibleWidth(content)
	pad := innerWidth - visible - 1 // 左侧 1 空格
	if pad < 0 {
		pad = 0
	}
	return ColorMuted + "│" + Reset + " " + content + strings.Repeat(" ", pad) + ColorMuted + "│" + Reset
}
