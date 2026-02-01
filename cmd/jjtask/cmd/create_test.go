package cmd_test

import (
	"strings"
	"testing"
)

func TestCreateSuggestsChainingWhenWip(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "WIP task")
	wipID := repo.GetTaskID("todo")
	repo.Run("jjtask", "flag", "wip", "--rev", wipID)
	repo.Run("jj", "edit", wipID)

	// Create task with different parent
	output := repo.Run("jjtask", "create", "root()", "Other task")

	if !strings.Contains(output, "is a WIP task") {
		t.Error("expected WIP suggestion")
	}

}

func TestCreateNoSuggestionNotWip(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	output := repo.Run("jjtask", "create", "New task")

	if strings.Contains(output, "is a WIP task") {
		t.Error("should not suggest when @ not WIP")
	}

}

func TestCreateNoSuggestionParentAt(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "WIP task")
	wipID := repo.GetTaskID("todo")
	repo.Run("jjtask", "flag", "wip", "--rev", wipID)
	repo.Run("jj", "edit", wipID)

	output := repo.Run("jjtask", "create", "Child of wip")

	// Should NOT suggest since we're already using @ as parent
	if strings.Contains(output, "is a WIP task") {
		t.Error("should not suggest when parent is @")
	}

}

func TestCreateChainAt(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "--chain", "Task 1")
	repo.Run("jjtask", "create", "--chain", "Task 2")
	repo.Run("jjtask", "create", "--chain", "Task 3")

	// Verify Task 3 is descendant of Task 1 (chained, not sibling)
	task3Parent := repo.Run("jj", "log",
		"-r", "description(substring:\"Task 3\") & tasks()",
		"--no-graph", "-T", "parents.map(|p| p.description().first_line()).join(\",\")")

	if !strings.Contains(task3Parent, "Task 2") {
		t.Error("Task 3 should have Task 2 as parent")
	}

}

func TestCreateChainParent(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Parent")
	parentID := repo.GetTaskID("todo")

	repo.Run("jjtask", "create", "--chain", parentID, "Child 1")
	repo.Run("jjtask", "create", "--chain", parentID, "Child 2")

	// Child 2 should be child of Child 1, not sibling
	child2Parent := repo.Run("jj", "log",
		"-r", "description(substring:\"Child 2\") & tasks()",
		"--no-graph", "-T", "parents.map(|p| p.description().first_line()).join(\",\")")

	if !strings.Contains(child2Parent, "Child 1") {
		t.Error("Child 2 should have Child 1 as parent")
	}

}

func TestCreateDefaultDirect(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Parent")
	parentID := repo.GetTaskID("todo")

	repo.Run("jjtask", "create", parentID, "Child 1")
	repo.Run("jjtask", "create", parentID, "Child 2")

	// Child 2 should be sibling of Child 1 (both direct children of parent)
	child2Parent := repo.Run("jj", "log",
		"-r", "description(substring:\"Child 2\") & tasks()",
		"--no-graph", "-T", "parents.map(|p| p.description().first_line()).join(\",\")")

	if !strings.Contains(child2Parent, "Parent") {
		t.Error("Child 2 should have Parent as parent")
	}

	// Verify they're siblings
	siblingCount := strings.Count(
		repo.Run("jj", "log", "-r", "children("+parentID+") & tasks()", "--no-graph"),
		"[task:")

	if siblingCount < 2 {
		t.Error("expected 2+ siblings")
	}

}

func TestCreateWipSuggestionSnapshot(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "WIP task")
	wipID := repo.GetTaskID("todo")
	repo.Run("jjtask", "flag", "wip", "--rev", wipID)
	repo.Run("jj", "edit", wipID)

	// Create task with different parent - capture suggestion message format
	repo.Run("jjtask", "create", "root()", "Other task")

}
