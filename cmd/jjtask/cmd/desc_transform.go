package cmd

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var descTransformCmd = &cobra.Command{
	Use:   "desc-transform <rev> <sed-expr|command...>",
	Short: "Transform revision description",
	Long: `Transform a revision description through a command.

If a single argument starting with 's/' is provided, sed is assumed.
Otherwise, the command and arguments are executed directly.

Examples:
  jjtask desc-transform @ 's/foo/bar/'
  jjtask desc-transform @ sed 's/foo/bar/'
  jjtask desc-transform mxyz awk '/^##/{print}'`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		rev := args[0]
		cmdArgs := args[1:]

		// If single argument starting with s/, assume sed
		if len(cmdArgs) == 1 && strings.HasPrefix(cmdArgs[0], "s/") {
			cmdArgs = []string{"sed", cmdArgs[0]}
		}

		// Get current description
		desc, err := client.GetDescription(rev)
		if err != nil {
			return fmt.Errorf("failed to get description: %w", err)
		}

		// Check command exists
		cmdName := cmdArgs[0]
		if _, err := exec.LookPath(cmdName); err != nil {
			return fmt.Errorf("command %q not found", cmdName)
		}

		// Run transformation
		transformCmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		transformCmd.Stdin = strings.NewReader(desc)
		var stdout, stderr bytes.Buffer
		transformCmd.Stdout = &stdout
		transformCmd.Stderr = &stderr

		if err := transformCmd.Run(); err != nil {
			if stderr.Len() > 0 {
				return fmt.Errorf("transform failed: %s", stderr.String())
			}
			return fmt.Errorf("transform failed: %w", err)
		}

		newDesc := stdout.String()
		if newDesc == desc {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Nothing changed.")
			return nil
		}

		return client.SetDescription(rev, newDesc)
	},
}

func init() {
	rootCmd.AddCommand(descTransformCmd)
	descTransformCmd.ValidArgsFunction = completeRevision
}
