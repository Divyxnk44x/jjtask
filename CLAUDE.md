# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

jjtask is a portable Claude Code plugin for structured task management using JJ (Jujutsu) version control. It uses empty revisions as TODO markers with `[task:*]` flags in descriptions, forming a DAG of tasks that can be planned and executed.

## Architecture

```
jjtask/
├── cmd/jjtask/             # Go CLI source
│   ├── main.go             # Entry point
│   └── cmd/                # Cobra commands (find, create, flag, next, etc.)
├── internal/               # Go internal packages
│   ├── jj/                 # JJ interaction layer
│   ├── parallel/           # Parallel agent session management
│   └── workspace/          # Multi-workspace support
├── bin/
│   ├── jjtask              # Dispatcher (downloads/runs jjtask-go)
│   └── jj                  # Wrapper with agent guardrails
├── claude-plugin/          # Distributable plugin package
│   ├── .claude-plugin/     # Plugin manifest
│   ├── bin/                # Binaries (jjtask, jj, jjtask-go)
│   ├── commands/           # Slash commands (/jjtask:create, etc.)
│   └── skills/             # Skills (jj, jjtask)
├── config/
│   └── conf.d/
│       └── 10-jjtask.toml  # JJ revset aliases and templates
├── shell/fish/
│   ├── completions/        # Generated fish completions
│   └── functions/          # jjtask-env.fish shell setup
├── test/                   # Integration tests and snapshots
├── .github/workflows/      # CI and release automation
├── install.sh              # Installer (builds Go, symlinks, completions)
├── test.sh                 # Integration test runner
└── .mise.toml              # Toolchain and tasks (Go 1.25, golangci-lint)
```

## Multi-Workspace Support

For projects with multiple jj repos, create `.jj-workspaces.yaml` in project root:

```yaml
repos:
  - path: frontend
    name: frontend
  - path: backend
    name: backend
  - path: .
    name: root
```

Scripts auto-detect this config and operate across all repos:
- `jjtask find` shows tasks grouped by repo
- `jjtask all log`, `jjtask all diff` aggregate output
- Context line shows: `cwd: subdir | repo: name | workspace: ../..`

## Task Flags

Status progression: `draft` → `todo` → `wip` → `done`

Additional flags: `blocked`, `standby`, `untested`, `review`

Revset aliases: `tasks()`, `tasks_pending()`, `tasks_todo()`, `tasks_wip()`, `tasks_done()`

## Development

Requires [mise](https://mise.jdx.dev/) for toolchain management.

```bash
mise install       # Install Go 1.25 + golangci-lint

mise run build     # Build binary to bin/jjtask-go
mise run test      # Run integration tests (./test.sh)
mise run lint      # Run golangci-lint
mise run fmt       # Format Go code
mise run dev       # Dev setup: symlinks + completions
```

Manual workflow:
```bash
go build -o bin/jjtask-go ./cmd/jjtask   # Build
./test.sh                                 # Test
./install.sh                              # Install to ~/.local/bin
./install.sh --uninstall                  # Remove
```

## Releasing

```bash
mise run release v0.1.0   # Updates plugin.json, commits, tags, pushes
```

This will:
1. Update version in `claude-plugin/.claude-plugin/plugin.json`
2. Commit the version bump
3. Push to origin
4. Create and push the git tag
5. GitHub Actions builds binaries and creates the release

## Conventions

- Go code in `cmd/jjtask/cmd/` for commands, `internal/` for shared packages
- Use Cobra for CLI structure with persistent flags for JJ globals (-R, --quiet)
- Prefer `change_id.shortest()` over `change_id.short()` in templates
- Integration tests use snapshot comparison (test/snapshots/)
- Update snapshots with `SNAPSHOT_UPDATE=1 ./test.sh`

## Creating Skills/Commands

Use Claude Code skill creation workflow:
- `/create-agent-skill` for SKILL.md
- `/create-slash-command` for commands/
- `/audit-skill` and `/audit-slash-command` to verify

## Shell Integration

Fish shell function for temporary environment activation:

```fish
source ~/jjtask/shell/fish/functions/jjtask-env.fish
jjtask-env        # Activate: adds bin/ to PATH, sets JJ_CONFIG
jjtask-env off    # Deactivate
```

For wrapper-based activation (recommended), use `install.sh --wrapper` which creates
a fish function or bash alias that wraps `claude` with jjtask environment.

## Config

The installer symlinks `config/conf.d/10-jjtask.toml` into `~/.config/jj/conf.d/` which adds:
- Task revset aliases (tasks(), tasks_pending(), etc.)
- Templates for task display

For agent mode (Claude Code), use `install.sh --agent` for instructions on setting JJ_CONFIG.

## Slash Command Pattern

Commands use dynamic context with `!` backtick syntax:

```yaml
---
description: Create a new todo task revision
argument-hint: [parent] <title> [description]
allowed-tools:
 - Skill(jjtask)
 - Read
 - Bash
---

<context>
Existing tasks:
!`jjtask find 2>/dev/null || echo "no tasks"`
</context>

<process>
1. Run: `jjtask create $ARGUMENTS`
</process>
```
