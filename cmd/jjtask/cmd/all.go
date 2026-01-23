package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"jjtask/internal/workspace"
)

var allCmd = &cobra.Command{
	Use:                "all <jj-command> [args...]",
	Short:              "Run jj command across all workspaces",
	Long:               `Run a jj command across all repositories in a multi-workspace setup.`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("usage: jjtask all <jj-command> [args...]")
		}

		repos, workspaceRoot, err := workspace.GetRepos()
		if err != nil {
			return err
		}

		isMulti := len(repos) > 1

		// Show context hint
		if hint := workspace.ContextHint(); hint != "" {
			fmt.Println(hint)
			fmt.Println()
		}

		for _, repo := range repos {
			repoPath := workspace.ResolveRepoPath(repo, workspaceRoot)

			if isMulti {
				displayPath := workspace.RelativePath(repoPath)
				fmt.Printf("=== %s: jj -R %s %s ===\n", workspace.DisplayName(repo), displayPath, args[0])
			}

			// Build jj command
			jjArgs := []string{}
			if globals.Color != "" {
				jjArgs = append(jjArgs, "--color", globals.Color)
			} else if client.IsTTY {
				jjArgs = append(jjArgs, "--color=always")
			}
			jjArgs = append(jjArgs, "-R", repoPath)
			jjArgs = append(jjArgs, args...)

			jjCmd := exec.Command("jj", jjArgs...)
			jjCmd.Stdin = os.Stdin
			jjCmd.Stdout = os.Stdout
			jjCmd.Stderr = os.Stderr
			jjCmd.Env = append(os.Environ(), "JJ_ALLOW_TASK=1", "JJ_NO_HINTS=1")

			err := jjCmd.Run()
			if err != nil && isMulti {
				// In multi-repo mode, continue on error
				if client.IsTTY {
					fmt.Println("~  \033[32m(no output)\033[0m")
				} else {
					fmt.Println("~  (no output)")
				}
			} else if err != nil {
				return err
			}

			if isMulti {
				fmt.Println()
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(allCmd)
}
