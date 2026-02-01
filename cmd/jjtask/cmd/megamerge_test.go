package cmd_test

import (
	"strings"
	"testing"
)

func TestWipSingleTask(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Single task")
	taskID := repo.GetTaskID("todo")

	repo.Run("jjtask", "wip", taskID)

	// Task should have wip flag
	desc := repo.runSilent("jj", "log", "-r", taskID, "--no-graph", "-T", "description")
	if !strings.Contains(desc, "[task:wip]") {
		t.Error("task should have wip flag")
	}

	// Task should be a parent of @
	parents := repo.runSilent("jj", "log", "-r", "parents(@)", "--no-graph", "-T", `change_id.shortest() ++ "\n"`)
	if !strings.Contains(parents, strings.TrimSpace(repo.runSilent("jj", "log", "-r", taskID, "--no-graph", "-T", "change_id.shortest()"))) {
		t.Error("task should be parent of @")
	}

	output := repo.Run("jjtask", "find")
	if !strings.Contains(output, "[task:wip]") {
		t.Error("task should appear in find with wip flag")
	}
}

func TestWipMultipleTasks(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Task A")
	taskA := repo.GetTaskID("todo")

	// Create second task as sibling (not child)
	repo.Run("jjtask", "create", "root()", "Task B")
	taskB := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task B")`,
		"--no-graph", "-T", "change_id.shortest()"))

	// Mark first as wip
	repo.Run("jjtask", "wip", taskA)

	// Mark second as wip - should create merge
	repo.Run("jjtask", "wip", taskB)

	// @ should have multiple parents (merge commit)
	parentOut := repo.runSilent("jj", "log", "-r", "parents(@)", "--no-graph", "-T", `change_id.shortest() ++ "\n"`)
	parentCount := len(strings.Split(strings.TrimSpace(parentOut), "\n"))

	if parentCount < 2 {
		t.Errorf("expected 2+ parents (merge), got %d", parentCount)
	}
}

func TestDoneWithContent(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Task with content")
	taskID := repo.GetTaskID("todo")

	// Edit into task and add content
	repo.Run("jj", "edit", taskID)
	repo.WriteFile("workfile.txt", "actual work")
	repo.Run("jj", "status") // Trigger snapshot

	// Mark as wip first (to make it active)
	repo.Run("jjtask", "wip")

	// Now mark done
	repo.Run("jjtask", "done")

	// Task should be marked done
	desc := repo.runSilent("jj", "log", "-r", taskID, "--no-graph", "-T", "description")
	if !strings.Contains(desc, "[task:done]") {
		t.Errorf("expected [task:done], got: %s", desc)
	}
}

func TestDoneEmptyTask(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Empty planning task")
	taskID := repo.GetTaskID("todo")

	// Mark wip without adding content
	repo.Run("jjtask", "wip", taskID)

	// Mark done (empty) - need to specify the task since @ is child of task
	repo.Run("jjtask", "done", taskID)

	// Task should be marked done
	desc := repo.runSilent("jj", "log", "-r", taskID, "--no-graph", "-T", "description")
	if !strings.Contains(desc, "[task:done]") {
		t.Errorf("expected [task:done], got: %s", desc)
	}
}

func TestDropMarksStandby(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Task to drop")
	taskID := repo.GetTaskID("todo")

	// Mark wip first
	repo.Run("jjtask", "wip", taskID)

	// Drop it
	output := repo.Run("jjtask", "drop", taskID)

	if !strings.Contains(output, "standby") {
		t.Errorf("expected standby message, got: %s", output)
	}

	// Verify it's marked standby
	desc := repo.runSilent("jj", "log", "-r", taskID, "--no-graph", "-T", "description")
	if !strings.Contains(desc, "[task:standby]") {
		t.Error("task should be marked standby")
	}
}

func TestDropAbandon(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Task to abandon")
	taskID := repo.GetTaskID("todo")

	output := repo.Run("jjtask", "drop", "--abandon", taskID)

	if !strings.Contains(output, "Abandoned") {
		t.Errorf("expected Abandoned message, got: %s", output)
	}

	// Task should no longer exist
	check := repo.RunExpectFail("jj", "log", "-r", taskID, "--no-graph")
	if !strings.Contains(check, "doesn't exist") && !strings.Contains(check, "No matching revisions") {
		t.Error("task should not exist after abandon")
	}
}

func TestDoneLinearizesFromMerge(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	// Create base commit (not on root to avoid git limitations)
	repo.WriteFile("base.txt", "base")
	repo.Run("jj", "describe", "-m", "Base commit")

	// Create Task A as child of base
	repo.Run("jjtask", "create", "Task A")
	taskA := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task A")`,
		"--no-graph", "-T", "change_id.shortest()"))

	// Create Task B as sibling of Task A (also child of base)
	baseID := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `description(substring:"Base commit")`,
		"--no-graph", "-T", "change_id.shortest()"))
	repo.Run("jjtask", "create", baseID, "Task B")
	taskB := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task B")`,
		"--no-graph", "-T", "change_id.shortest()"))

	// Mark both as WIP - creates merge
	repo.Run("jjtask", "wip", taskA)
	repo.Run("jjtask", "wip", taskB)

	// Verify merge (2+ parents)
	parentOut := repo.runSilent("jj", "log", "-r", "parents(@)", "--no-graph", "-T", `change_id.shortest() ++ "\n"`)
	parentCount := len(strings.Split(strings.TrimSpace(parentOut), "\n"))
	if parentCount < 2 {
		t.Fatalf("expected merge with 2+ parents, got %d", parentCount)
	}

	// Mark task A as done - should linearize
	repo.Run("jjtask", "done", taskA)

	// Task A should be ancestor of Task B now
	isAncestor := repo.runSilent("jj", "log", "-r", taskA+"::"+taskB, "--no-graph", "-T", "change_id.shortest()")
	if isAncestor == "" || !strings.Contains(isAncestor, taskA) {
		t.Errorf("done task A should be ancestor of task B after linearization, got: %s", isAncestor)
	}

	// @ should have single parent now (linear chain)
	parentOut = repo.runSilent("jj", "log", "-r", "parents(@)", "--no-graph", "-T", `change_id.shortest() ++ "\n"`)
	lines := strings.Split(strings.TrimSpace(parentOut), "\n")
	var nonEmpty []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmpty = append(nonEmpty, line)
		}
	}
	if len(nonEmpty) != 1 {
		t.Errorf("expected single parent after linearization, got %d", len(nonEmpty))
	}
}

func TestSquashFlatten(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	// Create a common base (not root) to avoid Git merge-with-root issue
	repo.WriteFile("base.txt", "base")
	repo.Run("jj", "describe", "-m", "Base commit")

	repo.Run("jjtask", "create", "Task A")
	taskA := repo.GetTaskID("todo")

	// Add content to task A
	repo.Run("jj", "edit", taskA)
	repo.WriteFile("file_a.txt", "content A")
	repo.Run("jj", "status")

	// Create Task B as sibling (same parent as Task A)
	baseID := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `description(substring:"Base commit")`,
		"--no-graph", "-T", "change_id.shortest()"))
	repo.Run("jjtask", "create", baseID, "Task B")
	taskB := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task B")`,
		"--no-graph", "-T", "change_id.shortest()"))

	// Add content to task B
	repo.Run("jj", "edit", taskB)
	repo.WriteFile("file_b.txt", "content B")
	repo.Run("jj", "status")

	// Mark both as wip to create merge
	repo.Run("jjtask", "wip", taskA)
	repo.Run("jjtask", "wip", taskB)

	// Verify we have a merge
	parentOut := repo.runSilent("jj", "log", "-r", "parents(@)", "--no-graph", "-T", `change_id.shortest() ++ "\n"`)
	parentCount := len(strings.Split(strings.TrimSpace(parentOut), "\n"))
	if parentCount < 2 {
		t.Fatalf("expected merge with 2+ parents before squash, got %d", parentCount)
	}

	// Squash
	output := repo.Run("jjtask", "squash")

	if !strings.Contains(output, "Squashed") {
		t.Errorf("expected squash message, got: %s", output)
	}

	// @ should now have single parent (linear)
	parentOut = repo.runSilent("jj", "log", "-r", "parents(@)", "--no-graph", "-T", `change_id.shortest() ++ "\n"`)
	lines := strings.Split(strings.TrimSpace(parentOut), "\n")
	// Filter empty lines
	var nonEmpty []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmpty = append(nonEmpty, line)
		}
	}
	if len(nonEmpty) > 1 {
		t.Errorf("expected single parent after squash, got %d", len(nonEmpty))
	}
}

func TestSquashSingleParent(t *testing.T) {
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Single task")
	taskID := repo.GetTaskID("todo")

	repo.Run("jjtask", "wip", taskID)

	output := repo.Run("jjtask", "squash")

	if !strings.Contains(output, "Only one parent") && !strings.Contains(output, "No parents") && !strings.Contains(output, "single parent") {
		t.Errorf("expected single parent message, got: %s", output)
	}
}

// === Edge case tests discovered during development ===

func TestWipWhenAtIsTask(t *testing.T) {
	// When @ IS the task itself, wip should just mark it, not try to add as parent
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Task")
	taskID := repo.GetTaskID("todo")

	// Edit into the task so @ = task
	repo.Run("jj", "edit", taskID)

	// Mark as wip - should not error "cannot rebase onto itself"
	repo.Run("jjtask", "wip")

	desc := repo.runSilent("jj", "log", "-r", "@", "--no-graph", "-T", "description")
	if !strings.Contains(desc, "[task:wip]") {
		t.Errorf("expected [task:wip], got: %s", desc)
	}
}

func TestDoneWhenAtIsTask(t *testing.T) {
	// When @ IS the task, done should just mark it
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.Run("jjtask", "create", "Task")
	taskID := repo.GetTaskID("todo")

	repo.Run("jj", "edit", taskID)
	repo.WriteFile("work.txt", "content")
	repo.Run("jjtask", "wip")
	repo.Run("jjtask", "done")

	desc := repo.runSilent("jj", "log", "-r", "@", "--no-graph", "-T", "description")
	if !strings.Contains(desc, "[task:done]") {
		t.Errorf("expected [task:done], got: %s", desc)
	}
}

func TestWipMultipleRevsAtOnce(t *testing.T) {
	// wip a b c should mark all as wip and add all to merge
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.WriteFile("base.txt", "base")
	repo.Run("jj", "describe", "-m", "Base")
	baseID := strings.TrimSpace(repo.runSilent("jj", "log", "-r", "@", "--no-graph", "-T", "change_id.shortest()"))

	repo.Run("jjtask", "create", baseID, "Task A")
	taskA := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task A")`,
		"--no-graph", "-T", "change_id.shortest()"))

	repo.Run("jjtask", "create", baseID, "Task B")
	taskB := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task B")`,
		"--no-graph", "-T", "change_id.shortest()"))

	repo.Run("jjtask", "create", baseID, "Task C")
	taskC := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task C")`,
		"--no-graph", "-T", "change_id.shortest()"))

	// Mark all three at once
	repo.Run("jjtask", "wip", taskA, taskB, taskC)

	// All should be wip
	for _, task := range []string{taskA, taskB, taskC} {
		desc := repo.runSilent("jj", "log", "-r", task, "--no-graph", "-T", "description")
		if !strings.Contains(desc, "[task:wip]") {
			t.Errorf("task %s should be wip, got: %s", task, desc)
		}
	}

	// @ should have 3+ parents
	parentOut := repo.runSilent("jj", "log", "-r", "parents(@)", "--no-graph", "-T", `change_id.shortest() ++ "\n"`)
	lines := strings.Split(strings.TrimSpace(parentOut), "\n")
	if len(lines) < 3 {
		t.Errorf("expected 3+ parents, got %d", len(lines))
	}
}

func TestDoneMultipleRevsAtOnce(t *testing.T) {
	// done a b should mark both done
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.WriteFile("base.txt", "base")
	repo.Run("jj", "describe", "-m", "Base")

	repo.Run("jjtask", "create", "Task A")
	taskA := repo.GetTaskID("todo")
	repo.Run("jj", "edit", taskA)
	repo.WriteFile("a.txt", "a")

	repo.Run("jjtask", "create", "Task B")
	taskB := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task B")`,
		"--no-graph", "-T", "change_id.shortest()"))
	repo.Run("jj", "edit", taskB)
	repo.WriteFile("b.txt", "b")

	repo.Run("jjtask", "wip", taskA)
	repo.Run("jjtask", "wip", taskB)

	// Mark both done at once
	repo.Run("jjtask", "done", taskA, taskB)

	// Both should be done
	for _, task := range []string{taskA, taskB} {
		desc := repo.runSilent("jj", "log", "-r", task, "--no-graph", "-T", "description")
		if !strings.Contains(desc, "[task:done]") {
			t.Errorf("task %s should be done, got: %s", task, desc)
		}
	}
}

func TestDropMultipleRevsAtOnce(t *testing.T) {
	// drop a b should remove both from merge
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.WriteFile("base.txt", "base")
	repo.Run("jj", "describe", "-m", "Base")
	baseID := strings.TrimSpace(repo.runSilent("jj", "log", "-r", "@", "--no-graph", "-T", "change_id.shortest()"))

	repo.Run("jjtask", "create", baseID, "Task A")
	taskA := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task A")`,
		"--no-graph", "-T", "change_id.shortest()"))

	repo.Run("jjtask", "create", baseID, "Task B")
	taskB := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task B")`,
		"--no-graph", "-T", "change_id.shortest()"))

	repo.Run("jjtask", "wip", taskA)
	repo.Run("jjtask", "wip", taskB)

	// Drop both at once
	repo.Run("jjtask", "drop", taskA, taskB)

	// Both should be standby
	for _, task := range []string{taskA, taskB} {
		desc := repo.runSilent("jj", "log", "-r", task, "--no-graph", "-T", "description")
		if !strings.Contains(desc, "[task:standby]") {
			t.Errorf("task %s should be standby, got: %s", task, desc)
		}
	}
}

func TestDoneThreeWayMergeLinearizes(t *testing.T) {
	// With 3 WIP tasks in merge, done on first should linearize all
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.WriteFile("base.txt", "base")
	repo.Run("jj", "describe", "-m", "Base")
	baseID := strings.TrimSpace(repo.runSilent("jj", "log", "-r", "@", "--no-graph", "-T", "change_id.shortest()"))

	repo.Run("jjtask", "create", baseID, "Task A")
	taskA := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task A")`,
		"--no-graph", "-T", "change_id.shortest()"))

	repo.Run("jjtask", "create", baseID, "Task B")
	taskB := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task B")`,
		"--no-graph", "-T", "change_id.shortest()"))

	repo.Run("jjtask", "create", baseID, "Task C")
	taskC := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task C")`,
		"--no-graph", "-T", "change_id.shortest()"))

	repo.Run("jjtask", "wip", taskA)
	repo.Run("jjtask", "wip", taskB)
	repo.Run("jjtask", "wip", taskC)

	// Verify 3-way merge
	parentOut := repo.runSilent("jj", "log", "-r", "parents(@)", "--no-graph", "-T", `change_id.shortest() ++ "\n"`)
	if len(strings.Split(strings.TrimSpace(parentOut), "\n")) < 3 {
		t.Fatal("expected 3-way merge")
	}

	// Mark A as done - should linearize
	repo.Run("jjtask", "done", taskA)

	// @ should now have single parent (linear)
	parentOut = repo.runSilent("jj", "log", "-r", "parents(@)", "--no-graph", "-T", `change_id.shortest() ++ "\n"`)
	lines := strings.Split(strings.TrimSpace(parentOut), "\n")
	var nonEmpty []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			nonEmpty = append(nonEmpty, l)
		}
	}
	if len(nonEmpty) != 1 {
		t.Errorf("expected single parent after linearization, got %d: %v", len(nonEmpty), nonEmpty)
	}

	// A should be ancestor of B and C
	ancestorCheck := repo.runSilent("jj", "log", "-r", taskA+"::("+taskB+"|"+taskC+")", "--no-graph", "-T", "change_id.shortest()")
	if !strings.Contains(ancestorCheck, taskA) {
		t.Error("task A should be ancestor of B and C")
	}
}

func TestWipPreservesAtContent(t *testing.T) {
	// When @ has content and we add a wip task, content should be preserved
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.WriteFile("base.txt", "base")
	repo.Run("jj", "describe", "-m", "Base")

	// Create @ with content
	repo.Run("jj", "new", "-m", "Working commit")
	repo.WriteFile("work.txt", "my work")
	repo.Run("jj", "status") // snapshot

	// Create a task as sibling
	baseID := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `description(substring:"Base")`,
		"--no-graph", "-T", "change_id.shortest()"))
	repo.Run("jjtask", "create", baseID, "Task")
	taskID := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks()`,
		"--no-graph", "-T", "change_id.shortest()"))

	// Mark task as wip - @ should still have work.txt
	repo.Run("jjtask", "wip", taskID)

	// Check @ still has the file
	diff := repo.runSilent("jj", "diff", "-r", "@", "--stat")
	if !strings.Contains(diff, "work.txt") {
		t.Errorf("@ should still have work.txt after wip, diff: %s", diff)
	}
}

func TestDropPreservesAtContent(t *testing.T) {
	// When @ has content and we drop a task, content should be preserved
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.WriteFile("base.txt", "base")
	repo.Run("jj", "describe", "-m", "Base")
	baseID := strings.TrimSpace(repo.runSilent("jj", "log", "-r", "@", "--no-graph", "-T", "change_id.shortest()"))

	// Create two tasks
	repo.Run("jjtask", "create", baseID, "Task A")
	taskA := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task A")`,
		"--no-graph", "-T", "change_id.shortest()"))

	repo.Run("jjtask", "create", baseID, "Task B")
	taskB := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task B")`,
		"--no-graph", "-T", "change_id.shortest()"))

	repo.Run("jjtask", "wip", taskA)
	repo.Run("jjtask", "wip", taskB)

	// Add content to @ (merge)
	repo.WriteFile("merge_work.txt", "merge content")
	repo.Run("jj", "status")

	// Drop task A
	repo.Run("jjtask", "drop", taskA)

	// @ should still have merge_work.txt
	diff := repo.runSilent("jj", "diff", "-r", "@", "--stat")
	if !strings.Contains(diff, "merge_work.txt") {
		t.Errorf("@ should still have merge_work.txt after drop, diff: %s", diff)
	}
}

func TestDoneOrphanWarning(t *testing.T) {
	// When marking done on task that was never wip (not in @'s ancestry), should warn
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.WriteFile("base.txt", "base")
	repo.Run("jj", "describe", "-m", "Base")

	// Create task but never mark as wip
	repo.Run("jjtask", "create", "Orphan task")
	taskID := repo.GetTaskID("todo")

	// Work happens in @ directly, not in task
	repo.Run("jj", "new", "-m", "Working commit")
	repo.WriteFile("work.txt", "actual work")
	repo.Run("jj", "status")

	// Mark task done without ever doing wip - should warn about orphan
	output := repo.Run("jjtask", "done", taskID)

	// Should have warning about orphan
	if !strings.Contains(output, "not in @'s ancestry") && !strings.Contains(output, "orphan") {
		t.Errorf("expected orphan warning, got: %s", output)
	}

	// Task should still be marked done
	desc := repo.runSilent("jj", "log", "-r", taskID, "--no-graph", "-T", "description")
	if !strings.Contains(desc, "[task:done]") {
		t.Errorf("expected [task:done], got: %s", desc)
	}
}

func TestDoneSingleParentNoLinearization(t *testing.T) {
	// When task is only parent (not a merge), done should just mark it
	t.Parallel()
	repo := SetupTestRepo(t)

	repo.WriteFile("base.txt", "base")
	repo.Run("jj", "describe", "-m", "Base")

	repo.Run("jjtask", "create", "Task")
	taskID := repo.GetTaskID("todo")
	repo.Run("jj", "edit", taskID)
	repo.WriteFile("task.txt", "content")

	repo.Run("jjtask", "wip")

	// @ is child of task (single parent)
	repo.Run("jj", "new")

	// Mark task done
	repo.Run("jjtask", "done", taskID)

	// Task should be done
	desc := repo.runSilent("jj", "log", "-r", taskID, "--no-graph", "-T", "description")
	if !strings.Contains(desc, "[task:done]") {
		t.Errorf("expected [task:done], got: %s", desc)
	}

	// @ parent should still be the task (no orphaning)
	parentOut := repo.runSilent("jj", "log", "-r", "parents(@)", "--no-graph", "-T", "change_id.shortest()")
	if !strings.Contains(parentOut, taskID) {
		t.Errorf("@ parent should be task %s, got: %s", taskID, parentOut)
	}
}

func TestDoneMegaMergeWithWorkCommit(t *testing.T) {
	// Scenario: Mega-merge has task commits + work commits as parents.
	// When marking tasks done, work commits should remain as the base,
	// not be rebased onto task commits.
	//
	// Starting state:
	//   @    (merge working copy)
	//   ├─ taskA [task:wip]  (empty task)
	//   ├─ taskB [task:wip]  (empty task)
	//   └─ work3 (work commit with content)
	//      └─ work2
	//         └─ work1
	//            └─ base
	//
	// After `jjtask done taskA taskB`:
	//   @
	//   └─ work3 (unchanged)
	//      └─ work2
	//         └─ work1
	//            └─ taskB [task:done]
	//               └─ taskA [task:done]
	//                  └─ base
	//
	// The done tasks should linearize into the ancestry of the work branch,
	// not cause work commits to be rebased onto tasks.
	t.Parallel()
	repo := SetupTestRepo(t)

	// Create base
	repo.WriteFile("base.txt", "base")
	repo.Run("jj", "describe", "-m", "Base commit")
	baseID := strings.TrimSpace(repo.runSilent("jj", "log", "-r", "@", "--no-graph", "-T", "change_id.shortest()"))

	// Create work commits on top of base
	repo.Run("jj", "new", "-m", "Work commit 1")
	repo.WriteFile("work1.txt", "work 1")
	repo.Run("jj", "new", "-m", "Work commit 2")
	repo.WriteFile("work2.txt", "work 2")
	repo.Run("jj", "new", "-m", "Work commit 3")
	repo.WriteFile("work3.txt", "work 3")
	work3ID := strings.TrimSpace(repo.runSilent("jj", "log", "-r", "@", "--no-graph", "-T", "change_id.shortest()"))

	// Create task commits as siblings of work branch (children of base)
	repo.Run("jjtask", "create", baseID, "Task A")
	taskA := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task A")`,
		"--no-graph", "-T", "change_id.shortest()"))

	repo.Run("jjtask", "create", baseID, "Task B")
	taskB := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task B")`,
		"--no-graph", "-T", "change_id.shortest()"))

	// Go back to work3 and mark tasks as WIP to create mega-merge
	repo.Run("jj", "edit", work3ID)
	repo.Run("jj", "new") // Create new @ on top of work3
	repo.Run("jjtask", "wip", taskA)
	repo.Run("jjtask", "wip", taskB)

	// Verify we have a 3-way merge: work3, taskA, taskB
	parentOut := repo.runSilent("jj", "log", "-r", "parents(@)", "--no-graph", "-T", `change_id.shortest() ++ "\n"`)
	parentCount := len(strings.Split(strings.TrimSpace(parentOut), "\n"))
	if parentCount < 3 {
		t.Fatalf("expected 3-way merge, got %d parents", parentCount)
	}

	// Mark both tasks done
	repo.Run("jjtask", "done", taskA, taskB)

	// After done:
	// - @ should have single parent (linearized)
	// - Tasks should be ON TOP of work (tasks are newer work)
	// - Work1-2-3 chain should remain intact
	// - Result: work1 → work2 → work3 → taskA → taskB → @

	// Check @ has single parent (linearized)
	parentOut = repo.runSilent("jj", "log", "-r", "parents(@)", "--no-graph", "-T", `change_id.shortest() ++ "\n"`)
	lines := strings.Split(strings.TrimSpace(parentOut), "\n")
	var nonEmpty []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmpty = append(nonEmpty, line)
		}
	}
	if len(nonEmpty) != 1 {
		t.Errorf("expected single parent after linearization, got %d: %v", len(nonEmpty), nonEmpty)
	}

	// The parent of @ should be a done task (tasks are on top)
	atParent := strings.TrimSpace(nonEmpty[0])
	atParentDesc := repo.runSilent("jj", "log", "-r", atParent, "--no-graph", "-T", "description")
	if !strings.Contains(atParentDesc, "[task:done]") {
		t.Errorf("@ parent should be a done task, got: %s", atParentDesc)
	}

	// Work1-2-3 chain should remain intact (work2 is parent of work3)
	work3Parents := repo.runSilent("jj", "log", "-r", "parents("+work3ID+")", "--no-graph", "-T", "description")
	if !strings.Contains(work3Parents, "Work commit 2") {
		t.Errorf("work3 parent should still be work2, got: %s", work3Parents)
	}

	// Work3 should be ancestor of tasks (tasks on top of work)
	ancestorCheck := repo.runSilent("jj", "log", "-r", work3ID+"::"+taskA, "--no-graph", "-T", "change_id.shortest()")
	if !strings.Contains(ancestorCheck, work3ID) {
		t.Errorf("work3 should be ancestor of taskA")
	}

	// Check for conflicts - there should be none
	status := repo.runSilent("jj", "log", "-r", "@", "-T", `if(conflict, "CONFLICT", "ok")`)
	if strings.Contains(status, "CONFLICT") {
		t.Error("@ should not have conflicts after done")
	}
}

func TestDoneMegaMergeWithTaskContent(t *testing.T) {
	// Scenario: Mega-merge has task with actual content + work commits.
	// The task has changes that should be preserved and end up on top of work.
	//
	// Starting state:
	//   @    (merge working copy)
	//   ├─ taskA [task:wip] with task_a.txt
	//   └─ work2 (work commit)
	//      └─ work1
	//
	// After `jjtask done taskA`:
	//   @
	//   └─ taskA [task:done] with task_a.txt (content preserved)
	//      └─ work2
	//         └─ work1
	t.Parallel()
	repo := SetupTestRepo(t)

	// Create base
	repo.WriteFile("base.txt", "base")
	repo.Run("jj", "describe", "-m", "Base commit")
	baseID := strings.TrimSpace(repo.runSilent("jj", "log", "-r", "@", "--no-graph", "-T", "change_id.shortest()"))

	// Create work commits
	repo.Run("jj", "new", "-m", "Work commit 1")
	repo.WriteFile("work1.txt", "work 1")
	repo.Run("jj", "new", "-m", "Work commit 2")
	repo.WriteFile("work2.txt", "work 2")
	work2ID := strings.TrimSpace(repo.runSilent("jj", "log", "-r", "@", "--no-graph", "-T", "change_id.shortest()"))

	// Create task with content as sibling of work branch
	repo.Run("jjtask", "create", baseID, "Task A with content")
	taskA := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task A")`,
		"--no-graph", "-T", "change_id.shortest()"))

	// Add content to task
	repo.Run("jj", "edit", taskA)
	repo.WriteFile("task_a.txt", "task A content")
	repo.Run("jj", "status") // snapshot

	// Go back to work2 and create mega-merge with task
	repo.Run("jj", "edit", work2ID)
	repo.Run("jj", "new")
	repo.Run("jjtask", "wip", taskA)

	// Verify merge
	parentOut := repo.runSilent("jj", "log", "-r", "parents(@)", "--no-graph", "-T", `change_id.shortest() ++ "\n"`)
	parentCount := len(strings.Split(strings.TrimSpace(parentOut), "\n"))
	if parentCount < 2 {
		t.Fatalf("expected 2-way merge, got %d parents", parentCount)
	}

	// Mark task done
	repo.Run("jjtask", "done", taskA)

	// After done:
	// - @ should have single parent (the done task)
	// - Task should be on top of work2
	// - Task content (task_a.txt) should be preserved

	// Check @ parent is the done task
	atParent := strings.TrimSpace(repo.runSilent("jj", "log", "-r", "parents(@)", "--no-graph", "-T", "change_id.shortest()"))
	atParentDesc := repo.runSilent("jj", "log", "-r", atParent, "--no-graph", "-T", "description")
	if !strings.Contains(atParentDesc, "[task:done]") {
		t.Errorf("@ parent should be done task, got: %s", atParentDesc)
	}

	// Task should have work2 as parent (task on top of work)
	taskParent := repo.runSilent("jj", "log", "-r", "parents("+taskA+")", "--no-graph", "-T", "description")
	if !strings.Contains(taskParent, "Work commit 2") {
		t.Errorf("task parent should be work2, got: %s", taskParent)
	}

	// Task content should be preserved
	taskDiff := repo.runSilent("jj", "diff", "-r", taskA, "--stat")
	if !strings.Contains(taskDiff, "task_a.txt") {
		t.Errorf("task should still have task_a.txt, got: %s", taskDiff)
	}

	// No conflicts
	status := repo.runSilent("jj", "log", "-r", "@", "-T", `if(conflict, "CONFLICT", "ok")`)
	if strings.Contains(status, "CONFLICT") {
		t.Error("@ should not have conflicts after done")
	}
}

func TestDoneMegaMergeWithTaskInWorkChain(t *testing.T) {
	// Scenario: Task A is already in the work chain (not a direct parent of @).
	// Task B is a direct parent of @.
	// When marking both done, Task A should stay in place.
	//
	// Starting state:
	//   @    (merge)
	//   ├─ taskB [task:wip] with content  (direct parent)
	//   └─ work2
	//      └─ taskA [task:wip] with content  (in work chain, not direct parent)
	//         └─ work1
	//
	// After `jjtask done taskA taskB`:
	//   @
	//   └─ taskB [task:done]
	//      └─ work2
	//         └─ taskA [task:done]  (stays in place!)
	//            └─ work1
	t.Parallel()
	repo := SetupTestRepo(t)

	// Create base and work1
	repo.WriteFile("base.txt", "base")
	repo.Run("jj", "describe", "-m", "Base commit")
	repo.Run("jj", "new", "-m", "Work commit 1")
	repo.WriteFile("work1.txt", "work 1")
	work1ID := strings.TrimSpace(repo.runSilent("jj", "log", "-r", "@", "--no-graph", "-T", "change_id.shortest()"))

	// Create Task A with content as child of work1
	repo.Run("jjtask", "create", work1ID, "Task A in chain")
	taskA := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task A")`,
		"--no-graph", "-T", "change_id.shortest()"))
	repo.Run("jj", "edit", taskA)
	repo.WriteFile("task_a.txt", "task A content")
	repo.Run("jjtask", "wip") // Mark as wip

	// Create work2 on top of taskA
	repo.Run("jj", "new", "-m", "Work commit 2")
	repo.WriteFile("work2.txt", "work 2")
	work2ID := strings.TrimSpace(repo.runSilent("jj", "log", "-r", "@", "--no-graph", "-T", "change_id.short()"))

	// Create Task B as sibling of work2 (child of base)
	baseID := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `description(substring:"Base commit")`,
		"--no-graph", "-T", "change_id.shortest()"))
	repo.Run("jjtask", "create", baseID, "Task B sibling")
	taskB := strings.TrimSpace(repo.runSilent("jj", "log", "-r", `tasks() & description(substring:"Task B")`,
		"--no-graph", "-T", "change_id.shortest()"))
	repo.Run("jj", "edit", taskB)
	repo.WriteFile("task_b.txt", "task B content")

	// Create mega-merge: @ has work2 and taskB as parents
	repo.Run("jj", "edit", work2ID)
	repo.Run("jj", "new")
	repo.Run("jjtask", "wip", taskB)

	// Verify structure before done
	// taskA should be ancestor of work2, not direct parent of @
	ancestorCheck := repo.runSilent("jj", "log", "-r", `description(substring:"Task A")::`+work2ID, "--no-graph", "-T", "change_id.shortest()")
	if ancestorCheck == "" {
		t.Fatalf("taskA should be ancestor of work2")
	}

	// Mark both tasks done
	repo.Run("jjtask", "done", taskA, taskB)

	// After done:
	// - taskA should still be between work1 and work2 (stays in place)
	// - taskB should be on top of work2
	// - @ should be on top of taskB

	// Check taskA is still parent of work2
	work2Parents := repo.runSilent("jj", "log", "-r", "parents("+work2ID+")", "--no-graph", "-T", "description")
	if !strings.Contains(work2Parents, "Task A") {
		t.Errorf("work2 parent should still be taskA, got: %s", work2Parents)
	}

	// Check taskA has work1 as parent
	taskAParents := repo.runSilent("jj", "log", "-r", "parents("+taskA+")", "--no-graph", "-T", "description")
	if !strings.Contains(taskAParents, "Work commit 1") {
		t.Errorf("taskA parent should be work1, got: %s", taskAParents)
	}

	// Check @ parent is taskB (done, on top of work chain)
	atParent := strings.TrimSpace(repo.runSilent("jj", "log", "-r", "parents(@)", "--no-graph", "-T", "change_id.shortest()"))
	atParentDesc := repo.runSilent("jj", "log", "-r", atParent, "--no-graph", "-T", "description")
	if !strings.Contains(atParentDesc, "Task B") {
		t.Errorf("@ parent should be taskB, got: %s", atParentDesc)
	}

	// Both tasks should be marked done
	taskADesc := repo.runSilent("jj", "log", "-r", taskA, "--no-graph", "-T", "description")
	if !strings.Contains(taskADesc, "[task:done]") {
		t.Errorf("taskA should be done, got: %s", taskADesc)
	}
	taskBDesc := repo.runSilent("jj", "log", "-r", taskB, "--no-graph", "-T", "description")
	if !strings.Contains(taskBDesc, "[task:done]") {
		t.Errorf("taskB should be done, got: %s", taskBDesc)
	}

	// No conflicts
	status := repo.runSilent("jj", "log", "-r", "@", "-T", `if(conflict, "CONFLICT", "ok")`)
	if strings.Contains(status, "CONFLICT") {
		t.Error("@ should not have conflicts")
	}
}
