package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var hoistCmd = &cobra.Command{
	Use:   "hoist",
	Short: "Rebase pending empty tasks onto @",
	Long: `Rebase all pending empty tasks to be children of @.

This keeps your task DAG connected to your current work after making commits.
Only rebases tasks that are empty (no file changes).

Examples:
  jjtask hoist   # Rebase all pending empty tasks onto @`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Find pending empty tasks not connected to @ (not ancestors or descendants)
		// ~(::@) excludes ancestors, ~(@::) excludes descendants
		revset := "tasks_pending() & empty() & ~(::@ | @::)"

		revsOut, err := client.Query("log", "-r", revset, "--no-graph", "-T", `change_id.shortest() ++ "\n"`)
		if err != nil {
			return fmt.Errorf("failed to find tasks: %w", err)
		}

		var tasks []string
		for _, line := range strings.Split(strings.TrimSpace(revsOut), "\n") {
			if line != "" {
				tasks = append(tasks, line)
			}
		}

		if len(tasks) == 0 {
			fmt.Println("No pending empty tasks to hoist")
			return nil
		}

		// Rebase all roots of tasks onto @ in one command
		// Use roots() to only rebase the top-level tasks, descendants follow
		if err := client.Run("rebase", "-s", "roots("+revset+")", "-d", "@"); err != nil {
			return fmt.Errorf("failed to rebase: %w", err)
		}

		fmt.Printf("Hoisted %d task(s) onto @\n", len(tasks))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(hoistCmd)
}
