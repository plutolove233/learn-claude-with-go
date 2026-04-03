package tools

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type FileHandler struct {
}

func (f *FileHandler) Name() string {
	return "file_handler"
}

func (f *FileHandler) Description() string {
	return "Read and write files on the local filesystem. Use with caution, as it can access any file that the user running the program has access to."
}

var WORKDIR, _ = os.Getwd()

func safeFilePath(path string) bool {
	// 如果文件路径在WORKDIR目录下，或者在当前目录的子目录下，则认为是安全的
	if strings.HasPrefix(path, WORKDIR) || (os.IsPathSeparator(path[0]) || path[0] == '.') {
		return true
	}
	return false
}

func readfile(path string) (string, error) {
	content, err := os.ReadFile(path)
	return string(content), err
}

func writefile(path string, content string) error {
	os.WriteFile(path, []byte(content), os.ModePerm|os.ModeAppend)
	return nil
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
