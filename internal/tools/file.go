package tools

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var WORKDIR, _ = os.Getwd()

type FileHandler struct {
	BaseTool
}

func NewFileHandler() *FileHandler {
	return &FileHandler{}
}

func (f *FileHandler) Name() string {
	return "file_handler"
}

func (f *FileHandler) Description() string {
	return "Read and write files on the local filesystem. Use with caution, as it can access any file that the user running the program has access to."
}

func (f *FileHandler) Metadata() ToolMetadata {
	return ToolMetadata{
		Category:   CategoryFile,
		SafeToSkip: false,
		MaxRetries: 0,
	}
}

func safeFilePath(path string) bool {
	// Resolve to absolute path and verify it's within WORKDIR
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absWorkdir, err := filepath.Abs(WORKDIR)
	if err != nil {
		return false
	}
	// Must be within WORKDIR or its subdirectories
	return strings.HasPrefix(absPath, absWorkdir+string(filepath.Separator)) ||
		absPath == absWorkdir
}

func readfile(path string) (string, error) {
	content, err := os.ReadFile(path)
	return string(content), err
}

func writefile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func (f *FileHandler) Execute(input map[string]any) (string, error) {
	action, ok := input["action"].(string)
	if !ok {
		return "", errors.New("action field required and must be a string")
	}
	path, ok := input["path"].(string)
	if !ok {
		return "", errors.New("path field required and must be a string")
	}
	if !safeFilePath(path) {
		return "", fmt.Errorf("unsafe file path: %s, the path must be within %s or a subdirectory of it", path, WORKDIR)
	}
	switch action {
	case "read":
		return readfile(path)
	case "write":
		content, ok := input["content"].(string)
		if !ok {
			return "", errors.New("content field required for write action and must be a string")
		}
		return "", writefile(path, content)
	default:
		return "", errors.New("invalid action: must be 'read' or 'write'")
	}
}

func (f *FileHandler) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "The action to perform: read or write",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "The file path to read from or write to",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to write to the file (required for write action)",
			},
		},
		"required": []string{"action", "path"},
	}
}
