package cmd

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

var validFlags = []string{"draft", "todo", "wip", "untested", "standby", "review", "blocked", "done"}

var flagCmd = &cobra.Command{
	Use:   "flag <rev> <flag>",
	Short: "Update task status flag",
	Long: `Update the [task:*] flag in a revision description.

Valid flags: draft, todo, wip, untested, standby, review, blocked, done

Examples:
  jjtask flag @ wip
  jjtask flag mxyz done`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		rev := args[0]
		toFlag := args[1]

		if !slices.Contains(validFlags, toFlag) {
			return fmt.Errorf("invalid flag %q, must be one of: %s", toFlag, strings.Join(validFlags, ", "))
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

		// After marking done, check for stale tasks
		if toFlag == "done" {
			stale, err := client.Query("log", "-r", "tasks_stale()", "--no-graph", "-T", "change_id.shortest() ++ \" \"")
			if err == nil && strings.TrimSpace(stale) != "" {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Stale tasks: %s- consider: jjtask hoist or jj rebase -s TASK -d @\n", stale)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(flagCmd)

	// arg 0: revision, arg 1: flag
	flagCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeTaskRevision(cmd, args, toComplete)
		}
		if len(args) == 1 {
			return completeTaskFlag(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}
