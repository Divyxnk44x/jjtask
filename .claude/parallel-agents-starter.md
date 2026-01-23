# Parallel Agents Feature - Session Starter

## Quick Start (Single Agent - Sequential)

```
Continue jjtask parallel agents implementation.

Read the handoff first:
  jj task show-desc qq

Task DAG starting point:
  jj log -r 'descendants(mt)'

Start with these tasks in order:
1. zr - Schema parser (internal/parallel/schema.go)
2. zt - Workspace management (internal/workspace/workspace.go)
3. pp - parallel-start command (cmd/jjtask/cmd/parallel_start.go)
4. qk - Mode 1 shared filesystem implementation

Mark tasks done as you complete them: jj task flag <id> done
```

---

## Parallel Start (2 Agents - Shared Mode)

### Setup (run once before launching agents)
```bash
cd /Users/alex/Projects/jjtask
jj edit mt
jj task flag @ wip
```

### Agent A Prompt
```
You are agent-a implementing jjtask parallel agents feature.

Repo: /Users/alex/Projects/jjtask
Mode: shared (you share @ with agent-b)

YOUR ASSIGNMENT: internal/parallel/**
AVOID: internal/workspace/** (agent-b)

FIRST: Read the handoff
  jj task show-desc qq

YOUR TASK: zr (Format: Task description schema)
  jj task show-desc zr

Create internal/parallel/schema.go with:
- ParallelSession struct
- Agent struct
- ParseParallelSession(description string) function
- FormatParallelSession(session) function

When done: jj task flag zr done
Then continue with: lo (agent-context command)
```

### Agent B Prompt
```
You are agent-b implementing jjtask parallel agents feature.

Repo: /Users/alex/Projects/jjtask
Mode: shared (you share @ with agent-a)

YOUR ASSIGNMENT: internal/workspace/**
AVOID: internal/parallel/** (agent-a)

FIRST: Read the handoff
  jj task show-desc qq

YOUR TASK: zt (Setup: .jjtask-workspaces management)
  jj task show-desc zt

Create internal/workspace/workspace.go with:
- EnsureWorkspacesDir() function
- EnsureIgnored() function (adds to .git/info/exclude)
- CreateWorkspace(name, revision) function
- CleanupWorkspaces() function

When done: jj task flag zt done
Then continue with: pp (parallel-start command) with agent-a
```

---

## Parallel Start (2 Agents - Workspace Mode)

### Setup (run once)
```bash
cd /Users/alex/Projects/jjtask
echo '.jjtask-workspaces/' >> .git/info/exclude
mkdir -p .jjtask-workspaces

# Create workspaces for independent work
jj workspace add .jjtask-workspaces/agent-a --revision zr --name agent-a
jj workspace add .jjtask-workspaces/agent-b --revision zt --name agent-b
```

### Agent A Prompt
```
You are agent-a implementing jjtask parallel agents feature.

Working directory: /Users/alex/Projects/jjtask/.jjtask-workspaces/agent-a
Mode: workspace (isolated)

FIRST: Read the handoff
  jj task show-desc qq

YOUR TASK: zr (Format: Task description schema)
  jj task show-desc @-

Create internal/parallel/schema.go with:
- ParallelSession struct
- Agent struct
- ParseParallelSession(description string) function
- FormatParallelSession(session) function

When done: jj task flag @- done
```

### Agent B Prompt
```
You are agent-b implementing jjtask parallel agents feature.

Working directory: /Users/alex/Projects/jjtask/.jjtask-workspaces/agent-b
Mode: workspace (isolated)

FIRST: Read the handoff
  jj task show-desc qq

YOUR TASK: zt (Setup: .jjtask-workspaces management)
  jj task show-desc @-

Create internal/workspace/workspace.go with:
- EnsureWorkspacesDir() function
- EnsureIgnored() function
- CreateWorkspace(name, revision) function
- CleanupWorkspaces() function

When done: jj task flag @- done
```

### Cleanup (after both agents done)
```bash
jj workspace forget agent-a agent-b
rm -rf .jjtask-workspaces/
# Merge work
jj new zr zt -m "Parallel agents foundation"
```

---

## Task Reference

| ID | Task | Files |
|----|------|-------|
| qq | Handoff doc | (read this first) |
| zr | Schema parser | internal/parallel/schema.go |
| zt | Workspace mgmt | internal/workspace/workspace.go |
| pp | parallel-start cmd | cmd/jjtask/cmd/parallel_start.go |
| qk | Mode 1 (shared) | cmd/jjtask/cmd/parallel_start.go |
| qum | Mode 2 (workspace) | cmd/jjtask/cmd/parallel_start.go |
| lo | agent-context cmd | cmd/jjtask/cmd/agent_context.go |
| lt | parallel-status cmd | cmd/jjtask/cmd/parallel_status.go |
| nx | parallel-stop cmd | cmd/jjtask/cmd/parallel_stop.go |
| rn | prime enhancement | cmd/jjtask/cmd/prime.go |

## Useful Commands

```bash
jj task show-desc <id>     # Read task spec
jj task flag <id> wip      # Start working
jj task flag <id> done     # Mark complete
jj log -r 'descendants(mt)' # See DAG
jj task find               # See pending tasks
```
