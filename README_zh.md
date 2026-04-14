# ClaudeGo

一款基于 Go 语言开发的 CLI AI 编程助手，通过流式 REPL 界面与 LLM（大语言模型）交互，完成编码任务。
![imgs](./assets/image.png)

## 功能特性

- **交互式 REPL** — 支持行编辑、历史记录（`liner`）、ANSI 彩色输出、Ctrl+C 中断及对话回滚
- **LLM 流式响应** — 兼容 OpenAI API 格式，支持流式对话补全和工具调用
- **计划模式** — 复杂任务自动分解为多步骤计划：创建 → 执行 → 暂停/恢复 → 完成
- **内置工具** — `bash`（含危险命令检测）和 `file_handler` 文件处理工具
- **对话状态管理** — 检查点/回滚机制保证中断后对话历史完整
- **计划持久化** — 计划以 JSON 格式保存至 `~/.claudego/plans/`
- **日志轮转** — 日志文件存储在 `~/.claudego/logs/`，保留 7 天

## 架构概览

```
cmd/claudego/main.go     — REPL 入口，信号处理，命令路由
internal/loop/agent.go    — Agent 循环：LLM 流式响应，工具执行
internal/plan/            — 计划模式（创建、执行、持久化）
internal/tools/           — 工具注册表、内置工具
internal/config/         — 配置文件加载器
pkg/llm/                 — LLM 客户端，封装 OpenAI SDK
pkg/conversation/        — 对话状态管理
pkg/ui/                  — 命令行样式、Markdown 渲染
pkg/logger/              — 单例日志记录器
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

### 自动检测

ClaudeGo 会自动识别复杂任务（重构、迁移、实现、构建等）并切换至计划模式。

### 计划模式

进入计划模式后：
1. Agent 分析目标并生成分步计划
2. 步骤展示并保存至 `~/.claudego/plans/`
3. 每个步骤依次执行，拥有完整的 LLM + 工具能力
4. 按 `Ctrl+C` 可中断执行 — 对话回滚至上一个检查点
5. 可通过保存的计划 ID 恢复中断的计划

### 内置工具

**bash** — 执行 Shell 命令
- 危险命令会被拦截：`rm -rf /`、`sudo`、`shutdown`、fork 炸弹、远程脚本注入等

**file_handler** — 文件的读取、写入和编辑

### 对话回滚

在任何 LLM 调用过程中按 `Ctrl+C` 即可中断。当前查询前的对话状态将回滚至检查点，确保对话历史完整性。

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
│   ├── conversation/          # 对话状态
│   ├── ui/                    # 命令行样式
│   └── logger/                # 日志记录
└── utils/                     # 工具函数
```

## 依赖

- [openai-go](https://github.com/openai/openai-go) — LLM API 客户端
- [liner](https://github.com/peterh/liner) — REPL 行编辑
- [logrus](https://github.com/sirupsen/logrus) — 日志库
- [go-playground/validator](https://github.com/go-playground/validator) — 输入验证

## 开源协议

MIT
