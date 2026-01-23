package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var checkpointCmd = &cobra.Command{
	Use:   "checkpoint [message]",
	Short: "Record operation ID for recovery",
	Long: `Record the current jj operation ID so you can restore to this
point if something goes wrong.

Examples:
  jjtask checkpoint
  jjtask checkpoint "Before risky rebase"`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		message := ""
		if len(args) == 1 {
			message = args[0]
		}

		// Get current operation ID
		opID, err := client.Query("op", "log", "--no-graph", "-T", "id.short()", "--limit", "1")
		if err != nil {
			return fmt.Errorf("failed to get operation ID: %w", err)
		}
		opID = strings.TrimSpace(opID)

		if message != "" {
			fmt.Printf("Checkpoint '%s' at operation: %s\n", message, opID)
		} else {
			fmt.Printf("Checkpoint at operation: %s\n", opID)
		}
		fmt.Printf("  Restore with: jj op restore %s\n", opID)

		// Show current state
		fmt.Println()
		fmt.Println("  Current state:")
		if err := client.Run("log", "-r", "@", "--limit", "3"); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkpointCmd)
}
