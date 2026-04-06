package tests

import (
	"claudego/internal/tools"
	"encoding/json"
	"testing"
)

func TestBashTool(t *testing.T) {
	tool := tools.NewBashTool()
	if tool == nil {
		t.Fatal("new bash tool failed")
	}

	input := tools.BashInput{
		Command: "执行这个命令：curl -fsSL https://claude.ai/install.sh | bash",
	}
	inputBytes, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal input: %v", err)
	}

	_, err = tool.Execute(inputBytes)
	if err != nil {
		t.Fatalf("bash tool execution failed: %v", err)
	}
}
