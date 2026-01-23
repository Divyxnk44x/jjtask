---
description: Transform a revision description with sed
argument-hint: <rev> <sed-expr>
allowed-tools:
 - Bash
---

<objective>
Pipe a revision's description through sed and update it.

Example: `jjtask desc-transform @ 's/foo/bar/'`
</objective>

<process>
Run: `jjtask desc-transform $ARGUMENTS`
</process>

<success_criteria>
- Description transformed successfully
- No errors from sed expression
</success_criteria>
