---
description: Show revision description
argument-hint: [rev]
allowed-tools:
 - Bash
---

<objective>
Print the description of a revision. Defaults to @ if no rev specified.
</objective>

<process>
Run: `jjtask show-desc $ARGUMENTS`
</process>

<success_criteria>
- Description printed to output
</success_criteria>
