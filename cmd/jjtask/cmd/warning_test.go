package cmd_test

import (
	"strings"
	"testing"
)

func TestFlagDoneWarnsWhenAtHasDiff(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Task to complete")
	taskID := repo.GetTaskID("todo")
	// Make changes in @ (not in task)
	repo.WriteFile("workfile.txt", "work")

	output := repo.Run("jjtask", "flag", "done", "--rev", taskID)

	if !strings.Contains(output, "Working copy (@) has uncommitted changes") {
		t.Error("expected warning about uncommitted changes")
	}
	if !strings.Contains(output, "Were any of these changes part of this task") {
		t.Error("expected question about changes")
	}

}

func TestFlagWipWarnsWhenAtHasDiff(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Task to start")
	taskID := repo.GetTaskID("todo")
	repo.WriteFile("workfile.txt", "work")

	output := repo.Run("jjtask", "flag", "wip", "--rev", taskID)

	if !strings.Contains(output, "Working copy (@) has uncommitted changes") {
		t.Error("expected warning about uncommitted changes")
	}
	if !strings.Contains(output, "Were any of these changes part of this task") {
		t.Error("expected question about changes")
	}

}

func TestFlagNoWarningWhenAtIsTask(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Task to complete")
	taskID := repo.GetTaskID("todo")
	repo.Run("jj", "edit", taskID)
	repo.WriteFile("workfile.txt", "work")

	output := repo.Run("jjtask", "flag", "done")

	if strings.Contains(output, "Working copy (@) has uncommitted changes") {
		t.Error("should not warn when @ is the task")
	}

}

func TestFlagNoWarningWhenClean(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Task to complete")
	taskID := repo.GetTaskID("todo")

	output := repo.Run("jjtask", "flag", "done", "--rev", taskID)

	if strings.Contains(output, "Working copy (@) has uncommitted changes") {
		t.Error("should not warn when clean")
	}

}

func TestFlagDoneWarnsPendingChildren(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Parent task")
	parentID := repo.GetTaskID("todo")
	repo.Run("jjtask", "create", parentID, "Child task")

	output := repo.Run("jjtask", "flag", "done", "--rev", parentID)

	if !strings.Contains(output, "pending children") {
		t.Error("expected warning about pending children")
	}
	if !strings.Contains(output, "Child task") {
		t.Error("expected child task in warning")
	}

}

func TestFlagDoneNoWarningChildrenDone(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Parent task")
	parentID := repo.GetTaskID("todo")
	repo.Run("jjtask", "create", parentID, "Child task")

	// Mark child done first
	output := repo.Run("jj", "log", "-r", "children("+parentID+") & tasks()",
		"--no-graph", "-T", "change_id.shortest()")
	childID := strings.TrimSpace(strings.Split(output, "\n")[0])
	repo.Run("jjtask", "flag", "done", "--rev", childID)

	output = repo.Run("jjtask", "flag", "done", "--rev", parentID)

	if strings.Contains(output, "pending children") {
		t.Error("should not warn when children done")
	}

}

func TestFlagWipWarnsBlockedAncestor(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Parent task")
	parentID := repo.GetTaskID("todo")
	repo.Run("jjtask", "flag", "blocked", "--rev", parentID)
	repo.Run("jjtask", "create", parentID, "Child task")

	// Find child
	output := repo.Run("jj", "log", "-r", "children("+parentID+") & tasks()",
		"--no-graph", "-T", "change_id.shortest()")
	childID := strings.TrimSpace(strings.Split(output, "\n")[0])

	output = repo.Run("jjtask", "flag", "wip", "--rev", childID)

	if !strings.Contains(output, "Ancestor task is blocked") {
		t.Error("expected blocked ancestor warning")
	}

}

func TestFlagWipNoWarningNotBlocked(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Parent task")
	parentID := repo.GetTaskID("todo")
	repo.Run("jjtask", "create", parentID, "Child task")

	// Find child
	output := repo.Run("jj", "log", "-r", "children("+parentID+") & tasks()",
		"--no-graph", "-T", "change_id.shortest()")
	childID := strings.TrimSpace(strings.Split(output, "\n")[0])

	output = repo.Run("jjtask", "flag", "wip", "--rev", childID)

	if strings.Contains(output, "Ancestor task is blocked") {
		t.Error("should not warn when ancestor not blocked")
	}

}

func TestFlagWipWarnsExistingWip(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Task A")
	repo.Run("jjtask", "create", "@", "Task B")

	// Find both tasks
	taskA := repo.Run("jj", "log", "-r", "tasks() & description(substring:\"Task A\")",
		"--no-graph", "-T", "change_id.shortest()")
	taskA = strings.TrimSpace(taskA)
	taskB := repo.Run("jj", "log", "-r", "tasks() & description(substring:\"Task B\")",
		"--no-graph", "-T", "change_id.shortest()")
	taskB = strings.TrimSpace(taskB)

	// Mark A as wip
	repo.Run("jjtask", "flag", "wip", "--rev", taskA)

	// Try to mark B as wip
	output := repo.Run("jjtask", "flag", "wip", "--rev", taskB)

	if !strings.Contains(output, "Another WIP task exists") {
		t.Error("expected existing wip warning")
	}

}

func TestFlagWipNoWarningSameChain(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Parent task")
	parentID := repo.GetTaskID("todo")
	repo.Run("jjtask", "create", parentID, "Child task")

	// Find child
	childOutput := repo.Run("jj", "log", "-r", "children("+parentID+") & tasks()",
		"--no-graph", "-T", "change_id.shortest()")
	childID := strings.TrimSpace(strings.Split(childOutput, "\n")[0])

	// Mark parent as wip
	repo.Run("jjtask", "flag", "wip", "--rev", parentID)

	// Mark child as wip - should NOT warn (same chain)
	output := repo.Run("jjtask", "flag", "wip", "--rev", childID)

	if strings.Contains(output, "Another WIP task exists") {
		t.Error("should not warn for same chain")
	}

}

func TestFlagWipWarnsDoneAncestor(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Parent task")
	parentID := repo.GetTaskID("todo")
	repo.Run("jjtask", "create", parentID, "Child task")

	// Find child
	output := repo.Run("jj", "log", "-r", "children("+parentID+") & tasks()",
		"--no-graph", "-T", "change_id.shortest()")
	childID := strings.TrimSpace(strings.Split(output, "\n")[0])

	// Mark parent as done
	repo.Run("jjtask", "flag", "done", "--rev", parentID)

	// Try to mark child as wip - should warn about done ancestor
	output = repo.Run("jjtask", "flag", "wip", "--rev", childID)

	if !strings.Contains(output, "Ancestor task is already done") {
		t.Error("expected done ancestor warning")
	}
	if !strings.Contains(output, "Parent task") {
		t.Error("expected parent task in warning")
	}

}

func TestFlagDoneWarnsEmpty(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Empty task")
	taskID := repo.GetTaskID("todo")

	output := repo.Run("jjtask", "flag", "done", "--rev", taskID)

	if !strings.Contains(output, "Task is empty") {
		t.Error("expected empty task warning")
	}

}

func TestFlagDoneNoWarningWithContent(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Task with work")
	taskID := repo.GetTaskID("todo")
	repo.Run("jj", "edit", taskID)
	repo.WriteFile("workfile.txt", "actual work")
	repo.Run("jj", "status") // Trigger snapshot

	output := repo.Run("jjtask", "flag", "done")

	if strings.Contains(output, "Task is empty") {
		t.Error("should not warn with content")
	}

}
