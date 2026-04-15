package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"claudego/internal/config"
	"claudego/internal/loop"
	"claudego/internal/plan"
	"claudego/internal/tools"
	"claudego/pkg/conversation"
	"claudego/pkg/logger"
	"claudego/pkg/skill"
	"claudego/pkg/ui"
	"claudego/utils"

	"github.com/peterh/liner"
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

	// Load skills from ~/.claudego/skills/
	skillRegistry := skill.GetSkillRegistry()
	home, _ := os.UserHomeDir()
	skillsDir := filepath.Join(home, ".claudego", "skills")
	if err := skillRegistry.LoadFromDir(skillsDir); err != nil {
		// Skills are optional - log warning but don't fail startup
		log.Warning("Failed to load skills: %v", err)
	}

	tools.RegisterDefaults()
	registry := tools.GetRegistry()


	conv := conversation.New()
	agent := loop.New(cfg, log, registry)
	executor := plan.NewExecutor(cfg, log, registry)

	cwd, _ := os.Getwd()
	cwd, _ = utils.AbsToTilde(cwd)
	ui.Welcome("ClaudeGo Agent", "v1.0", cfg.Model, cwd)

	rootCtx := context.Background()

	line := liner.NewLiner()
	defer line.Close()
	line.SetCtrlCAborts(true)

	// Configure skill auto-completion for liner
	line.SetCompleter(func(line string) []string {
		if len(line) > 0 && line[0] == '/' {
			return skillRegistry.Completions(line[1:])
		}
		return nil
	})

	for {
		query, err := line.Prompt(">_ ")
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

		ctx, cancel := context.WithCancel(rootCtx)

		// 启动 Ctrl+C 中断监听，仅在模型调用期间生效
		stopListener := startInterruptListener(cancel)

		// Check for skill slash commands
		if matched, loadedSkill, err := skill.MatchAndExecute(ctx, query, skillRegistry, registry); matched {
			if err != nil {
				fmt.Fprintf(os.Stderr, "Skill error: %v\n", err)
			} else if strings.TrimSpace(loadedSkill) != "" {
				conv.AddUserMessage(loadedSkill)
			}
			// stopListener()
			// cancel()
			// fmt.Println()
			// continue
		}

		if strings.HasPrefix(query, "/plan") {
			handlePlanCommand(ctx, executor, query)
		} else if isComplexTask(query) {
			ui.Info("Detected complex task - entering plan mode...")
			if _, err := executor.RunWithPlan(ctx, query); err != nil {
				if ctx.Err() != nil {
					ui.Warning("Interrupted. Rolling back conversation.")
				} else {
					log.Warning("Plan execution failed: %v", err)
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
			}
		} else {
			conv.AddUserMessage(query)
			checkpoint := conv.Checkpoint()
			messages := conv.GetMessages()

			if err := agent.Run(ctx, messages); err != nil {
				if ctx.Err() != nil {
					ui.Warning("Generation interrupted. Rolling back conversation.")
					conv.Rollback(checkpoint)
				} else {
					log.Warning("Agent run failed: %v", err)
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
			}
		}

		stopListener() // 停止监听，恢复终端状态
		cancel()
		fmt.Println()
	}
}

// startInterruptListener 在模型调用期间监听Ctrl+C，
// 触发都会调用 cancel() 中断 context。
// 返回的 stop 函数必须在模型调用结束后调用，以恢复终端状态。
func startInterruptListener(cancel context.CancelFunc) (stop func()) {
	done := make(chan struct{})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)

	go func() {
		defer signal.Stop(sigCh)
		select {
		case <-sigCh:
			cancel()
		case <-done:
		}
	}()

	return func() { close(done) }
}

// -------- 以下函数无改动 --------

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
		ui.Info("Plans:")
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
		ui.Info(fmt.Sprintf("Resuming plan: %s", p.Name))
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
	ui.Info("Plan Commands:")
	fmt.Println("  /plan list              - List all plans")
	fmt.Println("  /plan resume <id>       - Resume a paused plan")
	fmt.Println("  /plan status <id>       - Show plan status")
}

func isComplexTask(query string) bool {
	complexKeywords := []string{
		"refactor", "重构", "migrate", "迁移", "implement", "实现",
		"build", "创建", "develop", "开发", "setup", "设置",
		"convert", "转换", "upgrade", "升级", "audit",
		"analyze", "分析", "design", "设计", "architecture",
		"系统", "项目", "multiple", "several", "many",
	}

	queryLower := strings.ToLower(query)
	for _, kw := range complexKeywords {
		if strings.Contains(queryLower, kw) {
			return true
		}
	}

	return strings.Contains(query, " and ") || strings.Contains(query, "、")
}
