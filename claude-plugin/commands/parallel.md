---
description: Create sibling tasks under a parent
argument-hint: <parent> <title1> <title2> [title3...]
allowed-tools:
 - Skill(jjtask)
 - Bash
---

<objective>
Create multiple parallel task branches from the same parent.

Use for tasks that can be worked on independently.

Part of `/jjtask` - run that skill for full workflow context.
</objective>

<context>
Recent commits (potential parents):
!`jj log --limit 10`
</context>

<process>
1. Run: `jjtask parallel $ARGUMENTS`
2. Confirm all tasks were created
</process>

<success_criteria>
- Multiple sibling tasks created under specified parent
</success_criteria>
