---
description: Start a parallel agent session
argument-hint: [--mode shared|workspace] [--agents N] <parent-task>
allowed-tools:
 - Skill(jjtask)
 - Bash
 - Read
---

<objective>
Start a parallel agent session for multi-agent work on the same repo.

Modes:
- shared (default): All agents share same @ revision, partitioned by file patterns
- workspace: Each agent gets isolated workspace directory

Part of `/jjtask` - run that skill for full workflow context.
</objective>

<context>
Current tasks:
!`jj task find 2>/dev/null | head -20 || echo "no tasks"`
</context>

<process>
1. Run: `jj task parallel-start $ARGUMENTS`
2. Note the agent assignments and modes
3. For workspace mode, confirm workspaces created in `.jjtask-workspaces/`
</process>

<success_criteria>
- Session started with specified mode
- Agent assignments recorded in parent task description
- Workspaces created (if workspace mode)
</success_criteria>
