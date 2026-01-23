---
description: Update task flag on a revision
argument-hint: <rev> <flag>
allowed-tools:
 - Skill(jjtask)
 - Bash
---

<objective>
Change the task status flag on a revision.

Flags: draft, todo, wip, blocked, standby, untested, review, done

Part of `/jjtask` - run that skill for full workflow context.
</objective>

<context>
Current tasks:
!`jjtask find 2>/dev/null | head -20`
</context>

<process>
1. Run: `jjtask flag $ARGUMENTS`
2. Confirm the flag was updated
</process>

<success_criteria>
- Task flag updated
- No conflicts created
</success_criteria>
