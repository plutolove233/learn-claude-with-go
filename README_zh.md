# ClaudeGo

一款使用 Go 语言开发的交互式 AI 编程助手，通过流式 REPL 界面与 LLM（大语言模型）交互完成编码任务。

![imgs](./assets/image.png)

## 功能特性

- **交互式 REPL** — 基于 `liner` 库实现完整行编辑、历史浏览（上下箭头）、ANSI 彩色输出，支持 Ctrl+C 中断当前 LLM 调用并回滚对话状态
- **LLM 流式响应** — 通过 `openai-go` SDK 封装 OpenAI API 兼容接口，支持流式文本补全与函数工具调用（Function Calling）
- **计划模式（Plan Mode）** — 自动识别复杂任务（重构、迁移、多文件实现等），将任务分解为多步骤计划并逐一执行；计划以 JSON 格式持久化至 `~/.claudego/plans/`，支持通过 plan ID 恢复中断的任务
- **内置工具** — 插件化的 `ToolRegistry` 架构，内置 `bash` 工具（含危险命令检测，拦截 `rm -rf /`、fork 炸弹等）和 `file_handler` 工具（文件读取/写入/编辑），支持扩展自定义工具
- **图工作流运行时** — 新增 `pkg/graph` 包，提供类 LangGraph 的共享状态、顺序边、条件路由、循环保护、执行轨迹，以及基于现有 LLM 接口的节点适配能力
- **对话状态管理** — 基于检查点（checkpoint）的对话回滚机制，确保中断后对话历史完整性
- **日志轮转** — 基于 `logrus` 的结构化日志，文件日志按天轮转，保留 7 天

## 架构概览

```
cmd/claudego/main.go        — REPL 入口，信号处理，命令路由
internal/loop/agent.go      — Agent 循环：LLM 流式响应，工具执行
internal/plan/              — 计划模式（创建、执行、持久化）
  ├── plan.go              — 计划数据结构
  ├── planner.go           — 计划创建器
  └── executor.go          — 计划执行器
internal/tools/            — 工具注册表 + 内置工具
  ├── registry.go          — 工具注册表
  ├── base_tool.go         — 工具基类接口
  ├── bash.go              — bash 工具（含危险命令检测）
  ├── file.go              — file_handler 工具
  └── task.go              — 任务工具
internal/config/           — 配置文件加载器
pkg/llm/                   — LLM 客户端，封装 openai-go SDK
pkg/conversation/          — 对话状态管理，检查点/回滚机制
pkg/ui/                    — 命令行样式、Markdown 渲染、流式输出
pkg/logger/                — 单例日志记录器（logrus）
pkg/graph/                 — 面向 supervisor / expert agent 编排的图工作流运行时
pkg/skill/                 — Skill 扩展系统
  ├── skill_registry.go    — Skill 注册表
  ├── loader.go            — Markdown 文件加载器（支持文件夹结构）
  ├── executor.go          — Skill 执行器（LLM 集成）
  └── handler.go           — Skill 匹配与执行入口
pkg/types/                 — 共享类型定义
pkg/interfaces/            — 核心接口定义（ToolInterface, LLMInterface）
```

## 安装

```bash
git clone https://github.com/yizhigopher/learn-claude-with-go.git
cd learn-claude-with-go
go build -o claudego ./cmd/claudego
```

## 配置

创建 `~/.claudego/config.json`：

```json
{
  "api_key": "your-api-key",
  "base_url": "https://api.deepseek.com/v1",
  "model": "deepseek-chat"
}
```

- `api_key` — LLM 服务商 API 密钥
- `base_url` — OpenAI 兼容 API 端点
- `model` — 模型名称（如 `deepseek-chat`、`gpt-4o`）

## 使用方法

```bash
./claudego
```

### REPL 命令

| 命令 | 说明 |
|------|------|
| `q` / `exit` | 退出会话 |
| `/plan <目标>` | 强制为指定目标启用计划模式 |
| `/skill <名称> [参数]` | 执行指定 skill 扩展 |

### 自动检测

ClaudeGo 会自动识别复杂任务（重构、迁移、实现、构建等）并切换至计划模式。

### 计划模式

进入计划模式后：

1. Agent 分析目标并生成分步计划
2. 步骤展示并保存至 `~/.claudego/plans/`
3. 每个步骤依次执行，拥有完整的 LLM + 工具能力
4. 按 `Ctrl+C` 可中断执行 — 对话回滚至上一个检查点
5. 可通过保存的计划 ID 恢复中断的计划

![plan1](./assets/plan-mode.png)
![plan2](./assets/plan-mode-2.png)

### Skill 扩展系统

ClaudeGo 支持通过 Markdown 文件定义自定义 Skill，存放在 `~/.claudego/skills/` 目录。

**Skill 文件格式：**

```markdown
---
name: skill-name
description: Skill 功能描述
---

# Skill Name

[LLM 提示词内容]
```

加载后可通过 `/skill skill-name` 或在对话中触发自动补全执行。

### 内置工具

**bash** — 执行 Shell 命令
- 危险命令会被拦截：`rm -rf /`、`sudo`、`shutdown`、fork 炸弹、远程脚本注入等

**file_handler** — 文件的读取、写入和编辑

### 对话回滚

在任何 LLM 调用过程中按 `Ctrl+C` 即可中断。当前查询前的对话状态将回滚至检查点，确保对话历史完整性。

### 图工作流运行时

`pkg/graph` 是一层轻量级编排运行时，适合在现有 `LLMClient` 和 `ToolRegistry` 基础上构建领域专业智能体。

核心概念：

- `graph.State`：图中所有节点共享的状态
- `graph.AddEdge`：确定性的顺序流转
- `graph.AddConditionalEdges`：supervisor 风格的条件分发
- `graph.Command{Goto: ...}`：节点运行后动态 handoff
- `graph.NewLLMNode(...)`：把现有 LLM 能力封装成图节点

最小示例：

```go
g := graph.New(graph.WithMaxSteps(16))

_ = g.AddNode("supervisor", func(ctx context.Context, state *graph.State) (*graph.Command, error) {
	domain, _ := graph.Value[string](state, "domain")
	return &graph.Command{Goto: domain}, nil
})
_ = g.AddNode("legal", func(ctx context.Context, state *graph.State) (*graph.Command, error) {
	state.Set("answer", "由法律专家节点处理")
	return nil, nil
})
_ = g.AddNode("medical", func(ctx context.Context, state *graph.State) (*graph.Command, error) {
	state.Set("answer", "由医疗专家节点处理")
	return nil, nil
})

_ = g.SetEntryPoint("supervisor")
_ = g.AddEdge("legal", graph.End)
_ = g.AddEdge("medical", graph.End)

result, err := g.Run(ctx, map[string]any{"domain": "legal"})
```

也可以直接把现有 LLM 抽象包装成节点：

```go
node, _ := graph.NewLLMNode(graph.LLMNodeConfig{
	Client:       llmClient,
	Registry:     toolRegistry,
	SystemPrompt: "你是法律合规专家。",
	BuildUserPrompt: func(state *graph.State) string {
		task, _ := graph.Value[string](state, "task")
		return task
	},
	ResponseKey: "specialist_output",
})
```

## 依赖

- [openai-go](https://github.com/openai/openai-go) — LLM API 客户端
- [liner](https://github.com/peterh/liner) — REPL 行编辑
- [logrus](https://github.com/sirupsen/logrus) — 日志库
- [go-playground/validator](https://github.com/go-playground/validator) — 输入验证

## 项目结构

```
claudego/
├── cmd/claudego/main.go       # 应用入口
├── internal/
│   ├── loop/agent.go          # Agent 循环
│   ├── plan/                  # 计划模式（创建、执行、持久化）
│   ├── tools/                 # 工具注册表 + 内置工具
│   └── config/                # 配置加载
├── pkg/
│   ├── llm/                   # LLM 客户端
│   ├── graph/                 # 图工作流运行时
│   ├── conversation/          # 对话状态
│   ├── ui/                    # 命令行样式
│   ├── logger/                # 日志记录
│   ├── skill/                 # Skill 扩展系统
│   ├── types/                 # 共享类型
│   └── interfaces/            # 核心接口定义
└── utils/                     # 工具函数
```

## 依赖

- [openai-go](https://github.com/openai/openai-go) — LLM API 客户端
- [liner](https://github.com/peterh/liner) — REPL 行编辑
- [logrus](https://github.com/sirupsen/logrus) — 日志库
- [go-playground/validator](https://github.com/go-playground/validator) — 输入验证

## 开源协议

MIT
