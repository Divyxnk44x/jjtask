package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"jjtask/internal/jj"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for jjtask.

To load completions:

Bash:
  $ source <(jjtask completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ jjtask completion bash > /etc/bash_completion.d/jjtask
  # macOS:
  $ jjtask completion bash > $(brew --prefix)/etc/bash_completion.d/jjtask

Zsh:
  $ source <(jjtask completion zsh)
  # To load completions for each session, execute once:
  $ jjtask completion zsh > "${fpath[1]}/_jjtask"

Fish:
  $ jjtask completion fish | source
  # To load completions for each session, execute once:
  $ jjtask completion fish > ~/.config/fish/completions/jjtask.fish

PowerShell:
  PS> jjtask completion powershell | Out-String | Invoke-Expression
  # To load completions for each session, execute once:
  PS> jjtask completion powershell > jjtask.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(cmd.OutOrStdout())
		case "zsh":
			return rootCmd.GenZshCompletion(cmd.OutOrStdout())
		case "fish":
			return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

// completeRevision provides completion for jj revision arguments
func completeRevision(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	c := jj.NewWithGlobals(globals)

	// Get recent changes with their short descriptions
	output, err := c.Query("log", "-r", "all()", "--limit", "20", "--no-graph",
		"-T", `change_id.shortest() ++ "\t" ++ description.first_line().substr(0, 50) ++ "\n"`)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			completions = append(completions, parts[0]+"\t"+parts[1])
		} else {
			completions = append(completions, parts[0])
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeTaskRevision provides completion for task revision arguments only
func completeTaskRevision(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	c := jj.NewWithGlobals(globals)

	// Get task revisions only
	output, err := c.Query("log", "-r", "tasks()", "--no-graph",
		"-T", `change_id.shortest() ++ "\t" ++ description.first_line().substr(0, 50) ++ "\n"`)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			completions = append(completions, parts[0]+"\t"+parts[1])
		} else {
			completions = append(completions, parts[0])
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeTaskFlag provides completion for task flag values
func completeTaskFlag(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	flags := []string{
		"draft\tPlaceholder, needs specification",
		"todo\tReady to work",
		"wip\tWork in progress",
		"blocked\tWaiting on dependency",
		"standby\tAwaits decision",
		"untested\tNeeds testing",
		"review\tNeeds review",
		"done\tComplete",
	}
	return flags, cobra.ShellCompDirectiveNoFileComp
}

// completeFindFlag provides completion for find command flag argument
func completeFindFlag(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	flags := []string{
		"pending\tAll non-done tasks (default)",
		"todo\tTodo tasks only",
		"wip\tWork in progress",
		"done\tCompleted tasks",
		"blocked\tBlocked tasks",
		"standby\tTasks awaiting decision",
		"untested\tImplementation done, needs testing",
		"draft\tDraft tasks",
		"review\tTasks needing review",
		"all\tAll tasks",
	}
	return flags, cobra.ShellCompDirectiveNoFileComp
}
