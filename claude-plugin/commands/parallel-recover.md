---
description: Recover from parallel session issues
argument-hint: [--workspace <agent>] [--session <task>] [--reset|--abandon|--recreate]
allowed-tools:
 - Skill(jjtask)
 - Bash
---

<objective>
Recover from problems in parallel agent sessions.

Options:
- --workspace <agent>: Recover specific agent workspace
- --session <task>: Recover entire session
- --reset: Reset workspace to task revision
- --abandon: Abandon uncommitted changes
- --recreate: Delete and recreate workspace

Part of `/jjtask` - run that skill for full workflow context.
</objective>

<process>
1. Run: `jj task parallel-recover $ARGUMENTS`
2. If no args, review status and choose recovery action
3. Verify workspace/session recovered
</process>

<success_criteria>
- Workspace or session recovered
- No data loss
</success_criteria>
