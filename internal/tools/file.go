package tools

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type fileHandlerInput struct {
	Action  string `json:"action"  validate:"required,oneof=read write"`
	Path    string `json:"path"    validate:"required,filepath"`
	Content string `json:"content" validate:"required_if=Action write"`
}

type FileHandler struct {
	BaseTool[fileHandlerInput]
}

func NewFileHandler() *FileHandler {
	return &FileHandler{
		BaseTool[fileHandlerInput]{
			name:          "file_handler",
			description:   "Read and write files on the local filesystem",
			fn:            fileExecute,
			extraValidate: sandboxCheck,
		},
	}
}

func (f *FileHandler) Name() string {
	return "file_handler"
}

func (f *FileHandler) Description() string {
	return "Read and write files on the local filesystem. Only paths within the current working directory and its subdirectories are allowed." +
		"Read action returns the content of the file. Write action writes content to the file, creating it if it doesn't exist." +
		"Input must include 'action' (read or write), 'path', and for write action, 'content'."
}

func (f *FileHandler) Metadata() ToolMetadata {
	return ToolMetadata{
		Category:   CategoryFile,
		SafeToSkip: false,
		MaxRetries: 0,
	}
}

func (f *FileHandler) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"read", "write"},
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

// sandboxCheck ensures the requested path stays within the working directory.
// Separated from executeFile so the pipeline reports it as a validation error
// before any filesystem access occurs.
func sandboxCheck(p fileHandlerInput) error {
	workdir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve working directory: %w", err)
	}
	absPath, err := filepath.Abs(p.Path)
	if err != nil {
		return fmt.Errorf("resolve path %q: %w", p.Path, err)
	}
	absWorkdir, err := filepath.Abs(workdir)
	if err != nil {
		return fmt.Errorf("resolve working directory %q: %w", workdir, err)
	}
	if absPath != absWorkdir && !strings.HasPrefix(absPath, absWorkdir+string(filepath.Separator)) {
		return fmt.Errorf("path %q is outside the working directory %q", p.Path, workdir)
	}
	return nil
}

func fileExecute(input fileHandlerInput) (string, error) {
	switch input.Action {
	case "read":
		return readfile(input.Path)
	case "write":
		return "", writefile(input.Path, input.Content)
	default:
		return "", errors.New("invalid action: must be 'read' or 'write'")
	}
}

func readfile(path string) (string, error) {
	content, err := os.ReadFile(path)
	return string(content), err
}

func writefile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
