package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var createDraft bool

var createCmd = &cobra.Command{
	Use:   "create <parent> <title> [description]",
	Short: "Create a new task revision",
	Long: `Create a new task revision as a child of parent.

Parent is required to ensure tasks form a proper DAG.
Use @ for current revision, or a task ID to chain tasks.

Examples:
  jjtask create @ "Fix bug"
  jjtask create @ "Fix bug" "## Context\nDetails here"
  jjtask create --draft @ "Future work"
  jjtask create mxyz "Subtask" "Chains after task mxyz"`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		var parent, title, desc string

		parent = args[0]
		title = args[1]
		if len(args) == 3 {
			desc = args[2]
		}

		flag := "todo"
		if createDraft {
			flag = "draft"
		}

		message := fmt.Sprintf("[task:%s] %s", flag, title)
		if desc != "" {
			message = message + "\n\n" + desc
		}

		err := client.Run("new", "--no-edit", parent, "-m", message)
		if err != nil {
			return err
		}

		// Get the created revision's change ID
		out, err := client.Query("log", "-r", "children("+parent+") & description(substring:\"[task:\") & heads(all())", "--no-graph", "-T", "change_id.shortest()", "--limit", "1")
		if err != nil {
			fmt.Printf("Created task [task:%s] %s (could not resolve ID: %v)\n", flag, title, err)
			return nil
		}
		changeID := strings.TrimSpace(out)
		if changeID == "" {
			fmt.Printf("Created task [task:%s] %s\n", flag, title)
		} else {
			fmt.Printf("Created new commit %s (empty) [task:%s] %s\n", changeID, flag, title)
		}

		return nil
	},
}

func init() {
	createCmd.Flags().BoolVar(&createDraft, "draft", false, "Create with [task:draft] flag")
	rootCmd.AddCommand(createCmd)
	createCmd.ValidArgsFunction = completeRevision
}
