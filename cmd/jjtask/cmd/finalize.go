package cmd

import (
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
)

var finalizeCmd = &cobra.Command{
	Use:   "finalize [rev]",
	Short: "Strip task prefix for final commit",
	Long: `Remove the [task:*] prefix from a revision description.

This is typically used after a task is marked done and ready
to be treated as a regular commit.

Examples:
  jjtask finalize @
  jjtask finalize mxyz`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rev := "@"
		if len(args) == 1 {
			rev = args[0]
		}

		desc, err := client.GetDescription(rev)
		if err != nil {
			return fmt.Errorf("failed to get description: %w", err)
		}

		// Remove [task:*] prefix
		pattern := regexp.MustCompile(`^\[task:\w+\]\s*`)
		newDesc := pattern.ReplaceAllString(desc, "")

		if newDesc == desc {
			fmt.Println("No [task:*] prefix found")
			return nil
		}

		return client.SetDescription(rev, newDesc)
	},
}

func init() {
	rootCmd.AddCommand(finalizeCmd)
	finalizeCmd.ValidArgsFunction = completeTaskRevision
}
