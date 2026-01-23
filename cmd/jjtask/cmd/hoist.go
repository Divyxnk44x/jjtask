package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var hoistCmd = &cobra.Command{
	Use:   "hoist",
	Short: "Rebase pending tasks to children of @",
	Long: `Rebase stale pending tasks so they become children of the current
working copy (@).

This is useful after doing work when tasks have become "stale"
(not in the ancestry of @). After hoisting, tasks are children of @
so you can jj edit them to start work.

Examples:
  jjtask hoist`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Find task roots that need hoisting (stale tasks that are roots of pending subtrees)
		out, err := client.Query("log", "-r", "roots(tasks_stale())", "--no-graph", "-T", "change_id ++ \"\\n\"")
		if err != nil {
			return fmt.Errorf("failed to find stale tasks: %w", err)
		}

		roots := strings.Split(strings.TrimSpace(out), "\n")
		if len(roots) == 0 || (len(roots) == 1 && roots[0] == "") {
			fmt.Println("No stale tasks to hoist")
			return nil
		}

		// Filter empty strings
		var validRoots []string
		for _, r := range roots {
			r = strings.TrimSpace(r)
			if r != "" {
				validRoots = append(validRoots, r)
			}
		}

		if len(validRoots) == 0 {
			fmt.Println("No stale tasks to hoist")
			return nil
		}

		// Get short IDs for display
		shortIDs := make([]string, 0, len(validRoots))
		for _, r := range validRoots {
			short, err := client.Query("log", "-r", r, "--no-graph", "-T", "change_id.shortest()")
			if err == nil {
				shortIDs = append(shortIDs, strings.TrimSpace(short))
			}
		}

		fmt.Printf("Found %d task root(s) to hoist: %s\n", len(validRoots), strings.Join(shortIDs, " "))

		// Build single rebase command with multiple -s flags
		rebaseArgs := make([]string, 0, 2+len(validRoots)*2)
		for _, root := range validRoots {
			rebaseArgs = append(rebaseArgs, "-s", root)
		}
		rebaseArgs = append(rebaseArgs, "-d", "@")

		fullArgs := append([]string{"rebase"}, rebaseArgs...)
		if err := client.Run(fullArgs...); err != nil {
			return fmt.Errorf("failed to rebase tasks: %w", err)
		}

		fmt.Println("Done")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(hoistCmd)
}
