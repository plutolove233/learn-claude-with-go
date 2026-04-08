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
	"claudego/internal/plan"
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
	executor := plan.NewExecutor(cfg, log, registry)

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

	fmt.Println("ClaudeGo Agent (type 'q' to quit, '/plan' for plan commands)")
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

		// Handle plan commands
		if strings.HasPrefix(query, "/plan") {
			handlePlanCommand(ctx, executor, query)
			fmt.Println()
			continue
		}

		// Auto-plan mode: check if this is a complex task that needs planning
		if isComplexTask(query) {
			fmt.Println("\033[36m🔍 Detected complex task - entering plan mode...\033[0m")
			if _, err := executor.RunWithPlan(ctx, query); err != nil {
				log.Warning("Plan execution failed: %v", err)
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		} else {
			// Regular single-turn mode
			messages := []loop.Message{
				{Role: "user", Content: query},
			}

			if err := agent.Run(ctx, messages); err != nil {
				log.Warning("Agent run failed: %v", err)
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		}
		fmt.Println()
	}
}

func handlePlanCommand(ctx context.Context, executor *plan.Executor, cmd string) {
	args := strings.Fields(cmd)
	if len(args) < 2 {
		printPlanHelp()
		return
	}

	switch args[1] {
	case "list":
		plans, err := plan.ListPlans()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing plans: %v\n", err)
			return
		}
		if len(plans) == 0 {
			fmt.Println("No plans found.")
			return
		}
		fmt.Println("\033[36m📋 Plans:\033[0m")
		for _, p := range plans {
			fmt.Printf("  %s - %s (%s) [%s]\n", p.ID, p.Name, p.Goal, p.Status)
		}

	case "resume":
		if len(args) < 3 {
			fmt.Println("Usage: /plan resume <plan_id>")
			return
		}
		p, err := plan.LoadPlan(args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading plan: %v\n", err)
			return
		}
		fmt.Printf("\033[36mResuming plan: %s\033[0m\n", p.Name)
		if err := executor.ResumePlan(ctx, p); err != nil {
			fmt.Fprintf(os.Stderr, "Error resuming plan: %v\n", err)
		}

	case "status":
		if len(args) < 3 {
			fmt.Println("Usage: /plan status <plan_id>")
			return
		}
		p, err := plan.LoadPlan(args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading plan: %v\n", err)
			return
		}
		executor.DisplayPlan(p)

	default:
		printPlanHelp()
	}
}

func printPlanHelp() {
	fmt.Println("\033[36mPlan Commands:\033[0m")
	fmt.Println("  /plan list              - List all plans")
	fmt.Println("  /plan resume <id>       - Resume a paused plan")
	fmt.Println("  /plan status <id>       - Show plan status")
}

func isComplexTask(query string) bool {
	// Keywords that indicate complex multi-step tasks
	complexKeywords := []string{
		"refactor", "重构", "migrate", "迁移", "implement", "实现",
		"build", "创建", "develop", "开发", "setup", "设置",
		"convert", "转换", "upgrade", "升级", "audit", "review",
		"analyze", "分析", "design", "设计", "architecture",
		"系统", "项目", "multiple", "several", "many",
	}

	queryLower := strings.ToLower(query)
	for _, kw := range complexKeywords {
		if strings.Contains(queryLower, kw) {
			return true
		}
	}

	// Tasks with multiple items separated by "and" or "、"
	if strings.Contains(query, " and ") || strings.Contains(query, "、") {
		return true
	}

	return false
}
