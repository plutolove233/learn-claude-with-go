# Skill System Design

## Summary

为 ClaudeGo 增加 Skill 功能：用户通过 slash 命令（如 `/review`、`/tdd`）触发预设的工作流指令，提升 REPL 交互效率。

## Goals

- Skill 文件存放在 `~/.claudego/skills/*.md`，采用 Markdown + YAML frontmatter 格式
- 解析后注册为 slash 命令，用户输入 `/skill-name` 时触发
- Skill 本质是一段可复用的指令片段，执行时注入到 system prompt
- Skill 执行失败时打印错误并退出当前 skill 流程

## Non-Goals

- 不做 skill 市场或外部扩展加载（后续可扩展）
- 不改变现有 agent 模板机制（`pkg/agent/` 保持独立）
- 不支持 skill 嵌套调用

## File Format

```markdown
---
name: review
description: Review code changes for issues
---
You are a code reviewer. Analyze the git diff and provide feedback on:
1. Logic errors
2. Security vulnerabilities
3. Performance issues
4. Code style violations
```

## Data Model

```go
// pkg/skill/types.go
type Skill struct {
    Name        string // slash command name (e.g., "review")
    Description string // short description shown in help
    Path        string // source file path
    Instructions string // markdown body: injected as system prompt fragment
}
```

## Components

### 1. Loader (`pkg/skill/loader.go`)

```
~/.claudego/skills/*.md
  → ParseMarkdown(name, description, instructions)
  → []*Skill
```

- 扫描 `~/.claudego/skills/` 目录
- 跳过无 frontmatter 或无 name 字段的文件
- 文件名即 skill 名（去掉 `.md` 后缀）

### 2. Registry (`pkg/skill/registry.go`)

- 全局 `defaultRegistry = map[string]*Skill`
- `Register(skill)` / `Get(name)` / `List()` / `LoadFromDir(path)` 方法
- `LoadAndRegister()` 封装：扫描目录 + 注册

### 3. Command Handler (`pkg/skill/handler.go`)

```go
// MatchAndExecute 检查 input 是否为 slash 命令若是则执
func MatchAndExecute(ctx context.Context, input string, registry *Registry) (bool, error)
// 成功执行返回 (true, nil)
// 非 slash 命令返回 (false, nil)
// 执行失败返回 (true, error)
```

- 输入形如 `/review some args` → 解析为 skill 名 `review` + 参数 `some args`
- 参数透传给 skill 执行逻辑

### 4. Skill Executor (`pkg/skill/executor.go`)

```go
func Execute(ctx context.Context, skill *Skill, args string, llmClient *llm.Client, toolRegistry interfaces.ToolRegistry) error
```

- 将 skill.Instructions 注入 system prompt
- 调用 llmClient.Complete() 执行对话
- 完成后打印结果

## Integration Points

### REPL 层 (`cmd/claudego/main.go`)

```go
// 初始化时加载所有 skill
skillRegistry := skill.NewRegistry()
skillRegistry.LoadAndRegister("~/.claudego/skills/")

// 处理用户输入时优先检查 slash 命令
if matched, err := skill.MatchAndExecute(ctx, input, skillRegistry); matched {
    if err != nil {
        // 打印错误信息
    }
    continue // 回到 REPL 循环
}
// 非 skill 命令 → 走原有对话逻辑
```

## Directory Structure

```
pkg/
  └── skill/
        types.go       # Skill struct
        loader.go      # LoadFromFile / LoadAgents
        registry.go    # global registry + registration
        executor.go    # Execute skill with LLM
        handler.go     # MatchAndExecute slash command
cmd/claudego/main.go   # integrate into REPL loop
```

## Error Handling

- Skill 执行失败时，打印错误到 stderr，skill 流程终止
- 用户回到普通 REPL 交互，可重新输入命令
- 加载阶段（启动时）的错误 panic，避免带病启动

## Testing Strategy

- `loader_test.go`: 测试 frontmatter 解析、错误文件跳过
- `registry_test.go`: 测试注册、去重、覆盖
- `handler_test.go`: 测试 slash 命令匹配与参数解析

## Example Skill Files

`~/.claudego/skills/review.md`:
```markdown
---
name: review
description: Review code changes for issues
---
You are a code reviewer. Analyze the git diff and provide feedback on:
1. Logic errors
2. Security vulnerabilities
3. Performance issues
4. Code style violations
```

`~/.claudego/skills/tdd.md`:
```markdown
---
name: tdd
description: Test-driven development workflow
---
You are following TDD methodology:
1. Write a failing test first
2. Write the minimal code to pass the test
3. Refactor as needed
4. Repeat until feature is complete
```
