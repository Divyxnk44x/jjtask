---
description: Remove tasks from @ merge without marking done
argument-hint: <tasks...> [--abandon]
allowed-tools:
 - Bash
 - AskUserQuestion
model: haiku
---

<objective>
Remove tasks from @ parents without marking them done.

Marks as 'standby' by default so they can be re-added later.
Use --abandon to permanently remove.

Part of mega-merge workflow - see `/jjtask` for full context.
</objective>

<context>
Current WIP tasks:
!`jjtask find wip 2>/dev/null || echo "no wip tasks"`
</context>

<process>
Run: `jjtask drop $ARGUMENTS`

- `jjtask drop xyz` - mark as standby, remove from @
- `jjtask drop a b c` - drop multiple tasks
- `jjtask drop --abandon xyz` - abandon task entirely
</process>
