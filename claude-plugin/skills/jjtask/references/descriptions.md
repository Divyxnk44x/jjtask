# Description Management Reference

## Task Description Format

```
Short title (< 50 chars)

## Context
Why this task exists, what problem it solves.

## Requirements
- Specific requirement 1
- Specific requirement 2

## Acceptance criteria
- Criterion 1 (testable)
- Criterion 2 (testable)
```

## Modifying Descriptions

Flag changes only update status. To modify description content:

```bash
# Add completion notes when marking done
jjtask done xyz
jj desc -r xyz -m "$(jjtask show-desc -r xyz)

## Completion
- What was done
- Deviations from spec"

# Check off acceptance criteria
jjtask desc-transform 's/- \[ \] First criterion/- [x] First criterion/'

# Append a section
jjtask desc-transform 's/$/\n\n## Notes\nAdditional context here/'

# Batch update multiple tasks
jjtask batch-desc 's/old-term/new-term/g' -r 'tasks_todo()'
```

## When to Use What

- `jjtask flag` - status only
- `jj desc -r REV -m "..."` - replace entire description
- `jjtask desc-transform` - partial find/replace (supports multiline with `\n`)
- `jjtask desc-transform --stdin` - read new description from stdin
- `jjtask batch-desc` - same transform across multiple tasks
