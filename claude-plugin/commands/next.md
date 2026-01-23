---
description: Review current jj task or transition to next
argument-hint: [--mark-as <status> <rev>]
allowed-tools:
 - Skill(jjtask)
 - Read
 - Bash
---

<objective>
Review current jj todo task status or transition to the next task.

Without arguments: shows current task specs and available next tasks.
With --mark-as: marks current task with status and moves to specified revision.

Part of `/jjtask` - run that skill for full workflow context.
</objective>

<context>
Current status: !`jjtask next`
</context>

<process>
1. If no arguments, review the current task output above
2. If transitioning, run: `jjtask next $ARGUMENTS`
3. Confirm the transition completed
</process>

<success_criteria>
- Current task status is clear
- If transitioning, new task is now current
</success_criteria>
