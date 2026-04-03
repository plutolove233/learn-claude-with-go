package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/peterh/liner"

	"claudego/internal/config"
	"claudego/internal/loop"
	"claudego/internal/tools"
	"claudego/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		fmt.Fprintf(os.Stderr, "Please create config at %s\n", config.DefaultConfigPath())
		fmt.Fprintf(os.Stderr, "Example: {\"api_key\": \"...\", \"base_url\": \"https://api.deepseek.com/v1\", \"model\": \"deepseek-chat\"}\n")
		os.Exit(1)
	}

	log := logger.GetLogger()
	defer log.Close()

	bashTool := tools.NewBashTool()
	agent := loop.New(cfg, log, []tools.Tool{bashTool})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	fmt.Println("ClaudeGo Agent (type 'q' to quit)")
	fmt.Println("=")

	line := liner.NewLiner()
	defer line.Close()
	line.SetCtrlCAborts(true)

	for {
		query, err := line.Prompt("s01 >> ")
		if err != nil {
			if err == liner.ErrPromptAborted {
				continue
			}
			break
		}

		query = trimSpace(query)
		if query == "" || query == "q" || query == "exit" {
			break
		}

		line.AppendHistory(query)

		messages := []loop.Message{
			{Role: "user", Content: query},
		}

		if err := agent.Run(ctx, messages); err != nil {
			log.Logf("Agent run failed: %v", err)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		fmt.Println()
	}
}

func trimSpace(s string) string {
	// Simple trim - remove leading/trailing whitespace
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
