---
description: Find revisions with specific task flags
argument-hint: [flag]
allowed-tools:
 - Skill(jjtask)
 - Bash
---

<objective>
List task revisions filtered by status flag.

Without arguments: shows all pending tasks.
With flag: shows only tasks with that flag (todo, wip, done, blocked, etc.)

Part of `/jjtask` - run that skill for full workflow context.
</objective>

<context>
!`jjtask find $ARGUMENTS`
</context>

<process>
1. Review the task list above
2. Suggest next actions based on task states
</process>

<success_criteria>
- Task list displayed
- Actionable suggestions provided
</success_criteria>
