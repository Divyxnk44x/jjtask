---
name: jjtask
description: Structured TODO commit workflow using JJ (Jujutsu). Use to plan tasks as empty commits with [task:*] flags, track progress through status transitions, manage parallel task DAGs with dependency checking. Enforces completion discipline. Enables to divide work between Planners and Workers.
---

<context>
Designed for JJ 0.36.x+. Uses revset aliases and templates defined in jjtask config.
</context>

<objective>
Manage a DAG of empty revisions as TODO markers representing tasks. Revision descriptions act as specifications. Two roles: Planners (create empty revisions with specs) and Workers (implement them). For JJ basics (revsets, commands, recovery), see the `/jj` skill.
</objective>

<quick_start>

```bash
# 1. Plan: Create TODO tasks
jjtask create "Add user validation" "Check email format and password strength"
jjtask create --chain "Add validation tests" "Test valid/invalid emails and passwords"

# 2. Start working on a task
jjtask wip abc123
# Single task: @ becomes the task (jj edit)
# Multiple WIP: @ becomes merge commit

# 3. Work and complete
# For single task: work directly in @
# For merge: jj edit TASK to work in specific task
jjtask done abc123
# Task rebases ON TOP of work commits, then @ rebases onto task

# 4. Flatten for push
jjtask squash
# Squashes all merged task content into linear commit
```
</quick_start>

<commands>

| Command                                | Purpose                            |
| -------------------------------------- | ---------------------------------- |
| `jjtask find [-s STATUS]`              | Show task DAG (pending by default) |
| `jjtask create [PARENT] TITLE [DESC]`  | Create TODO (parent defaults to @) |
| `jjtask wip [TASKS...]`                | Mark WIP, add as parents of @      |
| `jjtask done [TASKS...]`               | Mark done, rebase on top of work   |
| `jjtask drop TASKS...`                 | Remove from @ without completing   |
| `jjtask squash`                        | Flatten @ merge for push           |
| `jjtask flag STATUS [-r REV]`          | Update status flag                 |
| `jjtask show-desc [-r REV]`            | Print revision description         |

Status flags: `draft` → `todo` → `wip` → `done` (also: `blocked`, `standby`, `untested`, `review`)
</commands>

<completion_discipline>

Do NOT mark done unless ALL acceptance criteria are met.

Mark done when:
- Every requirement implemented
- All acceptance criteria pass
- Tests pass

Never mark done when:
- "Good enough" or "mostly works"
- Tests failing
- Partial implementation
</completion_discipline>

<anti_patterns>
<pitfall name="stop-and-report">
If you encounter these issues, STOP and report:
- Made changes in wrong revision
- Previous work needs fixes
- Uncertain about how to proceed
- Dependencies unclear

Do NOT attempt to fix using JJ operations not in this workflow.
</pitfall>
</anti_patterns>

<references>
For detailed documentation, read these files as needed:
- `references/workflow.md` - DAG validation, working in merges, squashing
- `references/parallel.md` - Parallel tasks, multi-repo, parallel agents
- `references/descriptions.md` - Description format, transforms, batch operations
- `references/command-syntax.md` - Full command reference with all flags
</references>
