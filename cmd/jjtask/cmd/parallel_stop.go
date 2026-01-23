package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"jjtask/internal/parallel"
)

var (
	parallelStopMerge          bool
	parallelStopForce          bool
	parallelStopKeepWorkspaces bool
)

var parallelStopCmd = &cobra.Command{
	Use:   "parallel-stop [parent-task]",
	Short: "Stop a parallel agent session",
	Long: `Clean up a parallel session - optionally merge work and remove workspaces.

Examples:
  jjtask parallel-stop              # cleanup only
  jjtask parallel-stop --merge      # merge all done agents into parent
  jjtask parallel-stop --force      # cleanup even if agents not done`,
	Args: cobra.MaximumNArgs(1),
	RunE: runParallelStop,
}

func init() {
	parallelStopCmd.Flags().BoolVar(&parallelStopMerge, "merge", false, "Merge completed agent work into parent")
	parallelStopCmd.Flags().BoolVar(&parallelStopForce, "force", false, "Stop even if agents are not done")
	parallelStopCmd.Flags().BoolVar(&parallelStopKeepWorkspaces, "keep-workspaces", false, "Don't remove workspace directories")
	rootCmd.AddCommand(parallelStopCmd)
	parallelStopCmd.ValidArgsFunction = completeRevision
}

func runParallelStop(cmd *cobra.Command, args []string) error {
	var session *parallel.Session
	var parentRev, parentDesc string
	var err error

	if len(args) > 0 {
		parentRev = args[0]
		parentDesc, err = client.GetDescription(parentRev)
		if err != nil {
			return fmt.Errorf("get description: %w", err)
		}
		session, err = parallel.ParseSession(parentDesc)
		if err != nil {
			return fmt.Errorf("parse session from %s: %w", parentRev, err)
		}
	} else {
		session, parentRev, parentDesc, err = findParallelSession()
		if err != nil {
			return err
		}
	}

	if session == nil {
		return fmt.Errorf("no parallel session found")
	}

	repoRoot, err := client.Root()
	if err != nil {
		return fmt.Errorf("get repo root: %w", err)
	}

	// Check agent completion status
	var incomplete []string
	var doneAgents []parallel.Agent
	for _, agent := range session.Agents {
		var flag string
		if session.Mode == "workspace" && agent.TaskID != "" {
			flag = getTaskFlag(agent.TaskID)
		} else {
			flag = getTaskFlag(parentRev)
		}

		if flag == "done" {
			doneAgents = append(doneAgents, agent)
		} else {
			incomplete = append(incomplete, fmt.Sprintf("%s (%s)", agent.ID, flag))
		}
	}

	if len(incomplete) > 0 && !parallelStopForce {
		return fmt.Errorf("agents not done: %s\nUse --force to stop anyway", strings.Join(incomplete, ", "))
	}

	// Merge if requested
	if parallelStopMerge && session.Mode == "workspace" {
		fmt.Println("Merging completed work...")
		for _, agent := range doneAgents {
			if agent.TaskID == "" {
				continue
			}
			fmt.Printf("  Squashing %s (%s) into %s\n", agent.ID, agent.TaskID, parentRev)
			if err := client.Run("squash", "--from", agent.TaskID, "--into", parentRev); err != nil {
				fmt.Printf("  Warning: failed to squash %s: %v\n", agent.ID, err)
			}
		}
	}

	// Cleanup workspaces
	if session.Mode == "workspace" && !parallelStopKeepWorkspaces {
		fmt.Println("Cleaning up workspaces...")
		for _, agent := range session.Agents {
			fmt.Printf("  Removing %s\n", agent.ID)
			if err := parallel.CleanupWorkspace(client, repoRoot, agent.ID); err != nil {
				fmt.Printf("  Warning: %v\n", err)
			}
		}
	}

	// Remove parallel session from description
	newDesc := removeParallelSession(parentDesc)
	if err := client.SetDescription(parentRev, newDesc); err != nil {
		return fmt.Errorf("update description: %w", err)
	}

	// Mark parent done if all complete
	if len(incomplete) == 0 {
		if err := setTaskFlag(parentRev, "done"); err != nil {
			fmt.Printf("Warning: failed to mark parent done: %v\n", err)
		} else {
			fmt.Printf("Marked %s as done\n", parentRev)
		}
	}

	fmt.Println("Parallel session stopped")
	return nil
}

func removeParallelSession(desc string) string {
	lines := strings.Split(desc, "\n")
	var result []string
	var inParallelSection bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## Parallel Session") {
			inParallelSection = true
			continue
		}

		if inParallelSection {
			if strings.HasPrefix(trimmed, "## ") {
				inParallelSection = false
				result = append(result, line)
			}
			continue
		}

		result = append(result, line)
	}

	// Clean up extra blank lines
	return strings.TrimSpace(strings.Join(result, "\n")) + "\n"
}
