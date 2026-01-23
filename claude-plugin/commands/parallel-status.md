---
description: Show status of parallel agent session
argument-hint: [parent-task]
allowed-tools:
 - Skill(jjtask)
 - Bash
---

<objective>
View status of all agents in a parallel session.

Shows:
- Session mode and duration
- Each agent's task flag (todo/wip/done)
- File changes per agent
- Potential conflicts

Part of `/jjtask` - run that skill for full workflow context.
</objective>

<process>
1. Run: `jj task parallel-status $ARGUMENTS`
2. Review agent progress
3. Check for conflicts
</process>

<success_criteria>
- All agents and their status displayed
- Conflicts highlighted if any
</success_criteria>
