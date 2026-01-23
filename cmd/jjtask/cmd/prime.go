package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"jjtask/internal/parallel"
	"jjtask/internal/workspace"
)

var primeCmd = &cobra.Command{
	Use:   "prime",
	Short: "Output session context for hooks",
	Long: `Output current task context for use in hooks or prompts.

This is typically used by SessionStart hooks to provide context
about pending tasks to AI assistants.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println()
		fmt.Println("## JJ TASK Quick Reference")
		fmt.Println()
		fmt.Println("Task flags: draft → todo → wip → done (also: blocked, standby, untested, review)")
		fmt.Println()

		fmt.Println("### Revsets")
		fmt.Println("tasks(), tasks_pending(), tasks_todo(), tasks_wip(), tasks_done(), tasks_blocked()")
		fmt.Println()

		fmt.Println("### Commands (all support -R, --quiet, etc.)")
		fmt.Println("jjtask find [FLAG] [-r REVSET]       List tasks (flags: todo/wip/done/all, -r for revset)")
		fmt.Println("jjtask create PARENT TITLE [DESC]    Create task child of PARENT (required)")
		fmt.Println("jjtask flag REV FLAG                 Change task flag")
		fmt.Println("jjtask next [--mark-as FLAG] [REV]   Review current specs, optionally transition")
		fmt.Println("jjtask hoist                         Rebase pending tasks to children of @")
		fmt.Println("jjtask finalize [REV]                Strip [task:*] prefix for final commit")
		fmt.Println("jjtask parallel PARENT T1 T2...      Create sibling tasks under PARENT")
		fmt.Println("jjtask show-desc [REV]               Print revision description")
		fmt.Println("jjtask desc-transform REV SED_EXPR   Transform description with sed")
		fmt.Println("jjtask batch-desc SED_EXPR REVSET    Transform multiple descriptions")
		fmt.Println("jjtask checkpoint [MSG]              Create checkpoint commit")
		fmt.Println("jjtask all CMD [ARGS]                Run jj CMD across workspaces")
		fmt.Println()

		fmt.Println("### Workflow")
		fmt.Println("1. `/jjtask` - load skill for full workflow docs")
		fmt.Println("2. `jjtask find` - see task DAG (DAG order = priority)")
		fmt.Println("3. `jjtask show-desc REV` - read FULL spec before starting")
		fmt.Println("4. `jj edit REV && jjtask flag @ wip` - start work")
		fmt.Println("5. `jjtask hoist` - after commits, rebase tasks to stay children of @")
		fmt.Println("6. `jjtask next --mark-as done NEXT` - only when ALL criteria met")
		fmt.Println()

		fmt.Println("### Rules")
		fmt.Println("- DAG = priority: parent tasks complete before children")
		fmt.Println("- Chain related tasks: `jjtask create PREV_TASK 'Next step'`")
		fmt.Println("- Read full spec before editing - descriptions are specifications")
		fmt.Println("- Never mark done unless ALL acceptance criteria pass")
		fmt.Println("- Use --mark-as review/blocked/untested if incomplete")
		fmt.Println("- `jjtask hoist` keeps task DAG connected to current work")
		fmt.Println("- Stop and report if unsure - don't attempt JJ recovery ops")
		fmt.Println()

		fmt.Println("### Native Task Tools")
		fmt.Println("TaskCreate, TaskUpdate, TaskList, TaskGet - for session workflow tracking")
		fmt.Println("Use for: multi-step work within a session, dependency ordering, progress display")
		fmt.Println("jjtask = persistent tasks in repo history; Task* = ephemeral session tracking")
		fmt.Println()

		// Check for parallel session context
		printParallelContext()

		fmt.Println("### Current Tasks")

		repos, workspaceRoot, _ := workspace.GetRepos()
		isMulti := len(repos) > 1

		for _, repo := range repos {
			repoPath := workspace.ResolveRepoPath(repo, workspaceRoot)

			if isMulti {
				fmt.Printf("--- %s ---\n", workspace.DisplayName(repo))
			}

			// Show pending tasks (include @ only if it's a task)
			out, err := client.Query("-R", repoPath, "log", "--no-graph", "-r", "tasks_pending() | (@ & tasks())", "-T", "task_log_flat")
			if err == nil {
				outStr := strings.TrimRight(out, "\n")
				if outStr != "" {
					fmt.Println(outStr)
				}
			}

			if isMulti {
				fmt.Println()
			}
		}

		return nil
	},
}

func printParallelContext() {
	// Detection strategies:
	// 1. JJTASK_AGENT_ID env var (explicit)
	// 2. In .jjtask-workspaces/<agent-id>/ directory
	// 3. Current task has parallel session markers

	agentID := os.Getenv("JJTASK_AGENT_ID")

	// Check if in workspace directory
	if agentID == "" {
		cwd, _ := os.Getwd()
		if strings.Contains(cwd, parallel.WorkspacesDir) {
			// Extract agent ID from path
			parts := strings.Split(cwd, parallel.WorkspacesDir+string(filepath.Separator))
			if len(parts) > 1 {
				// Get first path component after .jjtask-workspaces/
				agentPath := strings.Split(parts[1], string(filepath.Separator))
				if len(agentPath) > 0 && agentPath[0] != "" {
					agentID = agentPath[0]
				}
			}
		}
	}

	// Find parallel session
	session, parentRev, parentDesc, err := findParallelSession()
	if err != nil || session == nil {
		return
	}

	// Get parent title
	parentTitle := ""
	lines := strings.Split(parentDesc, "\n")
	if len(lines) > 0 {
		parentTitle = strings.TrimSpace(lines[0])
	}

	fmt.Println("### Parallel Session Active")
	fmt.Println()
	fmt.Printf("Mode: %s\n", session.Mode)
	fmt.Printf("Parent: %s %s\n", parentRev, parentTitle)

	if agentID != "" {
		fmt.Printf("Agent: %s\n", agentID)

		agent := session.GetAgentByID(agentID)
		if agent != nil && agent.FilePattern != "" {
			fmt.Printf("Your assignment: %s\n", agent.FilePattern)
		}

		// Show files to avoid
		others := session.OtherAgents(agentID)
		if len(others) > 0 {
			var avoidPatterns []string
			for _, o := range others {
				if o.FilePattern != "" {
					avoidPatterns = append(avoidPatterns, o.FilePattern)
				}
			}
			if len(avoidPatterns) > 0 {
				fmt.Printf("Avoid: %s\n", strings.Join(avoidPatterns, ", "))
			}
		}
	}

	// Show other agents status
	fmt.Println()
	fmt.Println("Agents:")
	for _, a := range session.Agents {
		flag := "?"
		if session.Mode == "workspace" && a.TaskID != "" {
			flag = getTaskFlag(a.TaskID)
		}
		marker := ""
		if a.ID == agentID {
			marker = " ← you"
		}
		pattern := ""
		if a.FilePattern != "" {
			pattern = fmt.Sprintf(" (%s)", a.FilePattern)
		}
		fmt.Printf("- %s: [%s]%s%s\n", a.ID, flag, pattern, marker)
	}
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(primeCmd)
}
