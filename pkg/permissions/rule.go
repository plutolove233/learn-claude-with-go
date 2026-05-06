package permissions

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"
)

type ruleKind int

const (
	ruleKindToolWildcard ruleKind = iota
	ruleKindBash
	ruleKindFileHandler
)

type rule struct {
	raw     string
	tool    string
	kind    ruleKind
	action  string
	pattern string
}

func parseRule(raw string) (rule, error) {
	raw = strings.TrimSpace(raw)
	open := strings.Index(raw, "(")
	close := strings.LastIndex(raw, ")")
	if open <= 0 || close != len(raw)-1 {
		return rule{}, fmt.Errorf("invalid permission rule %q", raw)
	}

	tool := strings.TrimSpace(raw[:open])
	body := strings.TrimSpace(raw[open+1 : close])
	if tool == "" || body == "" {
		return rule{}, fmt.Errorf("invalid permission rule %q", raw)
	}

	r := rule{raw: raw, tool: tool, pattern: body}
	if body == "*" {
		r.kind = ruleKindToolWildcard
		return r, nil
	}

	switch tool {
	case "bash":
		r.kind = ruleKindBash
		return r, nil
	case "file_handler":
		parts := strings.SplitN(body, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return rule{}, fmt.Errorf("invalid file_handler permission rule %q", raw)
		}
		r.kind = ruleKindFileHandler
		r.action = parts[0]
		r.pattern = parts[1]
		return r, nil
	default:
		return rule{}, fmt.Errorf("tool %q only supports wildcard rules in this version", tool)
	}
}

func mustParseRule(raw string) rule {
	r, err := parseRule(raw)
	if err != nil {
		panic(err)
	}
	return r
}

func (r rule) matches(req Request) bool {
	if r.tool != req.ToolName {
		return false
	}
	if r.kind == ruleKindToolWildcard {
		return true
	}

	switch r.kind {
	case ruleKindBash:
		var args struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return false
		}
		return wildcardMatch(r.pattern, args.Command)
	case ruleKindFileHandler:
		var args struct {
			Action string `json:"action"`
			Path   string `json:"path"`
		}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return false
		}
		if r.action != "*" && r.action != args.Action {
			return false
		}
		return wildcardMatch(r.pattern, args.Path)
	default:
		return false
	}
}

func wildcardMatch(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	if ok, err := path.Match(pattern, value); err == nil && ok {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == value
	}

	parts := strings.Split(pattern, "*")
	pos := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		idx := strings.Index(value[pos:], part)
		if idx < 0 {
			return false
		}
		if i == 0 && !strings.HasPrefix(pattern, "*") && idx != 0 {
			return false
		}
		pos += idx + len(part)
	}
	last := parts[len(parts)-1]
	if last != "" && !strings.HasSuffix(pattern, "*") && !strings.HasSuffix(value, last) {
		return false
	}
	return true
}
