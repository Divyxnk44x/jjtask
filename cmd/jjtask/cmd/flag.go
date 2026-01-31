package cmd

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

var validFlags = []string{"draft", "todo", "wip", "untested", "standby", "review", "blocked", "done"}

var (
	flagRev string
)

var flagCmd = &cobra.Command{
	Use:       "flag [REV] <status>",
	Short:     "Update task status flag",
	ValidArgs: validFlags,
	Long: `Update the [task:*] flag in a revision description.

Valid flags: draft, todo, wip, untested, standby, review, blocked, done

Examples:
  jjtask flag wip
  jjtask flag xqq done
  jjtask flag done --rev mxyz`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		var toFlag string
		rev := flagRev

		// Support both "flag STATUS" and "flag REV STATUS"
		if len(args) == 2 {
			rev = args[0]
			toFlag = args[1]
		} else {
			toFlag = args[0]
		}

		if !slices.Contains(validFlags, toFlag) {
			return fmt.Errorf("invalid flag %q, must be one of: %s", toFlag, strings.Join(validFlags, ", "))
		}

		// Check for pending children and empty task when marking done
		if toFlag == "done" {
			checkPendingChildren(cmd, rev)
			checkEmptyTask(cmd, rev)
		}

		// Check for blocked ancestors, done ancestors, and existing WIP when marking wip
		if toFlag == "wip" {
			checkBlockedAncestors(cmd, rev)
			checkDoneAncestors(cmd, rev)
			checkExistingWip(cmd, rev)
		}

		desc, err := client.GetDescription(rev)
		if err != nil {
			return fmt.Errorf("failed to get description: %w", err)
		}

		// Detect current flag
		taskPattern := regexp.MustCompile(`^\[task:(\w+)\]`)
		match := taskPattern.FindStringSubmatch(desc)

		var newDesc string
		if match == nil {
			// No current flag - prepend the new one
			newDesc = fmt.Sprintf("[task:%s] %s", toFlag, desc)
		} else {
			// Replace old flag with new
			newDesc = taskPattern.ReplaceAllString(desc, fmt.Sprintf("[task:%s]", toFlag))
		}

		if err := client.SetDescription(rev, newDesc); err != nil {
			return fmt.Errorf("failed to set description: %w", err)
		}

		// Check if @ has uncommitted work and is not the task being marked
		checkWorkingCopyDiff(cmd, rev, toFlag)

		fmt.Fprintln(os.Stderr, "Tip: Consider using 'jjtask wip' or 'jjtask done' for the mega-merge workflow")

		return nil
	},
}

// checkExistingWip warns when marking a new task as WIP while another WIP exists
// Returns error if the new WIP task is not in the same chain as existing WIP
func checkExistingWip(cmd *cobra.Command, newWipRev string) {
	// Find existing WIP tasks (excluding the one we're about to mark)
	wipOutput, err := client.Query("log", "-r", fmt.Sprintf("tasks_wip() ~ %s", newWipRev), "--no-graph", "-T", `change_id.shortest() ++ " " ++ description.first_line() ++ "\n"`)
	if err != nil || strings.TrimSpace(wipOutput) == "" {
		return // No other WIP task
	}

	wipLine := strings.TrimSpace(strings.Split(wipOutput, "\n")[0])
	wipParts := strings.SplitN(wipLine, " ", 2)
	if len(wipParts) == 0 {
		return
	}
	wipID := wipParts[0]
	wipTitle := ""
	if len(wipParts) > 1 {
		wipTitle = wipParts[1]
	}

	// Check if new WIP is ancestor or descendant of existing WIP (same chain)
	checkRevset := fmt.Sprintf("(%s & (ancestors(%s) | descendants(%s)))", newWipRev, wipID, wipID)
	result, err := client.Query("log", "-r", checkRevset, "--no-graph", "-T", "change_id.shortest()", "--limit", "1")
	if err == nil && strings.TrimSpace(result) != "" {
		return // In same chain, OK
	}

	// Not in same chain - warn
	stderr := cmd.ErrOrStderr()
	_, _ = fmt.Fprintln(stderr)
	_, _ = fmt.Fprintf(stderr, "⚠️  Another WIP task exists: %s %s\n", wipID, wipTitle)
	_, _ = fmt.Fprintln(stderr, "Multiple WIP tasks in different branches can be confusing.")
	_, _ = fmt.Fprintln(stderr, "Options:")
	_, _ = fmt.Fprintf(stderr, "  • Pause existing: jjtask flag blocked -r %s\n", wipID)
	_, _ = fmt.Fprintf(stderr, "  • Switch to existing: jj edit %s\n", wipID)
	_, _ = fmt.Fprintf(stderr, "  • Rebase to chain: jj rebase -s %s -d %s\n", newWipRev, wipID)
	_, _ = fmt.Fprintln(stderr)
}

// checkDoneAncestors warns if any ancestor task is done
func checkDoneAncestors(cmd *cobra.Command, taskRev string) {
	// Get done ancestors (tasks with [task:done] prefix)
	doneRevset := fmt.Sprintf("ancestors(%s) & tasks_done()", taskRev)
	output, err := client.Query("log", "-r", doneRevset, "--no-graph", "-T", `change_id.shortest() ++ " " ++ description.first_line() ++ "\n"`, "--limit", "3")
	if err != nil {
		return
	}
	output = strings.TrimSpace(output)
	if output == "" {
		return
	}

	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return
	}

	stderr := cmd.ErrOrStderr()
	_, _ = fmt.Fprintln(stderr)
	_, _ = fmt.Fprintln(stderr, "⚠️  Ancestor task is already done:")
	for _, line := range lines {
		_, _ = fmt.Fprintf(stderr, "  • %s\n", line)
	}
	_, _ = fmt.Fprintln(stderr, "Starting work below done tasks is unusual. Consider:")
	_, _ = fmt.Fprintln(stderr, "  • Rebase as sibling: jj rebase -s", taskRev, "-d <done-task>~")
	_, _ = fmt.Fprintln(stderr, "  • Or squash done tasks: jjtask squash -r <done-task>")
	_, _ = fmt.Fprintln(stderr)
}

// checkBlockedAncestors warns if any ancestor task is blocked
func checkBlockedAncestors(cmd *cobra.Command, taskRev string) {
	// Get blocked ancestors
	blockedRevset := fmt.Sprintf("ancestors(%s) & tasks_blocked()", taskRev)
	output, err := client.Query("log", "-r", blockedRevset, "--no-graph", "-T", `change_id.shortest() ++ " " ++ description.first_line() ++ "\n"`)
	if err != nil {
		return
	}
	output = strings.TrimSpace(output)
	if output == "" {
		return
	}

	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return
	}

	stderr := cmd.ErrOrStderr()
	_, _ = fmt.Fprintln(stderr)
	_, _ = fmt.Fprintln(stderr, "⚠️  Ancestor task is blocked:")
	for _, line := range lines {
		_, _ = fmt.Fprintf(stderr, "  • %s\n", line)
	}
	_, _ = fmt.Fprintln(stderr, "Consider unblocking the ancestor first.")
	_, _ = fmt.Fprintln(stderr)
}

// checkPendingChildren warns if task has pending child tasks
func checkPendingChildren(cmd *cobra.Command, taskRev string) {
	// Get pending children (tasks that are not done)
	pendingRevset := fmt.Sprintf("children(%s) & tasks_pending()", taskRev)
	output, err := client.Query("log", "-r", pendingRevset, "--no-graph", "-T", `change_id.shortest() ++ " " ++ description.first_line() ++ "\n"`)
	if err != nil {
		return
	}
	output = strings.TrimSpace(output)
	if output == "" {
		return
	}

	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return
	}

	stderr := cmd.ErrOrStderr()
	_, _ = fmt.Fprintln(stderr)
	_, _ = fmt.Fprintf(stderr, "⚠️  Task has %d pending children:\n", len(lines))
	for _, line := range lines {
		_, _ = fmt.Fprintf(stderr, "  • %s\n", line)
	}
	_, _ = fmt.Fprintln(stderr, "Consider marking children done first, or they may be orphaned.")
	_, _ = fmt.Fprintln(stderr)
}

// checkEmptyTask warns if marking an empty revision as done
func checkEmptyTask(cmd *cobra.Command, taskRev string) {
	diff, err := client.Query("diff", "-r", taskRev, "--stat")
	if err != nil {
		return
	}
	diff = strings.TrimSpace(diff)
	if diff != "" && !strings.HasPrefix(diff, "0 files changed") {
		return // has content
	}

	stderr := cmd.ErrOrStderr()
	_, _ = fmt.Fprintln(stderr)
	_, _ = fmt.Fprintln(stderr, "⚠️  Task is empty - no changes to mark done")
	_, _ = fmt.Fprintln(stderr, "If this is a planning-only task, this warning can be ignored.")
	_, _ = fmt.Fprintln(stderr)
}

// checkWorkingCopyDiff warns if @ has changes that might belong to the task
func checkWorkingCopyDiff(cmd *cobra.Command, taskRev, _flag string) {
	// Get @ change id
	atID, err := client.Query("log", "-r", "@", "--no-graph", "-T", "change_id.shortest()")
	if err != nil {
		return
	}
	atID = strings.TrimSpace(atID)

	// Get task change id
	taskID, err := client.Query("log", "-r", taskRev, "--no-graph", "-T", "change_id.shortest()")
	if err != nil {
		return
	}
	taskID = strings.TrimSpace(taskID)

	// If @ is the task, no warning needed
	if atID == taskID {
		return
	}

	// Check if @ has a diff (actual file changes, not just the summary line)
	diff, err := client.Query("diff", "-r", "@", "--stat")
	if err != nil {
		return
	}
	diff = strings.TrimSpace(diff)
	// Empty diff or only summary line with "0 files changed" means no real changes
	if diff == "" || strings.HasPrefix(diff, "0 files changed") {
		return
	}

	// @ has changes and is not the task - warn
	stderr := cmd.ErrOrStderr()
	_, _ = fmt.Fprintln(stderr)
	_, _ = fmt.Fprintln(stderr, "⚠️  Working copy (@) has uncommitted changes:")
	_, _ = fmt.Fprintln(stderr, diff)
	_, _ = fmt.Fprintln(stderr, "Were any of these changes part of this task?")
	_, _ = fmt.Fprintln(stderr)
}

// setTaskFlag is a helper to update a task's flag in its description
func setTaskFlag(rev, flag string) error {
	desc, err := client.GetDescription(rev)
	if err != nil {
		return err
	}

	taskPattern := regexp.MustCompile(`^\[task:(\w+)\]`)
	var newDesc string
	if taskPattern.MatchString(desc) {
		newDesc = taskPattern.ReplaceAllString(desc, fmt.Sprintf("[task:%s]", flag))
	} else {
		newDesc = fmt.Sprintf("[task:%s] %s", flag, desc)
	}
	return client.SetDescription(rev, newDesc)
}

func init() {
	rootCmd.AddCommand(flagCmd)

	flagCmd.Flags().StringVarP(&flagRev, "rev", "r", "@", "revision to update")

	flagCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeTaskFlag(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Complete --rev flag with task revisions
	_ = flagCmd.RegisterFlagCompletionFunc("rev", completeTaskRevision)
}
