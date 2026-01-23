package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var parallelDraft bool

var parallelCmd = &cobra.Command{
	Use:   "parallel <parent> <title1> <title2> [title3...]",
	Short: "Create sibling tasks under parent",
	Long: `Create multiple parallel task branches from the same parent.

Examples:
  jjtask parallel @ "Widget A" "Widget B" "Widget C"
  jjtask parallel --draft @ "Future A" "Future B"`,
	Args: cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		parent := args[0]
		titles := args[1:]

		flag := "todo"
		if parallelDraft {
			flag = "draft"
		}

		for _, title := range titles {
			message := fmt.Sprintf("[task:%s] %s", flag, title)
			if err := client.Run("new", "--no-edit", parent, "-m", message); err != nil {
				return fmt.Errorf("failed to create task %q: %w", title, err)
			}
		}

		fmt.Printf("Created %d parallel task branches from %s\n", len(titles), parent)
		return nil
	},
}

func init() {
	parallelCmd.Flags().BoolVar(&parallelDraft, "draft", false, "Create with [task:draft] flag")
	rootCmd.AddCommand(parallelCmd)
	parallelCmd.ValidArgsFunction = completeRevision
}
