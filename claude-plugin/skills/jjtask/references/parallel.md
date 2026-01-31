# Parallel Tasks Reference

## Creating Parallel Tasks

```bash
# Create parallel branches from @ (default parent)
jjtask parallel "Widget A" "Widget B" "Widget C"

# Or specify parent explicitly
jjtask parallel --parent xyz123 "Widget A" "Widget B"

# Merge point (all parents must complete)
jj new --no-edit <A-id> <B-id> <C-id> -m "[task:todo] Integration\n\n..."
```

## Multi-Repo Setup

Create `.jj-workspaces.yaml` in project root:

```yaml
repos:
  - path: frontend
    name: frontend
  - path: backend
    name: backend
```

Scripts show output grouped by repo. Use `jjtask all log` or `jjtask all diff` across repos.

## Parallel Agents

Multiple Claude agents can work simultaneously using jj workspaces:

```bash
# Terminal 1: Agent working on task A
jj workspace add .workspaces/agent-a --revision task-a
cd .workspaces/agent-a
# work...
jjtask done  # Rebuilds this workspace's @

# Terminal 2: Agent working on task B
jj workspace add .workspaces/agent-b --revision task-b
cd .workspaces/agent-b
# work...
jjtask done  # Rebuilds this workspace's @

# Cleanup when done
jj workspace forget agent-a
rm -rf .workspaces/agent-a
```

Each workspace has its own @ that mega-merge rebuilds independently.
No special coordination needed - jj handles workspace isolation.
