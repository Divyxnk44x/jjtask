package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var wipCmd = &cobra.Command{
	Use:   "wip [tasks...]",
	Short: "Mark tasks as WIP and add to @ merge",
	Long: `Mark tasks as WIP and add them as parents of @.

When multiple tasks are WIP, @ becomes a merge showing their combined state.
Work directly in task branches with 'jj edit TASK'.

Examples:
  jjtask wip xyz       # Mark xyz as WIP, add to @ merge
  jjtask wip           # Mark @ as WIP (if it's a task)
  jjtask wip a b c     # Mark multiple tasks as WIP`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		revs := args
		if len(revs) == 0 {
			revs = []string{"@"}
		}

		// Collect change IDs and mark all as WIP first
		var changeIDs []string
		for _, rev := range revs {
			changeID, err := client.Query("log", "-r", rev, "--no-graph", "-T", "change_id.shortest()")
			if err != nil {
				return fmt.Errorf("getting change ID for %s: %w", rev, err)
			}
			changeID = strings.TrimSpace(changeID)
			changeIDs = append(changeIDs, changeID)

			if err := setTaskFlag(rev, "wip"); err != nil {
				return fmt.Errorf("failed to mark %s as WIP: %w", rev, err)
			}
		}

		// Single rebase to add all tasks as parents
		if err := client.AddMultipleToMerge(changeIDs); err != nil {
			return fmt.Errorf("adding tasks to merge: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(wipCmd)
}
