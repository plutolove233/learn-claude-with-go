# Skill System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 ClaudeGo 增加 Skill 功能：用户通过 slash 命令（如 `/review`、`/tdd`）触发预设工作流指令，支持自动补全。

**Architecture:** Skill 本质是可复用的指令片段，从 `~/.claude/go/skills/*.md` 加载，解析后注册为 slash 命令。执行时将 instructions 注入 system prompt 调用 LLM。

**Tech Stack:** Go, liner (REPL), gopkg.in/yaml.v3, openai-go

---

## File Structure

```
pkg/skill/
  types.go           # Skill struct
  loader.go          # ParseMarkdown / LoadFromDir
  skill_registry.go  # global registry + Register/Get/List/Completions
  executor.go        # Execute skill with LLM
  handler.go         # MatchAndExecute slash command
pkg/skill/skill_registry_test.go
pkg/skill/loader_test.go
cmd/claudego/main.go   # integrate into REPL loop
```

---

### Task 1: Create `pkg/skill/types.go`

**Files:**
- Create: `pkg/skill/types.go`

- [ ] **Step 1: Create types.go**

```go
package skill

import "claudego/pkg/types"

// Skill represents a slash command skill loaded from a markdown file.
type Skill struct {
    Name        string // slash command name (e.g., "review")
    Description string // short description shown in help
    Path        string // source file path
    Instructions string // markdown body: injected as system prompt fragment
}
```

- [ ] **Step 2: Commit**

```bash
git add pkg/skill/types.go
git commit -m "feat: add Skill type for slash command system"
```

---

### Task 2: Create `pkg/skill/loader.go`

**Files:**
- Create: `pkg/skill/loader.go`

- [ ] **Step 1: Create loader.go**

```go
package skill

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "gopkg.in/yaml.v3"
)

// Loader loads skill templates from markdown files with frontmatter.
type Loader struct {
    skillsDir string
}

// NewLoader creates a new skill loader for the specified directory.
func NewLoader(skillsDir string) *Loader {
    return &Loader{skillsDir: skillsDir}
}

// LoadSkills loads all skills from the configured directory.
func (l *Loader) LoadSkills() ([]*Skill, error) {
    entries, err := os.ReadDir(l.skillsDir)
    if err != nil {
        return nil, fmt.Errorf("failed to read skills directory: %w", err)
    }

    var skills []*Skill
    for _, entry := range entries {
        if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
            continue
        }

        path := filepath.Join(l.skillsDir, entry.Name())
        s, err := l.LoadFromFile(path)
        if err != nil {
            // Skip invalid files
            continue
        }
        skills = append(skills, s)
    }

    return skills, nil
}

// LoadFromFile loads a single skill from a markdown file.
func (l *Loader) LoadFromFile(path string) (*Skill, error) {
    content, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read file: %w", err)
    }

    return ParseMarkdown(string(content), path)
}

// ParseMarkdown parses a markdown string with frontmatter into a Skill.
func ParseMarkdown(markdown string, path string) (*Skill, error) {
    content := strings.TrimPrefix(markdown, "---\n")
    if content == markdown {
        return nil, fmt.Errorf("no frontmatter found in %s", path)
    }

    parts := strings.SplitN(content, "---\n", 2)
    if len(parts) != 2 {
        return nil, fmt.Errorf("invalid frontmatter format in %s", path)
    }

    frontmatter := parts[0]
    skillContent := parts[1]

    var fm yamlFrontmatter
    if err := yaml.Unmarshal([]byte(frontmatter), &fm); err != nil {
        return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
    }

    if fm.Name == "" {
        return nil, fmt.Errorf("skill name is required in %s", path)
    }

    return &Skill{
        Path:         path,
        Name:         fm.Name,
        Description:  fm.Description,
        Instructions: strings.TrimSpace(skillContent),
    }, nil
}

// yamlFrontmatter represents the YAML frontmatter structure.
type yamlFrontmatter struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description"`
}
```

- [ ] **Step 2: Commit**

```bash
git add pkg/skill/loader.go
git commit -m "feat: add skill loader for markdown frontmatter parsing"
```

---

### Task 3: Create `pkg/skill/skill_registry.go`

**Files:**
- Create: `pkg/skill/skill_registry.go`

- [ ] **Step 1: Create skill_registry.go**

```go
package skill

import (
    "fmt"
    "path/filepath"
    "strings"
    "sync"
)

// Registry holds all registered skills.
type Registry struct {
    mu     sync.RWMutex
    skills map[string]*Skill // skill name -> Skill
}

// defaultSkillRegistry is the global skill registry.
var defaultSkillRegistry = &Registry{
    skills: make(map[string]*Skill),
}

// NewRegistry creates a new registry.
func NewRegistry() *Registry {
    return &Registry{
        skills: make(map[string]*Skill),
    }
}

// DefaultRegistry returns the global skill registry.
func DefaultRegistry() *Registry {
    return defaultSkillRegistry
}

// Register adds a skill to the registry.
func (r *Registry) Register(s *Skill) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if s.Name == "" {
        return fmt.Errorf("skill name cannot be empty")
    }
    if _, exists := r.skills[s.Name]; exists {
        return fmt.Errorf("skill %q is already registered", s.Name)
    }
    r.skills[s.Name] = s
    return nil
}

// Get returns a skill by name, or nil if not found.
func (r *Registry) Get(name string) (*Skill, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    s, ok := r.skills[name]
    return s, ok
}

// List returns all registered skills.
func (r *Registry) List() []*Skill {
    r.mu.RLock()
    defer r.mu.RUnlock()
    skills := make([]*Skill, 0, len(r.skills))
    for _, s := range r.skills {
        skills = append(skills, s)
    }
    return skills
}

// LoadFromDir loads all skills from a directory and registers them.
func (r *Registry) LoadFromDir(dir string) error {
    loader := NewLoader(dir)
    skills, err := loader.LoadSkills()
    if err != nil {
        return err
    }

    for _, s := range skills {
        if err := r.Register(s); err != nil {
            // Skip duplicate skills (use first loaded)
            continue
        }
    }
    return nil
}

// LoadAndRegister is a convenience method that calls LoadFromDir on the default registry.
func LoadAndRegister(dir string) error {
    return defaultSkillRegistry.LoadFromDir(dir)
}

// Completions returns all skill names that start with the given prefix,
// each appended with a space for liner completion.
func (r *Registry) Completions(prefix string) []string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    var completions []string
    for name := range r.skills {
        if strings.HasPrefix(name, prefix) {
            completions = append(completions, name+" ")
        }
    }
    return completions
}
```

- [ ] **Step 2: Commit**

```bash
git add pkg/skill/skill_registry.go
git commit -m "feat: add skill registry with Register/Get/List/Completions"
```

---

### Task 4: Create `pkg/skill/executor.go`

**Files:**
- Create: `pkg/skill/executor.go`

- [ ] **Step 1: Create executor.go**

```go
package skill

import (
    "context"
    "fmt"

    "claudego/pkg/interfaces"
    "claudego/pkg/llm"
    "claudego/pkg/types"
)

// Execute runs a skill with the given args and LLM client.
func Execute(ctx context.Context, s *Skill, args string, llmClient *llm.Client, registry interfaces.ToolRegistry) error {
    // Build system prompt: skill instructions + user args as context
    system := s.Instructions
    if args != "" {
        system = system + "\n\nUser request: " + args
    }

    // Build initial user message with the skill context
    messages := []types.Message{
        {
            Role:    "user",
            Content: "Please help with the task described above.",
        },
    }

    // Execute with LLM
    result, err := llmClient.Complete(ctx, messages, system, registry)
    if err != nil {
        return fmt.Errorf("skill execution failed: %w", err)
    }

    // Output the result
    if result.Content != "" {
        fmt.Println(result.Content)
    }

    return nil
}
```

- [ ] **Step 2: Commit**

```bash
git add pkg/skill/executor.go
git commit -m "feat: add skill executor with LLM integration"
```

---

### Task 5: Create `pkg/skill/handler.go`

**Files:**
- Create: `pkg/skill/handler.go`

- [ ] **Step 1: Create handler.go**

```go
package skill

import (
    "context"
    "fmt"
    "strings"

    "claudego/pkg/interfaces"
    "claudego/pkg/llm"
    "claudego/pkg/ui"
)

// MatchAndExecute checks if input is a slash command and executes it if found.
// Returns (matched, error):
//   - (true, nil) if a skill was matched and executed successfully
//   - (true, error) if a skill was matched but execution failed
//   - (false, nil) if input is not a slash command
func MatchAndExecute(ctx context.Context, input string, registry *Registry, llmClient *llm.Client, toolRegistry interfaces.ToolRegistry) (bool, error) {
    if !strings.HasPrefix(input, "/") {
        return false, nil
    }

    // Parse slash command: /skill-name [args]
    parts := strings.SplitN(strings.TrimPrefix(input, "/"), " ", 2)
    skillName := parts[0]
    args := ""
    if len(parts) > 1 {
        args = parts[1]
    }

    // Look up skill
    s, ok := registry.Get(skillName)
    if !ok {
        return false, nil // Not a skill, let caller handle as regular command
    }

    // Execute skill
    ui.Info("Executing skill: %s", s.Name)
    if err := Execute(ctx, s, args, llmClient, toolRegistry); err != nil {
        return true, err
    }

    return true, nil
}
```

- [ ] **Step 2: Commit**

```bash
git add pkg/skill/handler.go
git commit -m "feat: add skill handler for slash command matching"
```

---

### Task 6: Write `pkg/skill/loader_test.go`

**Files:**
- Create: `pkg/skill/loader_test.go`

- [ ] **Step 1: Write loader tests**

```go
package skill

import (
    "os"
    "path/filepath"
    "testing"
)

func TestParseMarkdown(t *testing.T) {
    markdown := `---
name: review
description: Review code changes for issues
---
You are a code reviewer. Analyze the git diff.`

    s, err := ParseMarkdown(markdown, "review.md")
    if err != nil {
        t.Fatalf("ParseMarkdown failed: %v", err)
    }

    if s.Name != "review" {
        t.Errorf("expected name 'review', got '%s'", s.Name)
    }
    if s.Description != "Review code changes for issues" {
        t.Errorf("unexpected description: %s", s.Description)
    }
    if s.Instructions != "You are a code reviewer. Analyze the git diff." {
        t.Errorf("unexpected instructions: %s", s.Instructions)
    }
}

func TestParseMarkdownNoFrontmatter(t *testing.T) {
    markdown := `No frontmatter here`
    _, err := ParseMarkdown(markdown, "test.md")
    if err == nil {
        t.Error("expected error for missing frontmatter")
    }
}

func TestParseMarkdownNoName(t *testing.T) {
    markdown := `---
description: No name field
---
Content`
    _, err := ParseMarkdown(markdown, "test.md")
    if err == nil {
        t.Error("expected error for missing name")
    }
}

func TestLoader(t *testing.T) {
    tmpDir := t.TempDir()

    skill1 := `---
name: skill1
description: First test skill
---
Skill 1 content`
    skill2 := `---
name: skill2
description: Second test skill
---
Skill 2 content`

    if err := os.WriteFile(filepath.Join(tmpDir, "skill1.md"), []byte(skill1), 0644); err != nil {
        t.Fatal(err)
    }
    if err := os.WriteFile(filepath.Join(tmpDir, "skill2.md"), []byte(skill2), 0644); err != nil {
        t.Fatal(err)
    }

    loader := NewLoader(tmpDir)
    skills, err := loader.LoadSkills()
    if err != nil {
        t.Fatalf("LoadSkills failed: %v", err)
    }

    if len(skills) != 2 {
        t.Errorf("expected 2 skills, got %d", len(skills))
    }
}
```

- [ ] **Step 2: Run tests**

```bash
go test ./pkg/skill/... -v
```

- [ ] **Step 3: Commit**

```bash
git add pkg/skill/loader_test.go
git commit -m "test: add skill loader tests"
```

---

### Task 7: Write `pkg/skill/skill_registry_test.go`

**Files:**
- Create: `pkg/skill/skill_registry_test.go`

- [ ] **Step 1: Write registry tests**

```go
package skill

import (
    "os"
    "path/filepath"
    "testing"
)

func TestRegistryRegister(t *testing.T) {
    r := NewRegistry()
    s := &Skill{Name: "test", Description: "Test skill"}

    if err := r.Register(s); err != nil {
        t.Fatalf("Register failed: %v", err)
    }

    if _, ok := r.Get("test"); !ok {
        t.Error("Get failed to retrieve registered skill")
    }
}

func TestRegistryDuplicate(t *testing.T) {
    r := NewRegistry()
    s := &Skill{Name: "test", Description: "Test skill"}

    r.Register(s)
    err := r.Register(s)
    if err == nil {
        t.Error("expected error for duplicate registration")
    }
}

func TestRegistryCompletions(t *testing.T) {
    r := NewRegistry()
    r.Register(&Skill{Name: "review"})
    r.Register(&Skill{Name: "refactor"})
    r.Register(&Skill{Name: "tdd"})

    completions := r.Completions("re")
    if len(completions) != 2 {
        t.Errorf("expected 2 completions, got %d", len(completions))
    }
}

func TestRegistryLoadFromDir(t *testing.T) {
    tmpDir := t.TempDir()

    skill1 := `---
name: skill1
description: First skill
---
Content 1`

    if err := os.WriteFile(filepath.Join(tmpDir, "skill1.md"), []byte(skill1), 0644); err != nil {
        t.Fatal(err)
    }

    r := NewRegistry()
    if err := r.LoadFromDir(tmpDir); err != nil {
        t.Fatalf("LoadFromDir failed: %v", err)
    }

    if _, ok := r.Get("skill1"); !ok {
        t.Error("skill1 not loaded from directory")
    }
}
```

- [ ] **Step 2: Run tests**

```bash
go test ./pkg/skill/... -v
```

- [ ] **Step 3: Commit**

```bash
git add pkg/skill/skill_registry_test.go
git commit -m "test: add skill registry tests"
```

---

### Task 8: Integrate into REPL (`cmd/claudego/main.go`)

**Files:**
- Modify: `cmd/claudego/main.go`

- [ ] **Step 1: Read current main.go to understand existing imports and structure**

The current main.go has:
- `tools.RegisterDefaults()` at line 33
- `registry := tools.GetRegistry()` at line 34
- `agent := loop.New(cfg, log, registry)` at line 37

Need to:
1. Add skill import
2. Load skills after tools are registered
3. Add completer before the REPL loop
4. Handle skill commands in the input loop

- [ ] **Step 2: Modify main.go - add skill import**

Add after existing imports:
```go
"claudego/pkg/skill"
```

- [ ] **Step 3: Modify main.go - load skills after tool registration**

After `tools.RegisterDefaults()` and `registry := tools.GetRegistry()`, add:
```go
// Load skills from ~/.claudego/skills/
skillRegistry := skill.NewRegistry()
if err := skillRegistry.LoadFromDir("~/.claudego/skills/"); err != nil {
    // Skills are optional - log warning but don't fail startup
    log.Warning("Failed to load skills: %v", err)
}
```

- [ ] **Step 4: Modify main.go - add completer after line.SetCtrlCAborts(true)**

Add after `line.SetCtrlCAborts(true)`:
```go
// Configure skill auto-completion for liner
line.SetCompleter(func(line string) []string {
    if len(line) > 0 && line[0] == '/' {
        return skillRegistry.Completions(line[1:])
    }
    return nil
})
```

- [ ] **Step 5: Modify main.go - add skill handling in REPL loop**

After the `if query == "" || query == "q" || query == "exit"` check and before `isComplexTask`, add:
```go
// Check for skill slash commands
if matched, err := skill.MatchAndExecute(ctx, query, skillRegistry, agent.LLMClient(), registry); matched {
    if err != nil {
        fmt.Fprintf(os.Stderr, "Skill error: %v\n", err)
    }
    stopListener()
    cancel()
    fmt.Println()
    continue
}
```

- [ ] **Step 6: Verify agent.LLMClient exists or add accessor**

If `agent.LLMClient()` doesn't exist, add a method to Agent:
```go
// cmd/claudego is in a different package, so we need an accessor
// In internal/loop/agent.go, add:
func (a *Agent) LLMClient() *llm.Client {
    return a.llmClient
}
```

- [ ] **Step 7: Run build to verify**

```bash
go build ./cmd/claudego
```

- [ ] **Step 8: Commit**

```bash
git add cmd/claudego/main.go internal/loop/agent.go
git commit -m "feat: integrate skill system into REPL loop with auto-completion"
```

---

### Task 9: Create sample skill files in `~/.claudego/skills/`

**Files:**
- Create: `~/.claudego/skills/review.md`
- Create: `~/.claudego/skills/tdd.md`

- [ ] **Step 1: Create sample skills directory**

```bash
mkdir -p ~/.claudego/skills
```

- [ ] **Step 2: Create review.md**

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

- [ ] **Step 3: Create tdd.md**

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

---

## Self-Review Checklist

1. **Spec coverage:** All sections of the design are implemented
2. **No placeholders:** All code is complete, no TBD/TODO
3. **Type consistency:** Skill struct fields match across all files
4. **Import cycle check:** pkg/skill only imports pkg/interfaces and pkg/llm (no circular deps)

## Execution Options

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
