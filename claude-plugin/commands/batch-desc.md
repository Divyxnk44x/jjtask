---
description: Apply sed transformations to multiple revision descriptions
argument-hint: <sed-expr> <revset>
allowed-tools:
 - Bash
---

<objective>
Transform descriptions of multiple revisions matching a revset.

Example: `jjtask batch-desc 's/old/new/' 'tasks_pending()'`
</objective>

<process>
Run: `jjtask batch-desc $ARGUMENTS`
</process>

<success_criteria>
- Descriptions updated for matching revisions
- No errors reported
</success_criteria>
