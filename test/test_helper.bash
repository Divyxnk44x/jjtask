#!/usr/bin/env bash
# Test helper functions for jjtask tests

JJTASK_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export JJTASK_ROOT

# Set up test environment
export PATH="$JJTASK_ROOT/bin:$PATH"
export JJ_CONFIG="$JJTASK_ROOT/config/conf.d"

# Create isolated test repo
setup_test_repo() {
  TEST_REPO=$(mktemp -d)
  export TEST_REPO
  cd "$TEST_REPO" || exit 1

  # Initialize jj repo
  jj git init --colocate >/dev/null 2>&1

  # Configure for tests - use global to avoid warnings
  export JJ_USER="Test User"
  export JJ_EMAIL="test@example.com"
}

# Clean up test repo
teardown_test_repo() {
  if [[ -n "$TEST_REPO" && -d "$TEST_REPO" ]]; then
    rm -rf "$TEST_REPO"
  fi
}

# Assert command succeeds
assert_success() {
  if [[ $? -ne 0 ]]; then
    echo "FAIL: Expected success but got failure"
    return 1
  fi
}

# Assert command fails
assert_failure() {
  if [[ $? -eq 0 ]]; then
    echo "FAIL: Expected failure but got success"
    return 1
  fi
}

# Assert output contains string
assert_output_contains() {
  local expected="$1"
  local actual="$2"
  if [[ "$actual" != *"$expected"* ]]; then
    echo "FAIL: Expected output to contain '$expected'"
    echo "Actual: $actual"
    return 1
  fi
}

# Assert output equals string
assert_output_equals() {
  local expected="$1"
  local actual="$2"
  if [[ "$actual" != "$expected" ]]; then
    echo "FAIL: Expected '$expected'"
    echo "Actual: '$actual'"
    return 1
  fi
}

# Check if a task with given flag exists in description
# Uses all() since tasks created with --no-edit aren't in @'s ancestry
has_task_with_flag() {
  local flag="$1"
  jj log -r "all()" --no-graph -T 'description' 2>/dev/null | grep -q "\[task:$flag\]"
}

# Get first task ID with given flag
# Parse output to avoid wrapper blocking on [task: in template
get_task_id() {
  local flag="$1"
  # Get all revisions and grep for the flag, then extract ID
  jj log -r "all()" --no-graph -T 'change_id.shortest(8) ++ " " ++ description.first_line() ++ "\n"' 2>/dev/null | \
    grep "\[task:$flag\]" | head -1 | cut -d' ' -f1
}

# Create multi-repo test environment
# Structure: root/ with frontend/ and backend/ jj repos
setup_multi_repo() {
  TEST_REPO=$(mktemp -d)
  export TEST_REPO
  export WORKSPACE_ROOT="$TEST_REPO"
  cd "$TEST_REPO" || exit 1

  # Create nested repos
  mkdir -p frontend backend

  jj git init --colocate >/dev/null 2>&1
  (cd frontend && jj git init --colocate >/dev/null 2>&1)
  (cd backend && jj git init --colocate >/dev/null 2>&1)

  # Ignore workspace config in root repo (commit it so it doesn't show in diff)
  echo ".jj-workspaces.yaml" > .gitignore
  jj commit -m "Add .gitignore" >/dev/null 2>&1

  # Create workspace config
  cat > .jj-workspaces.yaml <<EOF
repos:
  - path: .
    name: root
  - path: frontend
    name: frontend
  - path: backend
    name: backend
EOF

  export JJ_USER="Test User"
  export JJ_EMAIL="test@example.com"
}

# Clean up multi-repo test
teardown_multi_repo() {
  teardown_test_repo
}

# Snapshot testing support
SNAPSHOTS_DIR="$JJTASK_ROOT/test/snapshots"

# Normalize output for snapshot comparison
# Replaces variable parts with stable placeholders
normalize_output() {
  sed -E \
    -e 's/tmp\.[A-Za-z0-9]+/TMPDIR/g' \
    -e 's/cwd: [^ ]+ \|/cwd: TMPDIR |/g' \
    -e 's/repo: [^ ]+ \|/repo: TMPDIR |/g' \
    -e 's/[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}/TIMESTAMP/g' \
    -e 's/[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}([+-][0-9]{2}:[0-9]{2}|Z)/TIMESTAMP/g' \
    -e 's|/tmp/[^/ ]+|/tmp/TMPDIR|g' \
    -e 's|/var/folders/[^/]+/[^/]+/[^/]+|/var/folders/TMPDIR|g' \
    -e 's/^(@|○|│)  [a-z]{1,8} /\1  ID /g' \
    -e 's/^(@|○|│)  [a-z]{8} *$/\1  ID/g' \
    -e 's/│ (○)  [a-z]{1,8} /│ \1  ID /g' \
    -e 's/^[a-z]{1,12} (\(empty\)|Test|\[task|\[todo|\[draft|\[wip|\[done|\[blocked)/ID \1/g' \
    -e 's/^([a-z]{1,12}) \(([a-z]{1,8})\)/ID (ID)/g' \
    -e 's/(ID) \(([a-z]{1,8})\)/ID (ID)/g' \
    -e 's/│ ○  (ID) \(([a-z]{1,8})\)/│ ○  ID (ID)/g' \
    -e 's/TIMESTAMP [a-z0-9]{1,8}$/TIMESTAMP ID/g' \
    -e 's/operation: [a-f0-9]{12}/operation: OPID/g' \
    -e 's/op restore [a-f0-9]{12}/op restore OPID/g' \
    -e 's/to hoist: [a-z ]+$/to hoist: ID/g' \
    -e 's/Rebasing [a-z]{1,8} /Rebasing ID /g' \
    -e 's/^  [a-z]{1,8} already/  ID already/g' \
    -e 's/Working copy .* at: [a-z]+ [a-f0-9]+/Working copy at: ID HASH/g' \
    -e 's/Parent commit .* : [a-z]+ [a-f0-9]+/Parent commit: ID HASH/g' \
    -e 's/Started: [0-9]+[hmd] ago/Started: TIME ago/g' \
    -e 's/started: TIMESTAMP/started: TIMESTAMP/g' \
    -e 's/Parent: [a-z]+ /Parent: ID /g' \
    -e 's/← [a-z]{1,8} \[task/← ID [task/g' \
    -e 's/Task: [a-z]{1,8}$/Task: ID/g' \
    -e 's/Session: [a-z]{1,12} \[task/Session: ID [task/g' \
    -e 's/Created new commit [a-z]+ [a-f0-9]+/Created new commit ID HASH/g' \
    -e 's/Rebased [0-9]+ descendant/Rebased N descendant/g'
}

# Assert output matches snapshot
# Usage: assert_snapshot "test_name" "$output"
# Set SNAPSHOT_UPDATE=1 to regenerate snapshots
assert_snapshot() {
  local name="$1"
  local actual="$2"
  local snapshot_file="$SNAPSHOTS_DIR/${name}.txt"
  local normalized
  normalized=$(echo "$actual" | normalize_output)

  if [[ "${SNAPSHOT_UPDATE:-}" == "1" ]]; then
    mkdir -p "$SNAPSHOTS_DIR"
    echo "$normalized" > "$snapshot_file"
    echo "  Updated snapshot: $name"
    return 0
  fi

  if [[ ! -f "$snapshot_file" ]]; then
    echo "FAIL: Snapshot not found: $snapshot_file"
    echo "Run with SNAPSHOT_UPDATE=1 to create"
    echo "Actual (normalized):"
    echo "$normalized"
    return 1
  fi

  local expected
  expected=$(cat "$snapshot_file")

  if [[ "$normalized" != "$expected" ]]; then
    echo "FAIL: Snapshot mismatch for $name"
    echo "--- Expected ---"
    echo "$expected"
    echo "--- Actual ---"
    echo "$normalized"
    echo "--- Diff ---"
    diff <(echo "$expected") <(echo "$normalized") || true
    return 1
  fi
}
