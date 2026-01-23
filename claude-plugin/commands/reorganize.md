---
description: Analyze task DAG and suggest reorganization
allowed-tools:
 - Skill(jjtask)
 - Bash
 - Read
---

<objective>
Review the current task DAG structure and suggest improvements:
- Identify dependency issues (task mentions another but isn't a child)
- Find parallelization opportunities (independent tasks that could run concurrently)
- Detect structural problems (orphaned tasks, blocked children, incomplete drafts)

Part of `/jjtask` - run that skill for full workflow context.
</objective>

<context>
Current task DAG:
!`jjtask find 2>/dev/null || echo "no tasks"`

Task descriptions (for dependency analysis):
!`for rev in $(jj log -r 'tasks_pending()' -T 'change_id.shortest() ++ "\n"' --no-graph 2>/dev/null | head -10); do echo "=== $rev ==="; jjtask show-desc "$rev" 2>/dev/null | head -20; echo; done`
</context>

<process>
1. Review the DAG structure above
2. Read task descriptions looking for dependency keywords: "after", "requires", "depends on", "needs", "once X is done"
3. Identify issues:
   - Tasks referencing others that aren't ancestors
   - Sequential tasks that could be parallel
   - Orphaned tasks needing hoist
4. Propose concrete rebase commands for each issue
5. Execute rebases only with user confirmation
</process>

<success_criteria>
- DAG analyzed for dependency/structure issues
- Concrete rebase commands proposed (if issues found)
- No rebases executed without user approval
</success_criteria>
