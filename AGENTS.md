# Repository Guidelines

## Project Structure & Module Organization

This repository contains `claudego`, a Go CLI AI coding agent. Entry points live in `cmd/claudego` and `cmd/demo`. Private application packages are under `internal/`, including configuration, tools, commands, version metadata, and the agent loop. Reusable libraries live in `pkg/`, grouped by concern: `conversation`, `llm`, `permissions`, `skill`, `ui`, `logger`, `compaction`, `agent`, `types`, and `interfaces`. Shared helpers are in `utils/`. Tests are colocated with implementation files as `*_test.go`. Assets for documentation are stored in `assets/`, design notes in `docs/`, and release artifacts are generated into `dist/`.

## Build, Test, and Development Commands

- `go test ./...` runs the full test suite.
- `go build -o claudego ./cmd/claudego` builds the local CLI binary.
- `./claudego` starts the REPL; create `~/.claudego/config.json` first, using `config.example.json` as a template.
- `./scripts/build.sh` cross-compiles release binaries into `dist/`.
- `./scripts/build.sh --bump patch` increments `VERSION`, builds all targets, and refreshes checksums.

## Coding Style & Naming Conventions

Use standard Go formatting. Run `gofmt` on edited Go files before committing. Keep package names short, lowercase, and aligned with directory names. Prefer explicit interfaces in `pkg/interfaces` only when multiple packages need the contract. Keep internal-only code under `internal/` and avoid importing it from reusable `pkg/` packages.

## Testing Guidelines

Use Go’s standard `testing` package with `testify` where assertions are already used. Name test files `*_test.go` and test functions `TestXxx`. Add focused regression tests for bug fixes, especially around permissions, conversation rollback, config loading, skill handling, and UI rendering. Run `go test ./...` before opening a pull request.

## Commit & Pull Request Guidelines

Recent history uses concise Conventional Commit-style prefixes such as `feat:` and `chore:`. Keep the subject imperative and scoped to the user-visible reason for the change. Pull requests should include a short summary, test evidence, linked issues when relevant, and screenshots or terminal output for UI/CLI behavior changes. Note any skipped tests or release-build impacts.

## Security & Configuration Tips

Never commit real API keys or local `~/.claudego/config.json` contents. Permission rules should default to least privilege: deny dangerous shell patterns, ask before writes or command execution, and document any broadened allow rules in the PR.
