package cmd_test

import (
	"strings"
	"testing"
)

func TestCreateTask(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Test task", "Test description")

	output := repo.Run("jjtask", "find")
	if !strings.Contains(output, "Test task") {
		t.Error("task not found in output")
	}

}

func TestCreateDraft(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "--draft", "@", "Draft task")

	output := repo.Run("jjtask", "find", "--status", "all")
	if !strings.Contains(output, "[task:draft]") {
		t.Error("draft flag not found")
	}

}

func TestFlagUpdatesStatus(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Test task")
	taskID := repo.GetTaskID("todo")
	repo.Run("jjtask", "flag", "wip", "--rev", taskID)

	output := repo.Run("jjtask", "find")
	if !strings.Contains(output, "[task:wip]") {
		t.Error("wip flag not found")
	}

}

func TestFindShowsTasks(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Task A")
	repo.Run("jjtask", "create", "Task B")

	output := repo.Run("jjtask", "find")
	if !strings.Contains(output, "Task A") || !strings.Contains(output, "Task B") {
		t.Error("tasks not found in output")
	}

}

func TestFindRevsetFiltersTasks(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "My task")
	repo.Run("jj", "new", "-m", "Regular commit")

	output := repo.Run("jjtask", "find", "-r", "all()")
	if !strings.Contains(output, "My task") {
		t.Error("task not found in output")
	}
	if strings.Contains(output, "Regular commit") {
		t.Error("regular commit should not appear in task find")
	}

}

func TestShowDesc(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Test title", "Test body content")
	taskID := repo.GetTaskID("todo")
	repo.Run("jj", "edit", taskID)

	output := repo.Run("jjtask", "show-desc")
	if !strings.Contains(output, "Test title") || !strings.Contains(output, "Test body content") {
		t.Error("description content not found")
	}

}

func TestParallelCreatesSiblings(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "parallel", "Task A", "Task B", "Task C")

	output := repo.Run("jjtask", "find")
	if !strings.Contains(output, "Task A") ||
		!strings.Contains(output, "Task B") ||
		!strings.Contains(output, "Task C") {
		t.Error("parallel tasks not found")
	}

}

func TestPrime(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	output := repo.Run("jjtask", "prime")
	if output == "" {
		t.Error("prime produced no output")
	}

}

func TestCheckpoint(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	output := repo.Run("jjtask", "checkpoint", "-m", "test-checkpoint")
	if output == "" {
		t.Error("checkpoint produced no output")
	}

}

func TestDescTransform(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Original title", "## Context\nSome context here")
	taskID := repo.GetTaskID("todo")
	repo.Run("jjtask", "desc-transform", "--rev", taskID, "sed", "s/Original/Modified/")

	output := repo.Run("jjtask", "show-desc", "--rev", taskID)
	if !strings.Contains(output, "Modified title") {
		t.Error("transform not applied")
	}

}

func TestDescTransformError(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Test title")
	taskID := repo.GetTaskID("todo")

	output := repo.RunExpectFail("jjtask", "desc-transform", "--rev", taskID, "nonexistent-cmd-xyz")
	if output == "" {
		t.Error("expected error output")
	}

}

func TestDescTransformMultiline(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Test task", "## Context\nOriginal content")
	taskID := repo.GetTaskID("todo")

	// Test multiline replacement with \n in replacement text
	repo.Run("jjtask", "desc-transform", "--rev", taskID, `s/Original content/New line 1\nNew line 2/`)

	output := repo.Run("jjtask", "show-desc", "--rev", taskID)
	if !strings.Contains(output, "New line 1\nNew line 2") {
		t.Errorf("multiline replacement failed, got: %s", output)
	}
}

func TestDescTransformGlobalFlag(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "foo bar foo", "foo baz foo")
	taskID := repo.GetTaskID("todo")

	// Test global replacement
	repo.Run("jjtask", "desc-transform", "--rev", taskID, "s/foo/XXX/g")

	output := repo.Run("jjtask", "show-desc", "--rev", taskID)
	if strings.Contains(output, "foo") {
		t.Errorf("global replacement failed, still contains 'foo': %s", output)
	}
	if !strings.Contains(output, "XXX bar XXX") {
		t.Errorf("global replacement failed, expected 'XXX bar XXX' in: %s", output)
	}
}

func TestDescTransformStdin(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Original", "old content")
	taskID := repo.GetTaskID("todo")

	// Test --stdin to replace entire description
	repo.RunWithStdin("[task:todo] New title\n\nMultiline\ncontent\nhere", "jjtask", "desc-transform", "--rev", taskID, "--stdin")

	output := repo.Run("jjtask", "show-desc", "--rev", taskID)
	if !strings.Contains(output, "New title") {
		t.Errorf("stdin replacement failed, expected 'New title' in: %s", output)
	}
	if !strings.Contains(output, "Multiline\ncontent\nhere") {
		t.Errorf("stdin multiline failed, got: %s", output)
	}
}

func TestConfigTaskLogDiffStats(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.WriteFile("testfile.txt", "test content")
	repo.Run("jj", "describe", "-m", "Test commit with changes")

	output := repo.Run("jj", "log", "-r", "@", "--no-graph", "-T", "task_log")
	if output == "" {
		t.Error("task_log template produced no output")
	}

}

func TestConfigTaskLogShortDesc(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Short title", "## Context\nThis is a longer description\nwith multiple lines")
	taskID := repo.GetTaskID("todo")

	output := repo.Run("jj", "log", "-r", taskID, "--no-graph", "-T", "task_log")
	if !strings.Contains(output, "Short title") {
		t.Error("title not found in task_log output")
	}

}

func TestFlagPositionalRev(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Test task")
	taskID := repo.GetTaskID("todo")

	// Test positional rev: "flag REV STATUS" instead of "flag STATUS --rev REV"
	repo.Run("jjtask", "flag", taskID, "wip")

	output := repo.Run("jjtask", "find")
	if !strings.Contains(output, "[task:wip]") {
		t.Error("wip flag not found after positional rev usage")
	}
}

func TestDescTransformPositionalRev(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Original title", "Some context")
	taskID := repo.GetTaskID("todo")

	// Test positional rev: "desc-transform REV 's/...'" instead of "desc-transform -r REV 's/...'"
	repo.Run("jjtask", "desc-transform", taskID, "s/Original/Modified/")

	output := repo.Run("jjtask", "show-desc", taskID)
	if !strings.Contains(output, "Modified title") {
		t.Errorf("transform with positional rev not applied, got: %s", output)
	}
}

func TestDescTransformAlternateDelimiter(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Task with path/in/title", "More context")
	taskID := repo.GetTaskID("todo")

	// Use | as delimiter to avoid escaping /
	repo.Run("jjtask", "desc-transform", taskID, "s|path/in/title|new/path|")

	output := repo.Run("jjtask", "show-desc", taskID)
	if !strings.Contains(output, "new/path") {
		t.Errorf("alternate delimiter transform failed, got: %s", output)
	}
}
