---
description: Finalize completed task with proper commit message
argument-hint: [rev]
allowed-tools:
 - Skill(jjtask)
 - Bash
---

<objective>
Strip [task:*] prefix from a completed task, converting it to a regular commit.

Use after task is done and ready to be merged/pushed.

Part of `/jjtask` - run that skill for full workflow context.
</objective>

<context>
Done tasks:
!`jjtask find done 2>/dev/null | head -10`
</context>

<process>
1. Run: `jjtask finalize $ARGUMENTS`
2. Confirm the task prefix was removed
</process>

<success_criteria>
- Task [task:done] prefix stripped
- Commit message is clean
</success_criteria>
