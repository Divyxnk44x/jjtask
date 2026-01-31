package cmd_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Base timestamp for deterministic tests (same as jj's test suite)
var baseTimestamp = time.Date(2001, 2, 3, 4, 5, 6, 0, time.FixedZone("", 7*3600))

// makeBaseEnv creates the base environment for deterministic jj tests.
// Match jj's test environment for reproducible operation IDs and timestamps.
func makeBaseEnv(projectRoot, homeDir string) []string {
	binPath := filepath.Join(projectRoot, "bin", "jjtask-go")
	configPath := filepath.Join(projectRoot, "config")
	return []string{
		"JJ_CONFIG=" + configPath,
		"PATH=" + filepath.Dir(binPath) + ":" + os.Getenv("PATH"),
		"JJ_USER=Test User",
		"JJ_EMAIL=test.user@example.com",
		"JJ_OP_HOSTNAME=host.example.com",
		"JJ_OP_USERNAME=test-username",
		"JJ_TZ_OFFSET_MINS=420",
		"HOME=" + homeDir,
	}
}

// TestRepo provides a test jj repository with command logging
type TestRepo struct {
	t          *testing.T
	dir        string
	log        *bytes.Buffer
	baseEnv    []string
	cmdCounter int
	lastDAG    string // for deduplicating before/after logs
}

// SetupTestRepo creates a fresh jj repo for testing.
// Automatically saves a snapshot at test end using the test name.
func SetupTestRepo(t *testing.T) *TestRepo {
	t.Helper()

	dir, err := os.MkdirTemp("", "jjtask-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	repo := &TestRepo{
		t:          t,
		dir:        dir,
		log:        &bytes.Buffer{},
		cmdCounter: 0,
	}

	repo.baseEnv = makeBaseEnv(repo.findProjectRoot(), dir)

	// Auto-snapshot at test end using test name
	t.Cleanup(func() {
		repo.autoSnapshot(t)
		_ = os.RemoveAll(dir)
	})

	// Initialize jj repo (uses deterministic env)
	repo.runSilent("jj", "git", "init", "--colocate")

	return repo
}

// getEnvForCommand returns environment with deterministic seed/timestamp for this command
func (r *TestRepo) getEnvForCommand() []string {
	r.cmdCounter++
	ts := baseTimestamp.Add(time.Duration(r.cmdCounter) * time.Second)
	tsStr := ts.Format("2006-01-02T15:04:05-07:00")

	env := make([]string, len(r.baseEnv), len(r.baseEnv)+3)
	copy(env, r.baseEnv)
	env = append(env,
		fmt.Sprintf("JJ_RANDOMNESS_SEED=%d", r.cmdCounter),
		"JJ_TIMESTAMP="+tsStr,
		"JJ_OP_TIMESTAMP="+tsStr,
	)
	return env
}

func (r *TestRepo) findProjectRoot() string {
	// Walk up from test file to find project root (has go.mod)
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			r.t.Fatal("could not find project root")
		}
		dir = parent
	}
}

// Run executes a command and logs it to the trace
func (r *TestRepo) Run(name string, args ...string) string {
	r.t.Helper()
	return r.runInternal(name, args, true)
}

// RunExpectFail executes a command expected to fail, logs it
func (r *TestRepo) RunExpectFail(name string, args ...string) string {
	r.t.Helper()
	return r.runInternal(name, args, false)
}

// runSilent executes without logging (for setup)
func (r *TestRepo) runSilent(name string, args ...string) string {
	r.t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = r.dir
	cmd.Env = r.getEnvForCommand()
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
	return string(out)
}

func (r *TestRepo) runInternal(name string, args []string, expectSuccess bool) string {
	r.t.Helper()

	// Auto-log DAG before/after jjtask commands
	logDAG := name == "jjtask"
	if logDAG {
		r.logDAGWithLabel("before")
	}

	// Use absolute path for jjtask-go to avoid PATH issues in CI
	execName := name
	if name == "jjtask" {
		execName = filepath.Join(r.findProjectRoot(), "bin", "jjtask-go")
	}
	cmd := exec.Command(execName, args...)
	cmd.Dir = r.dir
	cmd.Env = r.getEnvForCommand()
	out, err := cmd.CombinedOutput()

	// Log to trace
	fmt.Fprintf(r.log, "$ %s %s\n", name, strings.Join(args, " "))
	fmt.Fprintf(r.log, "%s\n", out)

	if expectSuccess && err != nil {
		r.t.Fatalf("command failed: %s %v\nerror: %v\noutput: %s", name, args, err, out)
	}
	if !expectSuccess && err == nil {
		r.t.Fatalf("expected command to fail: %s %v\n%s", name, args, out)
	}

	// Auto-log DAG after
	if logDAG && expectSuccess {
		r.logDAGWithLabel("after")
	}

	return string(out)
}

// logDAGWithLabel appends jj log output to the trace with a label
// Skips if "before" matches previous "after" (avoids duplicate output)
func (r *TestRepo) logDAGWithLabel(label string) {
	cmd := exec.Command("jj", "log", "-r", "all()", "-T", "test_log")
	cmd.Dir = r.dir
	cmd.Env = r.getEnvForCommand()
	out, _ := cmd.CombinedOutput()
	dag := string(out)

	// Skip "before" if it matches the previous "after"
	if label == "before" && dag == r.lastDAG {
		return
	}

	fmt.Fprintf(r.log, "# %s\n%s\n", label, dag)
	r.lastDAG = dag
}

// WriteFile creates a file in the repo
func (r *TestRepo) WriteFile(name, content string) {
	r.t.Helper()
	path := filepath.Join(r.dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		r.t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		r.t.Fatalf("failed to write file: %v", err)
	}
	fmt.Fprintf(r.log, "# wrote %s (%d bytes)\n\n", name, len(content))
}

// GetTaskID finds a task by flag and returns its shortest change ID
func (r *TestRepo) GetTaskID(flag string) string {
	r.t.Helper()
	out := r.runSilent("jj", "log", "-r", fmt.Sprintf("tasks_%s()", flag),
		"--no-graph", "-T", "change_id.shortest()")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 || lines[0] == "" {
		r.t.Fatalf("no task found with flag %s", flag)
	}
	return lines[0]
}

// RunWithStdin executes a command with input piped to stdin
func (r *TestRepo) RunWithStdin(input, name string, args ...string) string {
	r.t.Helper()

	execName := name
	if name == "jjtask" {
		execName = filepath.Join(r.findProjectRoot(), "bin", "jjtask-go")
	}
	cmd := exec.Command(execName, args...)
	cmd.Dir = r.dir
	cmd.Env = r.getEnvForCommand()
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.CombinedOutput()

	fmt.Fprintf(r.log, "$ echo '...' | %s %s\n", name, strings.Join(args, " "))
	fmt.Fprintf(r.log, "%s\n", out)

	if err != nil {
		r.t.Fatalf("command failed: %s %v\nerror: %v\noutput: %s", name, args, err, out)
	}

	return string(out)
}

// Trace returns the full command log
func (r *TestRepo) Trace() string {
	return r.log.String()
}

// autoSnapshot is called automatically at test end via Cleanup.
// Converts TestName to snake_case for the snapshot filename.
func (r *TestRepo) autoSnapshot(t *testing.T) {
	name := testNameToSnakeCase(t.Name())
	r.snapshotWithName(t, name)
}

// Snapshot compares the command trace against a golden file (legacy API)
func (r *TestRepo) Snapshot(t *testing.T, name string) {
	t.Helper()
	r.snapshotWithName(t, name)
}

func (r *TestRepo) snapshotWithName(t *testing.T, name string) {
	trace := r.normalizeTrace(r.log.String())

	snapshotDir := filepath.Join(r.findProjectRoot(), "test", "snapshots_go")
	snapshotFile := filepath.Join(snapshotDir, name+".txt")

	if os.Getenv("SNAPSHOT_UPDATE") != "" {
		_ = os.MkdirAll(snapshotDir, 0o755)
		if err := os.WriteFile(snapshotFile, []byte(trace), 0o644); err != nil {
			t.Fatalf("failed to write snapshot: %v", err)
		}
		return
	}

	expected, err := os.ReadFile(snapshotFile)
	if err != nil {
		t.Fatalf("snapshot not found: %s\nRun with SNAPSHOT_UPDATE=1 to create\n\nActual output:\n%s", snapshotFile, trace)
	}

	if string(expected) != trace {
		t.Errorf("snapshot mismatch: %s\n\nExpected:\n%s\n\nActual:\n%s", name, expected, trace)
	}
}

// testNameToSnakeCase converts "TestFooBar" to "foo_bar"
func testNameToSnakeCase(name string) string {
	name = strings.TrimPrefix(name, "Test")
	// Convert CamelCase to snake_case
	var result strings.Builder
	for i, r := range name {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result.WriteByte('_')
			}
			result.WriteByte(byte(r) + 32) // lowercase
		} else {
			result.WriteByte(byte(r))
		}
	}
	return result.String()
}

// normalizeTrace replaces variable content for deterministic snapshots
func (r *TestRepo) normalizeTrace(s string) string {
	// Replace temp directory paths
	s = strings.ReplaceAll(s, r.dir, "$REPO")
	return s
}
