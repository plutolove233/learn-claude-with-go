package ui

import (
	"fmt"
	"strings"
)

const (
	openThinkTag  = "<think>"
	closeThinkTag = "</think>"
)

type thinkMode int

const (
	thinkModeVisible thinkMode = iota
	thinkModeHidden
)

type ThinkFilter struct {
	mode        thinkMode
	pending     string
	hasThinking bool
}

func (f *ThinkFilter) Feed(chunk string) string {
	data := f.pending + chunk
	f.pending = ""

	var visible string
	for len(data) > 0 {
		switch f.mode {
		case thinkModeVisible:
			if idx := indexOfTag(data, openThinkTag); idx >= 0 {
				visible += data[:idx]
				data = data[idx+len(openThinkTag):]
				f.mode = thinkModeHidden
				f.hasThinking = true
				continue
			}
			flush, keep := splitForPartialTag(data, openThinkTag)
			visible += flush
			f.pending = keep
			data = ""
		case thinkModeHidden:
			if idx := indexOfTag(data, closeThinkTag); idx >= 0 {
				data = data[idx+len(closeThinkTag):]
				f.mode = thinkModeVisible
				continue
			}
			_, keep := splitForPartialTag(data, closeThinkTag)
			f.pending = keep
			data = ""
		}
	}

	return visible
}

func (f *ThinkFilter) Finish() string {
	if f.mode == thinkModeVisible {
		tail := f.pending
		f.pending = ""
		return tail
	}
	f.pending = ""
	return ""
}

func (f *ThinkFilter) HasThinking() bool {
	return f.hasThinking
}

type AssistantStreamer struct {
	renderer       Renderer
	filter         ThinkFilter
	started        bool
	atLineStart    bool
	contentPrinted bool
	thinkingShown  bool
	firstLine      bool
	spinner        Spinner
	spinStep       int
}

func NewAssistantStreamer() *AssistantStreamer {
	return &AssistantStreamer{
		renderer:    defaultRenderer,
		atLineStart: true,
		firstLine:   true,
	}
}

// Spin returns the next spinner frame string for animation.
// Returns empty string when not in thinking state or content already printed.
func (s *AssistantStreamer) Spin() string {
	if !s.thinkingShown || s.contentPrinted {
		return ""
	}
	s.spinStep++
	return s.spinner.Frame(s.spinStep % len(s.spinner.Frames))
}

func (s *AssistantStreamer) Write(chunk string) string {
	hadThinking := s.filter.HasThinking()
	visible := s.filter.Feed(chunk)

	if !hadThinking && s.filter.HasThinking() && !s.thinkingShown && !s.contentPrinted {
		s.start()
		s.spinner = NewSpinner(s.renderer.Theme, "Thinking about the next edit")
		fmt.Print("  " + s.spinner.Frame(0))
		s.thinkingShown = true
		s.atLineStart = false
	}

	if visible != "" {
		s.start()
		s.writeVisible(visible)
	}

	return visible
}

func (s *AssistantStreamer) Finish() string {
	tail := s.filter.Finish()
	if tail != "" {
		s.start()
		s.writeVisible(tail)
	}
	if s.started && !s.atLineStart {
		fmt.Println()
		s.atLineStart = true
	}
	return tail
}

func (s *AssistantStreamer) start() {
	if s.started {
		return
	}
	fmt.Println(paint("● Claude", s.renderer.Theme.Claude, "1"))
	s.started = true
}

func (s *AssistantStreamer) writeVisible(content string) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		isLast := i == len(lines)-1

		if s.firstLine && line == "" {
			continue
		}

		if s.firstLine {
			if !s.atLineStart {
				fmt.Println()
			}
			fmt.Print("  " + paint("⎿", s.renderer.Theme.Inactive, "") + "  " + line)
			s.firstLine = false
			s.atLineStart = false
		} else if line != "" {
			if s.atLineStart {
				// 后续行只缩进，不重复打印 ⎿
				fmt.Print("     " + line)
			} else {
				fmt.Print(line)
			}
			s.atLineStart = false
		}

		if !isLast {
			fmt.Println()
			s.atLineStart = true
		}
	}
	s.contentPrinted = !s.firstLine
}

func indexOfTag(s, tag string) int {
	for i := 0; i+len(tag) <= len(s); i++ {
		if s[i:i+len(tag)] == tag {
			return i
		}
	}
	return -1
}

func splitForPartialTag(s, tag string) (flush, keep string) {
	maxOverlap := min(len(tag)-1, len(s))
	for overlap := maxOverlap; overlap > 0; overlap-- {
		if s[len(s)-overlap:] == tag[:overlap] {
			return s[:len(s)-overlap], s[len(s)-overlap:]
		}
	}
	return s, ""
}
