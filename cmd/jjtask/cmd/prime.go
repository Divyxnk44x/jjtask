package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"jjtask/internal/config"
	"jjtask/internal/workspace"
)

// hookEvent represents the hook event name from Claude Code
type hookEvent string

const (
	hookEventSessionStart hookEvent = "SessionStart"
	hookEventPreCompact   hookEvent = "PreCompact"
)

// hookPayload represents the JSON payload from Claude Code hooks
type hookPayload struct {
	HookEventName string `json:"hook_event_name"`
	Trigger       string `json:"trigger"` // "manual" or "auto" for PreCompact
	Source        string `json:"source"`  // "startup", "resume", "clear", "compact" for SessionStart
}

// detectHookEvent detects which Claude Code hook triggered this invocation
func detectHookEvent() (event hookEvent, trigger string) {
	// Claude Code passes payload as JSON to stdin
	// Check if stdin is a pipe (not TTY)
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		// TTY - no hook data
		return hookEventSessionStart, ""
	}

	// Read stdin with timeout to avoid blocking on empty pipe
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	dataCh := make(chan []byte, 1)
	go func() {
		data, _ := io.ReadAll(os.Stdin)
		dataCh <- data
	}()

	select {
	case data := <-dataCh:
		if len(data) == 0 {
			return hookEventSessionStart, ""
		}
		var p hookPayload
		if err := json.Unmarshal(data, &p); err == nil {
			switch p.HookEventName {
			case "PreCompact":
				return hookEventPreCompact, p.Trigger
			case "SessionStart":
				return hookEventSessionStart, p.Source
			}
		}
	case <-ctx.Done():
		// Timeout - no data available
	}

	return hookEventSessionStart, ""
}

var primeCompact bool

var primeCmd = &cobra.Command{
	Use:   "prime",
	Short: "Output session context for hooks",
	Long: `Output current task context for use in hooks or prompts.

This is typically used by SessionStart and PreCompact hooks to provide
context about pending tasks to AI assistants.

For PreCompact (auto), outputs task state verification instructions.
For SessionStart, outputs full quick reference.

Use --compact for minimal output (task counts only).`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		event, trigger := detectHookEvent()

		// PreCompact auto = context nearly full, output verification prompt
		if event == hookEventPreCompact && trigger == "auto" {
			return printPreCompactContext()
		}

		if primeCompact {
			return printCompactPrime()
		}

		// Check for custom prime content
		customContent, hasCustom, err := config.GetPrimeContent()
		if err != nil {
			return fmt.Errorf("reading prime config: %w", err)
		}
		if hasCustom {
			fmt.Println()
			fmt.Print(customContent)
			if !strings.HasSuffix(customContent, "\n") {
				fmt.Println()
			}
			printCurrentTasks()
			return nil
		}

		fmt.Println()
		fmt.Println("## JJ TASK Quick Reference")
		fmt.Println()
		fmt.Println("Task flags: draft → todo → wip → done (also: blocked, standby, untested, review)")
		fmt.Println()

		fmt.Println("### Revsets")
		fmt.Println("tasks(), tasks_pending(), tasks_todo(), tasks_wip(), tasks_done(), tasks_blocked()")
		fmt.Println()

		fmt.Println("### Commands (all support -R, --quiet, etc.)")
		fmt.Println("jjtask create [PARENT] TITLE [DESC]  Create task (parent defaults to @)")
		fmt.Println("jjtask wip [TASKS...]                Mark WIP, add as parents of @")
		fmt.Println("jjtask done [TASKS...]               Mark done, rebase on top of work")
		fmt.Println("jjtask drop TASKS... [--abandon]     Remove from @ (standby or abandon)")
		fmt.Println("jjtask squash                        Flatten @ merge for push")
		fmt.Println("jjtask find [-s STATUS] [-r REVSET]  List tasks (status: todo/wip/done/all)")
		fmt.Println("jjtask flag STATUS [-r REV]          Change task flag (defaults to @)")
		fmt.Println("jjtask parallel T1 T2... [-p REV]    Create sibling tasks (defaults to @)")
		fmt.Println("jjtask show-desc [-r REV]            Print revision description")
		fmt.Println("jjtask desc-transform CMD [-r REV]   Transform description with command")
		fmt.Println("jjtask checkpoint [-m MSG]           Create checkpoint commit")
		fmt.Println("jjtask stale                         Find done tasks not in @'s ancestry")
		fmt.Println("jjtask all CMD [ARGS]                Run jj CMD across workspaces")
		fmt.Println()

		fmt.Println("### Workflow")
		fmt.Println("1. `jjtask create 'task'`        # Plan tasks")
		fmt.Println("2. `jjtask wip TASK`             # Start (single=edit, multi=merge)")
		fmt.Println("3. `jj edit TASK` to work        # Work directly in task branch")
		fmt.Println("4. `jjtask done`                 # Complete, rebases on top of work")
		fmt.Println("5. `jjtask squash`               # Flatten for push")
		fmt.Println()
		fmt.Println("Key: @ is merge of WIP tasks. Work in task branches directly.")
		fmt.Println("For merge: `jj edit TASK`, not bare `jj absorb`.")
		fmt.Println()

		fmt.Println("### Rules")
		fmt.Println("- DAG = priority: parent tasks complete before children")
		fmt.Println("- Chain related tasks: `jjtask create --chain 'Next step'`")
		fmt.Println("- Read full spec before editing - descriptions are specifications")
		fmt.Println("- Never mark done unless ALL acceptance criteria pass")
		fmt.Println("- Use `jjtask flag review/blocked/untested` if incomplete")
		fmt.Println("- Stop and report if unsure - don't attempt JJ recovery ops")
		fmt.Println()

		fmt.Println("### Before Saying Done")
		fmt.Println("[ ] All acceptance criteria in task spec pass")
		fmt.Println("[ ] `jjtask done TASK` - mark complete")
		fmt.Println("[ ] `jjtask squash` - flatten for push when ready")
		fmt.Println()

		fmt.Println("### Native Task Tools")
		fmt.Println("TaskCreate, TaskUpdate, TaskList, TaskGet - for session workflow tracking")
		fmt.Println("Use for: multi-step work within a session, dependency ordering, progress display")
		fmt.Println("jjtask = persistent tasks in repo history; Task* = ephemeral session tracking")
		fmt.Println()

		fmt.Println("### Current Tasks")
		printTaskDAG()

		return nil
	},
}

// printCurrentTasks outputs the current tasks section
func printCurrentTasks() {
	fmt.Println()
	fmt.Println("### Current Tasks")
	printTaskDAG()
}

// printTaskDAG outputs pending tasks with @ using graph view
func printTaskDAG() {
	repos, workspaceRoot, _ := workspace.GetRepos()
	PrintTasksWithRevset(repos, workspaceRoot, "tasks_pending() | @")
}

// printPreCompactContext outputs task verification when context is nearly full
func printPreCompactContext() error {
	fmt.Println()
	fmt.Println("## 🚨 Context Compacting - Verify Task State")
	fmt.Println()
	fmt.Println("Context window nearly full. Before compaction, verify:")
	fmt.Println()
	fmt.Println("### Current WIP Tasks")

	repos, workspaceRoot, _ := workspace.GetRepos()
	hasWIP := false

	for _, repo := range repos {
		repoPath := workspace.ResolveRepoPath(repo, workspaceRoot)
		out, err := client.Query("-R", repoPath, "log", "--no-graph", "-r", "tasks_wip()", "-T", "task_log_flat")
		if err == nil {
			outStr := strings.TrimRight(out, "\n")
			if outStr != "" {
				hasWIP = true
				fmt.Println(outStr)
			}
		}
	}

	if !hasWIP {
		fmt.Println("(no WIP tasks)")
	}

	fmt.Println()
	fmt.Println("### Verification Checklist")
	fmt.Println("[ ] WIP tasks still accurate? Update status if needed")
	fmt.Println("[ ] Any completed work not marked done?")
	fmt.Println("[ ] Need to create follow-up tasks before context lost?")
	fmt.Println()
	fmt.Println("### Actions")
	fmt.Println("- `jjtask find wip` - review all WIP tasks")
	fmt.Println("- `jjtask done TASK` - mark completed work")
	fmt.Println("- `jjtask create 'Follow-up'` - capture discovered work")
	fmt.Println()
	fmt.Println("Confirm with user if task state needs updates before proceeding.")
	fmt.Println()

	return nil
}

// printCompactPrime outputs minimal task summary
func printCompactPrime() error {
	repos, workspaceRoot, _ := workspace.GetRepos()

	var totalWIP, totalTodo, totalDraft int
	for _, repo := range repos {
		repoPath := workspace.ResolveRepoPath(repo, workspaceRoot)

		count := func(revset string) int {
			out, _ := client.Query("-R", repoPath, "log", "--no-graph", "-r", revset, "-T", "change_id.shortest() ++ \"\\n\"")
			if out == "" {
				return 0
			}
			return len(strings.Split(strings.TrimSpace(out), "\n"))
		}

		totalWIP += count("tasks_wip()")
		totalTodo += count("tasks_todo()")
		totalDraft += count("tasks_draft()")
	}

	fmt.Println()
	fmt.Println("## JJ TASK Quick Reference")
	fmt.Println()
	fmt.Println("Task flags: draft → todo → wip → done (also: blocked, standby, untested, review)")
	fmt.Printf("Current: %d wip, %d todo, %d draft\n", totalWIP, totalTodo, totalDraft)
	fmt.Println()
	fmt.Println("```")
	fmt.Println("jjtask find              # show task DAG")
	fmt.Println("jjtask show-desc -r ID   # read spec before starting")
	fmt.Println("jjtask wip ID            # start task")
	fmt.Println("jjtask done              # complete (all criteria met)")
	fmt.Println("jjtask create TITLE      # new task")
	fmt.Println("jjtask -h                # all commands")
	fmt.Println("```")
	fmt.Println()
	fmt.Println("Load `/jjtask` for full workflow.")

	return nil
}

func init() {
	rootCmd.AddCommand(primeCmd)
	primeCmd.Flags().BoolVar(&primeCompact, "compact", false, "minimal output (task counts only)")
}
