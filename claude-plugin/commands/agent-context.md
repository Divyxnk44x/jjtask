---
description: Get context for an agent in parallel session
argument-hint: <agent-id>
allowed-tools:
 - Skill(jjtask)
 - Bash
---

<objective>
Get assignment and context for an agent in a parallel session.

Shows:
- Mode (shared/workspace)
- Your file assignment (patterns to work on)
- Files to AVOID (other agents' assignments)
- Other agents' status

Part of `/jjtask` - run that skill for full workflow context.
</objective>

<process>
1. Run: `jj task agent-context $ARGUMENTS`
2. Review your assignment
3. Note files to avoid
</process>

<success_criteria>
- Assignment displayed
- Other agents' patterns shown
</success_criteria>
