# Workflow Reference

## DAG Validation

When reviewing tasks with `jjtask find`, look for structural issues:

Good DAG - chained tasks show priority, parallel tasks are siblings:
```
o  E [todo] Feature complete   <- gate: all children done, tests pass, reviewed
|-+-,
| | o  D2 [todo] Write docs    <- parallel with D1
| o |  D1 [todo] Add tests     <- parallel with D2
|-' |
o   |  C [todo] Implement      <- after B
o  -'  B [todo] Design API     <- after A
o  A [todo] Research           <- do first
@  current work
```
Reading bottom-up: A -> B -> C -> (D1 || D2) -> E (gate)

Task E is a "gate" - marks feature complete only when all children done.

Bad DAG - all siblings, no priority visible:
```
| o  E [todo] Deploy
|-'
| o  D [todo] Write docs
|-'
| o  C [todo] Implement
|-'
| o  B [todo] Design API
|-'
| o  A [todo] Research
|-'
@  current work
```
Problem: Which task comes first? No way to tell.
Fix: Chain dependent tasks with `jj rebase -s B -o A`

### Dependency problems
- Task mentions another task but isn't a child of it -> `jj rebase -s TASK -o DEPENDENCY`
- Task requires output from another but they're siblings -> rebase to make sequential
- Keywords: "after", "requires", "depends on", "once X is done", "needs"

### Parallelization opportunities
- Sequential tasks that don't share state -> could be parallel siblings
- Independent features under same parent -> good candidates for parallel agents

### Structural issues
- Done tasks with pending children -> children may be blocked
- Draft tasks mixed with todo -> drafts need specs before work begins

## Working in Merge

When @ is a merge of multiple WIP tasks:

**Recommended: Work directly in task branch**
```bash
jj edit task-a        # Switch to working in the task
# make changes...
jjtask wip task-a     # Rebuild merge to see combined state
```

**Alternative: Use absorb with explicit targets**
```bash
jj absorb --into 'tasks_wip()'  # Only route to WIP tasks
```

**Avoid bare `jj absorb`** - it may route changes to ancestor commits if you're editing lines not touched by your task branches.

## Squashing

After tasks are complete, flatten the merge for a clean push:

```bash
jjtask squash
# Combines all merged task content into a single linear commit
# Task descriptions become bullet points in commit message
```
