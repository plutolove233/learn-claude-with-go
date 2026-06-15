结论：ClaudeGo 现在还不能正面替代 Claude Code / Codex。它更像一个“可控、本地、Go 实现的 coding agent 原型”。要吸引竞品用户迁移，不能先拼模型能力，而要先
  把“可靠完成任务 + 可审计安全 + 可扩展生态 + 低迁移成本”做成产品闭环。

  当前定位
  ClaudeGo 已有几个不错的底座：

  - OpenAI-compatible 流式 LLM + tool calling：pkg/llm/client.go:35
  - 本地 REPL、历史、Ctrl+C 回滚：cmd/claudego/main.go:44
  - bash / file 工具和基础沙箱：internal/tools/bash.go:48、internal/tools/file.go:73
  - 权限 allow/ask/deny 与 session allow：pkg/permissions/permissions.go:69
  - hooks：SessionStart / PreToolUse / PostToolUse：pkg/hooks/runner.go:16
  - skill loader：pkg/skill/loader.go:23
  - context compaction：pkg/compaction/compactor.go:43

  但当前短板也很明确：README 提到的 internal/plan、pkg/graph 实际仓库里不存在，/plan 分支目前基本是空实现：cmd/claudego/main.go:104。这会直接影响用户信任，
  必须先修正文档或补齐功能。

  横向对比

   维度            ClaudeGo 当前                                                   Claude Code                          Codex
  ━━━━━━━━━━━━━━  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   核心体验        本地 REPL + 工具调用                                            终端、IDE、Desktop、Web 多入口；     CLI、IDE、Web/mobile、CI/CD SDK 多
                                                                                   能读代码、改文件、跑命令             入口
  ──────────────  ──────────────────────────────────────────────────────────────  ───────────────────────────────────  ────────────────────────────────────
   任务完成闭环    agent loop 基础可用，但缺少真正 plan/verify/commit/PR 闭环      支持规划、跨文件修改、验证、git、    支持云端任务、并行任务、沙箱、IDE/
                                                                                   PR、CI、Slack、定时任务              CLI/SDK、GitHub/Slack/Linear 等
  ──────────────  ──────────────────────────────────────────────────────────────  ───────────────────────────────────  ────────────────────────────────────
   安全            本地权限规则、危险命令 regex、cwd 文件沙箱                      权限模式、hooks、MCP、企业/多平台    sandbox、approvals、MCP、企业安
                                                                                   配置                                 全、云端隔离
  ──────────────  ──────────────────────────────────────────────────────────────  ───────────────────────────────────  ────────────────────────────────────
   生态扩展        Markdown skills、hooks；暂无 MCP                                CLAUDE.md、skills、hooks、MCP、子    AGENTS.md、MCP、skills、
                                                                                   代理、Agent SDK                      subagents、plugins、SDK
  ──────────────  ──────────────────────────────────────────────────────────────  ───────────────────────────────────  ────────────────────────────────────
   差异化机会      Go 单二进制、本地可控、国产/多模型 OpenAI-compatible、可审计    强模型与成熟产品体验                 云端并行、ChatGPT 生态、CI/CD、浏
                                                                                                                        览器/电脑操作

  官方口径也说明竞品已经是“多入口 + 多工具 + 自动化平台”：Claude Code 文档明确覆盖
  Terminal、IDE、Desktop、Web，并强调读代码、改文件、跑命令、MCP、hooks、skills、agent teams、定时任务等能力；Codex 官方文档也强调
  IDE、CLI、Web/mobile、CI/CD SDK 多入口，以及 MCP 和 shell sandbox 能力。(docs.anthropic.com (https://docs.anthropic.com/en/docs/claude-code/overview))
  (platform.openai.com (https://platform.openai.com/docs/guides/code-generation)) (platform.openai.com (https://platform.openai.com/docs/docs-mcp))
  (platform.openai.com (https://platform.openai.com/docs/guides/tools-shell))

  最该优化的 8 个方向

  1. 先补“可信完成任务”的闭环
     做真正的 plan mode：生成计划、展示、执行、失败重试、保存、恢复、验收。每次任务结束自动给出：改了什么、跑了什么测试、剩余风险。没有这个，Claude Code/
     Codex 用户不会迁移。

  2. 把工具层升级为工程级编辑能力
     现在 file_handler 只有 read/write，没有 patch、diff、search、glob、multi-edit、undo。应优先补：read_file、write_file、edit_file、apply_patch、grep、
     glob、git_diff。用户最在意的是“改得准、可审查、少误伤”。

  3. 支持 AGENTS.md / CLAUDE.md / 项目记忆
     Claude/Codex 用户已经习惯项目级规则。ClaudeGo 应自动读取 AGENTS.md、CLAUDE.md、.claudego/memory.md，并支持 SessionStart 注入。这样迁移成本最低。

  4. 接入 MCP
     这是生态关键点。Claude Code 和 Codex 都把 MCP 作为连接外部工具/文档/数据源的主路径。ClaudeGo 如果没有 MCP，只能靠自定义 Go tool，生态扩展会慢很多。

  5. 做“安全可审计”作为差异化卖点
     不要只用 regex 拦危险命令。建议升级为：命令分段解析、工作区写入边界、网络开关、敏感文件保护、审计日志、企业 policy 文件。ClaudeGo 可以主打“本地可控、
     每个工具调用都有证据”。

  6. 补 git / PR 工作流
     竞品用户常用场景是“修 bug、跑测试、提交 PR”。ClaudeGo 应内置：查看 diff、生成提交信息、遵循 Lore trailer、创建分支、PR 摘要、测试证据。没有 git 闭环，
     用户会觉得只是聊天工具。

  7. 多模型策略要成为优势
     ClaudeGo 用 OpenAI-compatible API，这是差异化机会。做 model profiles：DeepSeek、Qwen、OpenAI、Anthropic proxy、本地模型；再加任务路由：快模型搜索、强
     模型改代码、便宜模型总结。吸引点是“可控成本 + 不绑供应商”。

  8. 补“并行/子代理/后台任务”
     Codex 已有并行候选和云端任务，Claude Code 有 agent teams/background agents。ClaudeGo 可以先做轻量版：explore agent、test agent、review agent，本地
     goroutine 并行执行，最终由主 agent 合并。

  优先级路线
  第一阶段，2-4 周：修文档真实性；补 edit/apply_patch/grep/glob/git_diff；实现真实 /plan；自动读取 AGENTS/CLAUDE；任务结束自动测试和总结。

  第二阶段，1-2 个月：MCP client；git/commit/PR 工作流；更强权限策略；会话恢复和项目 memory；模型 profile。

  第三阶段，3 个月以上：IDE 插件、后台任务、并行子代理、云/容器沙箱、团队共享 skills marketplace。

  最有机会的定位
  不要宣传成“Claude Code/Codex 的全面替代品”。更好的切入点是：

  > ClaudeGo：一个本地优先、可审计、供应商无关的 Go coding agent，兼容 AGENTS/CLAUDE 规则、MCP、skills 和企业权限策略。

  验证：我检查了当前代码结构与关键实现，并运行了 go test ./...，测试全部通过；本次未修改文件。