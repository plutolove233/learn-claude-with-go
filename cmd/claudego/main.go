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

	line := liner.NewLiner()
	defer line.Close()
	line.SetCtrlCAborts(true)
	line.SetCompleter(func(line string) []string {
		if len(line) > 0 && line[0] == '/' {
			return skillRegistry.Completions(line[1:])
		}
		return nil
	})

	tools.RegisterDefaults()
	registry := tools.GetRegistry()

	conv := conversation.New()
	agent, err := loop.NewWithPermissionPrompter(cfg, log, registry, ui.NewPermissionPrompter(line.Prompt))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize agent: %v\n", err)
		os.Exit(1)
	}

	cwd, _ := os.Getwd()
	cwd, _ = utils.AbsToTilde(cwd)
	ui.Welcome("ClaudeGo Agent", "v1.0", cfg.Model, cwd)

	rootCtx := context.Background()

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
			stopListener()
			cancel()
			fmt.Println()
			continue
		}

		if strings.HasPrefix(query, "/plan") {

			// } else if isComplexTask(query) {
			// 	ui.Info("Detected complex task - entering plan mode...")
			// 	if _, err := executor.RunWithPlan(ctx, query); err != nil {
			// 		if ctx.Err() != nil {
			// 			ui.Warning("Interrupted. Rolling back conversation.")
			// 		} else {
			// 			log.Warning("Plan execution failed: %v", err)
			// 			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			// 		}
			// 	}
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
