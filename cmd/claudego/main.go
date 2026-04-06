package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
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

	// Register default tools and get the registry
	tools.RegisterDefaults()
	registry := tools.GetRegistry()

	// Optionally enable/disable tools by category
	// registry.DisableByCategory(tools.CategoryNetwork)

	agent := loop.New(cfg, log, registry)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			select {
			case <-sigCh:
				cancel()
			case <-ctx.Done():
				return
			}
		}
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

		query = strings.TrimSpace(query)
		if query == "" || query == "q" || query == "exit" {
			break
		}

		line.AppendHistory(query)

		messages := []loop.Message{
			{Role: "user", Content: query},
		}

		if err := agent.Run(ctx, messages); err != nil {
			log.Warning("Agent run failed: %v", err)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		fmt.Println()
	}
}
