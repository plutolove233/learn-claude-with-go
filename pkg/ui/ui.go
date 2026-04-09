package ui

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
)

const (
	ColorTitle   = "38;2;215;119;87"
	ColorDefault = "38;2;235;235;235"
	ColorWarning = "38;2;230;185;90"
	ColorSuccess = "38;2;105;219;124"
	ColorError   = "38;2;220;95;105"
	ColorMuted   = "38;2;150;150;150"
	Reset       = "0"
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

	topBorder := paint(BorderTopLeft+strings.Repeat(BorderHorizontal, width-2)+BorderTopRight, ColorTitle, "")
	bottomBorder := paint(BorderBottomLeft+strings.Repeat(BorderHorizontal, width-2)+BorderBottomRight, ColorTitle, "")
	titleBar := paint(BorderVertical+" "+padRight(title, width-4)+" "+BorderVertical, ColorTitle, "")
	divider := paint(BorderCross+" "+strings.Repeat(BorderHorizontal, width-2), ColorMuted, "")

	fmt.Println(topBorder)
	fmt.Println(titleBar)
	fmt.Println(divider)

	lines := wrapText(content, width-4)
	for _, line := range lines {
		fmt.Println(paint(BorderVertical+" "+padRight(line, width-4)+" "+BorderVertical, ColorDefault, ""))
	}

	fmt.Println(bottomBorder)
}

func Step(num, total int, text string) {
	label := fmt.Sprintf("Step %d/%d:", num, total)
	fmt.Println(paint("["+label+"]", ColorTitle, "") + " " + text)
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
	top := paint("┌"+label+strings.Repeat("─", lineWidth)+"┐", ColorMuted, "")
	bot := paint("└"+strings.Repeat("─", width-2)+"┘", ColorMuted, "")
	fmt.Println(top)
	for _, line := range wrapText(args, width-4) {
		fmt.Println(paint(BorderVertical+" "+padRight(line, width-4)+" "+BorderVertical, ColorDefault, ""))
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

	top := paint("╭"+strings.Repeat("─", innerWidth)+"╮", ColorMuted, "")
	bottom := paint("╰"+strings.Repeat("─", innerWidth)+"╯", ColorMuted, "")

	// 第一行：✻ 名称  版本
	icon := paint("✻ "+name, ColorTitle, "")
	ver := paint(version, ColorMuted, "")
	nameVer := icon + "  " + ver
	row1 := borderRow(nameVer, innerWidth)

	// 第二行：空行
	row2 := borderRow("", innerWidth)

	// 第三行：模型名
	modelInfo := paint("model: ", ColorMuted, "") + paint(model, ColorDefault, "")
	row3 := borderRow(modelInfo, innerWidth)

	// 第四行：工作目录
	work_path := paint("directory: ", ColorMuted, "") + paint(cwd, ColorDefault, "")
	row4 := borderRow(work_path, innerWidth)

	fmt.Println()
	fmt.Println(top)
	fmt.Println(row1)
	fmt.Println(row2)
	fmt.Println(row3)
	fmt.Println(row4)
	fmt.Println(bottom)

	// 快捷键提示
	dot := paint(" · ", ColorMuted, "")
	hint := func(key, desc string) string {
		return paint(key, ColorMuted, "") + " " + paint(desc, ColorTitle, "")
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
	return paint("│ ", ColorMuted, "") + content + strings.Repeat(" ", pad) + paint("│", ColorMuted, "")
}
