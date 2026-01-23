---
description: End parallel session and cleanup
argument-hint: [--merge] [--force] [parent-task]
allowed-tools:
 - Skill(jjtask)
 - Bash
---

<objective>
Clean up a parallel session - optionally merge work and remove workspaces.

Options:
- --merge: Squash completed agent work into parent task
- --force: Stop even if some agents aren't done
- --keep-workspaces: Don't remove workspace directories

Part of `/jjtask` - run that skill for full workflow context.
</objective>

<process>
1. Run: `jj task parallel-stop $ARGUMENTS`
2. Verify workspaces cleaned up (if workspace mode)
3. Confirm parent task status updated
</process>

<success_criteria>
- Session ended
- Workspaces removed (unless --keep-workspaces)
- Parent task marked done (if all agents complete)
</success_criteria>
