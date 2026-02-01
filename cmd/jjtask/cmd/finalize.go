package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var finalizeRevset string

var finalizeCmd = &cobra.Command{
	Use:   "finalize [REV]",
	Short: "Strip [task:*] prefix from commits",
	Long: `Remove [task:*] prefix from commit descriptions for clean history.

By default operates on @. Use --revset for multiple commits.

Examples:
  jjtask finalize              # Strip prefix from @
  jjtask finalize abc123       # Strip from specific revision
  jjtask finalize --revset '@-::@'  # Strip from range`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		revset := "@"
		if len(args) > 0 {
			revset = args[0]
		}
		if finalizeRevset != "" {
			revset = finalizeRevset
		}

		// Get revisions matching the revset
		revsOut, err := client.Query("log", "-r", revset, "--no-graph", "-T", `change_id.shortest() ++ "\n"`)
		if err != nil {
			return fmt.Errorf("failed to get revisions: %w", err)
		}

		taskPrefixRe := regexp.MustCompile(`^\[task:\w+\]\s*`)
		count := 0

		for _, rev := range strings.Split(strings.TrimSpace(revsOut), "\n") {
			if rev == "" {
				continue
			}

			desc, err := client.GetDescription(rev)
			if err != nil || desc == "" {
				continue
			}

			if !taskPrefixRe.MatchString(desc) {
				continue
			}

			newDesc := taskPrefixRe.ReplaceAllString(desc, "")
			if newDesc == desc {
				continue
			}

			if err := client.SetDescription(rev, newDesc); err != nil {
				return fmt.Errorf("failed to update %s: %w", rev, err)
			}
			count++

			firstLine := strings.Split(newDesc, "\n")[0]
			if len(firstLine) > 60 {
				firstLine = firstLine[:57] + "..."
			}
			fmt.Printf("%s: %s\n", rev, firstLine)
		}

		if count == 0 {
			fmt.Println("No task prefixes to strip")
		} else {
			fmt.Printf("Finalized %d commit(s)\n", count)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(finalizeCmd)
	finalizeCmd.Flags().StringVarP(&finalizeRevset, "revset", "r", "", "Revset to finalize (for multiple commits)")
	_ = finalizeCmd.RegisterFlagCompletionFunc("revset", completeRevision)
}
