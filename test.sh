#!/usr/bin/env bash
# Run jjtask tests in parallel (bounded by CPU cores)
# Usage: ./test.sh [-j N] [--sequential]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/test/test_helper.bash"

# Colors (only if terminal)
if [[ -t 1 ]]; then
  RED='\033[0;31m'
  GREEN='\033[0;32m'
  NC='\033[0m'
else
  RED=''
  GREEN=''
  NC=''
fi

# Run a single test in isolation, output result to stdout
run_one_test() {
  local name="$1"
  local func="$2"
  local setup_func="${3:-setup_test_repo}"
  local teardown_func="${4:-teardown_test_repo}"

  $setup_func

  if $func; then
    echo -e "${GREEN}✓${NC} $name"
    $teardown_func
    return 0
  else
    echo -e "${RED}✗${NC} $name"
    $teardown_func
    return 1
  fi
}

# Test definitions - each is: "name|function|setup|teardown"
TESTS=(
  # Basic tests
  "jjtask create creates a task|test_jot_create"
  "jjtask create with --draft flag|test_jot_create_draft"
  "jjtask flag updates task status|test_jot_flag"
  "jjtask find shows tasks|test_jot_find"
  "jjtask find output format (snapshot)|test_jot_find_snapshot"
  "jjtask find -r filters to tasks only|test_jot_find_revset_filters_tasks"
  "jjtask show-desc outputs description|test_jot_show_desc"
  "jjtask parallel creates sibling tasks|test_jot_parallel"
  "jjtask finalize strips task prefix|test_jot_finalize"
  "jjtask prime outputs context|test_jot_prime"
  "jjtask checkpoint creates checkpoint|test_jot_checkpoint"
  "jjtask desc-transform transforms description|test_jot_desc_transform"
  "jjtask desc-transform error on invalid command|test_jot_desc_transform_error"
  "jj config: task_log shows diff stats for @|test_config_task_log_diff_stats"
  "jj config: task_log shows short descriptions|test_config_task_log_short_desc"
  # Hoist tests
  "jjtask hoist single task|test_hoist_single"
  "jjtask hoist multiple tasks|test_hoist_multiple"
  "jjtask hoist no-op when already hoisted|test_hoist_noop"
  "jjtask hoist nested tasks|test_hoist_nested"
  "jjtask hoist with no pending tasks|test_hoist_empty"
  # Multi-repo tests
  "jjtask find across repos|test_multi_jot_find|setup_multi_repo|teardown_multi_repo"
  "jjtask all log across repos|test_multi_jot_all_log|setup_multi_repo|teardown_multi_repo"
  "workspace hint from subdirectory|test_multi_workspace_hint|setup_multi_repo|teardown_multi_repo"
  "complex multi-repo with tasks|test_multi_complex_tasks|setup_multi_repo|teardown_multi_repo"
  # Parallel agent tests
  "parallel-start shared mode|test_parallel_start_shared"
  "parallel-start workspace mode|test_parallel_start_workspace"
  "parallel-status shows session|test_parallel_status"
  "agent-context returns info|test_agent_context"
  "parallel-stop cleans up|test_parallel_stop"
  "parallel-start invalid mode error|test_parallel_start_invalid_mode"
  "agent-context unknown agent error|test_agent_context_unknown"
)

# Test functions
test_jot_create() {
  jjtask create @ "Test task" "Test description" >/dev/null 2>&1
  has_task_with_flag todo
}

test_jot_create_draft() {
  jjtask create --draft @ "Draft task" >/dev/null 2>&1
  has_task_with_flag draft
}

test_jot_flag() {
  jjtask create @ "Test task" >/dev/null 2>&1
  local task_id
  task_id=$(get_task_id todo)
  [[ -n "$task_id" ]] || return 1
  jjtask flag "$task_id" wip >/dev/null 2>&1
  has_task_with_flag wip
}

test_jot_find() {
  jjtask create @ "Task A" >/dev/null 2>&1
  jjtask create @ "Task B" >/dev/null 2>&1
  local output
  output=$(jjtask find 2>/dev/null)
  [[ "$output" == *"Task A"* ]] && [[ "$output" == *"Task B"* ]]
}

test_jot_find_snapshot() {
  jjtask create @ "First task" "Description A" >/dev/null 2>&1
  jjtask create @ "Second task" "Description B" >/dev/null 2>&1
  local output
  output=$(jjtask find 2>/dev/null)
  assert_snapshot "find_tasks" "$output"
}

test_jot_find_revset_filters_tasks() {
  # Create a task and a regular commit - find -r should only show tasks
  jjtask create @ "My task" >/dev/null 2>&1
  jj new -m "Regular commit" >/dev/null 2>&1
  local output
  # Using all() would show both, but find -r should filter to tasks only
  output=$(jjtask find -r 'all()' 2>/dev/null)
  [[ "$output" == *"My task"* ]] || return 1
  # Should NOT contain the regular commit
  [[ "$output" != *"Regular commit"* ]]
}

test_jot_show_desc() {
  jjtask create @ "Test title" "Test body content" >/dev/null 2>&1
  local task_id
  task_id=$(get_task_id todo)
  [[ -n "$task_id" ]] || return 1
  jj edit "$task_id" >/dev/null 2>&1
  local output
  output=$(jjtask show-desc @)
  assert_snapshot "show_desc" "$output"
}

test_jot_parallel() {
  jjtask parallel @ "Task A" "Task B" "Task C" >/dev/null 2>&1
  local output
  output=$(jjtask find 2>/dev/null)
  assert_snapshot "parallel_tasks" "$output"
}

test_jot_finalize() {
  jjtask create @ "Finalize test" "## Done criteria
- Task completed" >/dev/null 2>&1
  local task_id
  task_id=$(get_task_id todo)
  [[ -n "$task_id" ]] || return 1
  jjtask flag "$task_id" done >/dev/null 2>&1
  jjtask finalize "$task_id" >/dev/null 2>&1
  local output
  output=$(jjtask show-desc "$task_id")
  assert_snapshot "finalize_output" "$output"
}

test_jot_prime() {
  local output
  output=$(jjtask prime)
  assert_snapshot "prime_output" "$output"
}

test_jot_checkpoint() {
  local output
  output=$(jjtask checkpoint "test-checkpoint")
  assert_snapshot "checkpoint_output" "$output"
}

test_jot_desc_transform() {
  jjtask create @ "Original title" "## Context
Some context here" >/dev/null 2>&1
  local task_id
  task_id=$(get_task_id todo)
  [[ -n "$task_id" ]] || return 1
  jjtask desc-transform "$task_id" sed 's/Original/Modified/' >/dev/null 2>&1
  local output
  output=$(jjtask show-desc "$task_id")
  assert_snapshot "desc_transform_output" "$output"
}

test_jot_desc_transform_error() {
  jjtask create @ "Test title" >/dev/null 2>&1
  local task_id
  task_id=$(get_task_id todo)
  [[ -n "$task_id" ]] || return 1
  local output
  output=$(jjtask desc-transform "$task_id" "nonexistent-cmd-xyz" 2>&1) && return 1
  assert_snapshot "desc_transform_error" "$output"
}

test_config_task_log_diff_stats() {
  echo "test content" > testfile.txt
  jj describe -m "Test commit with changes" >/dev/null 2>&1
  local output
  output=$(jj log -r @ --no-graph -T task_log 2>/dev/null)
  assert_snapshot "task_log_diff_stats" "$output"
}

test_config_task_log_short_desc() {
  jjtask create @ "Short title" "## Context
This is a longer description
with multiple lines" >/dev/null 2>&1
  local task_id
  task_id=$(get_task_id todo)
  [[ -n "$task_id" ]] || return 1
  local output
  output=$(jj log -r "$task_id" --no-graph -T task_log 2>/dev/null)
  assert_snapshot "task_log_short_desc" "$output"
}

test_hoist_single() {
  jjtask create @ "Task to hoist" >/dev/null 2>&1
  jj new -m "New work" >/dev/null 2>&1
  local output
  output=$(jjtask hoist 2>&1)
  assert_snapshot "hoist_single" "$output"
}

test_hoist_multiple() {
  jjtask create @ "Task A" >/dev/null 2>&1
  jjtask create @ "Task B" >/dev/null 2>&1
  jjtask create @ "Task C" >/dev/null 2>&1
  jj new -m "New work" >/dev/null 2>&1
  local output
  output=$(jjtask hoist 2>&1)
  assert_snapshot "hoist_multiple" "$output"
}

test_hoist_noop() {
  jjtask create @ "Already hoisted task" >/dev/null 2>&1
  local output
  output=$(jjtask hoist 2>&1)
  assert_snapshot "hoist_noop" "$output"
}

test_hoist_nested() {
  jjtask create @ "Parent task" >/dev/null 2>&1
  local parent_id
  parent_id=$(get_task_id todo)
  jjtask create "$parent_id" "Child task" >/dev/null 2>&1
  jj new -m "New work" >/dev/null 2>&1
  local output
  output=$(jjtask hoist 2>&1)
  local find_output
  find_output=$(jjtask find 2>/dev/null)
  assert_snapshot "hoist_nested" "$output"$'\n'"---"$'\n'"$find_output"
}

test_hoist_empty() {
  local output
  output=$(jjtask hoist 2>&1)
  assert_snapshot "hoist_empty" "$output"
}

test_multi_jot_find() {
  (cd frontend && jjtask create @ "Frontend task" >/dev/null 2>&1)
  (cd backend && jjtask create @ "Backend task" >/dev/null 2>&1)
  local output
  output=$(jjtask find 2>/dev/null)
  assert_snapshot "multi_repo_find" "$output"
}

test_multi_jot_all_log() {
  (cd frontend && echo "test" > file.txt && jj describe -m "Frontend commit" >/dev/null 2>&1)
  (cd backend && echo "test" > file.txt && jj describe -m "Backend commit" >/dev/null 2>&1)
  local output
  output=$(jjtask all log -r @ 2>/dev/null)
  assert_snapshot "multi_repo_all_log" "$output"
}

test_multi_workspace_hint() {
  (cd frontend && jjtask create @ "Frontend task" >/dev/null 2>&1)
  mkdir -p frontend/src
  local output
  output=$(cd frontend/src && jjtask find 2>/dev/null)
  assert_snapshot "multi_workspace_hint" "$output"
}

test_multi_complex_tasks() {
  jjtask create @ "ROOT: CI/CD pipeline" >/dev/null 2>&1
  jjtask create --draft @ "ROOT: Terraform modules" >/dev/null 2>&1
  jjtask create @ "ROOT: Integration tests" >/dev/null 2>&1

  (cd frontend && \
    jjtask create @ "FE: Auth login page" >/dev/null 2>&1 && \
    jjtask create --draft @ "FE: Dark mode toggle" >/dev/null 2>&1 && \
    jjtask create @ "FE: Error boundaries" >/dev/null 2>&1)

  (cd backend && \
    jjtask create @ "BE: User API endpoints" >/dev/null 2>&1 && \
    jjtask create --draft @ "BE: GraphQL schema" >/dev/null 2>&1 && \
    jjtask create @ "BE: Background jobs" >/dev/null 2>&1)

  local output
  output=$(jjtask find 2>/dev/null)
  assert_snapshot "multi_repo_complex" "$output"
}

test_parallel_start_shared() {
  jjtask create @ "Parent task" "## Context
Test parallel session" >/dev/null 2>&1
  local parent_id
  parent_id=$(get_task_id todo)
  [[ -n "$parent_id" ]] || return 1
  jj edit "$parent_id" >/dev/null 2>&1

  local output
  output=$(jjtask parallel-start --mode shared --agents 2 "$parent_id" 2>&1)
  assert_snapshot "parallel_start_shared" "$output"
}

test_parallel_start_workspace() {
  jjtask create @ "Parent task" "## Context
Test workspace mode" >/dev/null 2>&1
  local parent_id
  parent_id=$(get_task_id todo)
  [[ -n "$parent_id" ]] || return 1
  jj edit "$parent_id" >/dev/null 2>&1

  local output
  output=$(jjtask parallel-start --mode workspace --agents 2 "$parent_id" 2>&1)
  [[ -d ".jjtask-workspaces/agent-a" ]] || return 1
  [[ -d ".jjtask-workspaces/agent-b" ]] || return 1
  assert_snapshot "parallel_start_workspace" "$output"
}

test_parallel_status() {
  jjtask create @ "Parent task" >/dev/null 2>&1
  local parent_id
  parent_id=$(get_task_id todo)
  [[ -n "$parent_id" ]] || return 1
  jj edit "$parent_id" >/dev/null 2>&1

  jjtask parallel-start --mode shared --agents 2 "$parent_id" >/dev/null 2>&1
  local output
  output=$(jjtask parallel-status "$parent_id" 2>&1)
  assert_snapshot "parallel_status" "$output"
}

test_agent_context() {
  jjtask create @ "Parent task" >/dev/null 2>&1
  local parent_id
  parent_id=$(get_task_id todo)
  [[ -n "$parent_id" ]] || return 1
  jj edit "$parent_id" >/dev/null 2>&1

  jjtask parallel-start --mode shared --agents 2 "$parent_id" >/dev/null 2>&1
  local output
  output=$(jjtask agent-context agent-a 2>&1)
  assert_snapshot "agent_context" "$output"
}

test_parallel_stop() {
  jjtask create @ "Parent task" >/dev/null 2>&1
  local parent_id
  parent_id=$(get_task_id todo)
  [[ -n "$parent_id" ]] || return 1
  jj edit "$parent_id" >/dev/null 2>&1

  jjtask parallel-start --mode shared --agents 2 "$parent_id" >/dev/null 2>&1
  local output
  output=$(jjtask parallel-stop --force "$parent_id" 2>&1)
  assert_snapshot "parallel_stop" "$output"
}

test_parallel_start_invalid_mode() {
  jjtask create @ "Parent task" >/dev/null 2>&1
  local parent_id
  parent_id=$(get_task_id todo)
  [[ -n "$parent_id" ]] || return 1
  jj edit "$parent_id" >/dev/null 2>&1

  local output
  output=$(jjtask parallel-start --mode invalid "$parent_id" 2>&1) && return 1
  [[ "$output" == *"invalid mode"* ]]
}

test_agent_context_unknown() {
  jjtask create @ "Parent task" >/dev/null 2>&1
  local parent_id
  parent_id=$(get_task_id todo)
  [[ -n "$parent_id" ]] || return 1
  jj edit "$parent_id" >/dev/null 2>&1

  jjtask parallel-start --mode shared --agents 2 "$parent_id" >/dev/null 2>&1
  local output
  output=$(jjtask agent-context agent-xyz 2>&1) && return 1
  [[ "$output" == *"not in session"* ]]
}

# Main execution
echo "Running jjtask tests..."
echo ""

# Parse args
SEQUENTIAL=false
JOBS=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --sequential) SEQUENTIAL=true; shift ;;
    -j) JOBS="$2"; shift 2 ;;
    -j*) JOBS="${1#-j}"; shift ;;
    *) shift ;;
  esac
done

# Default to CPU core count
if [[ -z "$JOBS" ]]; then
  JOBS=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)
fi

FAILED=0
PASSED=0
TOTAL=${#TESTS[@]}

if [[ "$SEQUENTIAL" == "true" ]]; then
  # Sequential mode
  for test_spec in "${TESTS[@]}"; do
    IFS='|' read -r name func setup teardown <<< "$test_spec"
    setup="${setup:-setup_test_repo}"
    teardown="${teardown:-teardown_test_repo}"
    if run_one_test "$name" "$func" "$setup" "$teardown"; then
      PASSED=$((PASSED + 1))
    else
      FAILED=$((FAILED + 1))
    fi
  done
else
  # Parallel mode - run tests with job limit
  RESULT_DIR=$(mktemp -d)
  RUNNING=0
  NEXT=0

  while [[ $NEXT -lt $TOTAL ]] || [[ $RUNNING -gt 0 ]]; do
    # Launch jobs up to limit
    while [[ $RUNNING -lt $JOBS ]] && [[ $NEXT -lt $TOTAL ]]; do
      i=$NEXT
      test_spec="${TESTS[$i]}"
      IFS='|' read -r name func setup teardown <<< "$test_spec"
      setup="${setup:-setup_test_repo}"
      teardown="${teardown:-teardown_test_repo}"

      (
        if run_one_test "$name" "$func" "$setup" "$teardown" > "$RESULT_DIR/$i.out" 2>&1; then
          echo "0" > "$RESULT_DIR/$i.exit"
        else
          echo "1" > "$RESULT_DIR/$i.exit"
        fi
      ) &
      echo "$!" > "$RESULT_DIR/$i.pid"
      RUNNING=$((RUNNING + 1))
      NEXT=$((NEXT + 1))
    done

    # Wait for any job to finish
    if [[ $RUNNING -gt 0 ]]; then
      wait -n 2>/dev/null || sleep 0.1
      # Count still-running jobs
      RUNNING=0
      for ((j=0; j<NEXT; j++)); do
        if [[ -f "$RESULT_DIR/$j.pid" ]] && ! [[ -f "$RESULT_DIR/$j.exit" ]]; then
          pid=$(cat "$RESULT_DIR/$j.pid")
          if kill -0 "$pid" 2>/dev/null; then
            RUNNING=$((RUNNING + 1))
          fi
        fi
      done
    fi
  done

  # Print results in order
  for i in "${!TESTS[@]}"; do
    cat "$RESULT_DIR/$i.out"
    if [[ "$(cat "$RESULT_DIR/$i.exit")" == "0" ]]; then
      PASSED=$((PASSED + 1))
    else
      FAILED=$((FAILED + 1))
    fi
  done

  rm -rf "$RESULT_DIR"
fi

echo ""
echo "Results: $PASSED/$TOTAL passed"

if [[ $FAILED -gt 0 ]]; then
  echo -e "${RED}$FAILED test(s) failed${NC}"
  exit 1
else
  echo -e "${GREEN}All tests passed!${NC}"
  exit 0
fi
