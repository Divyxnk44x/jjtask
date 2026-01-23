package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var jjCompletionCmd = &cobra.Command{
	Use:    "jj-completion [fish]",
	Short:  "Generate jj task completion wrapper",
	Long:   `Generate shell completions for "jj task" subcommand.`,
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "fish":
			fmt.Print(fishJJCompletion)
			return nil
		default:
			return fmt.Errorf("only fish is supported for jj-completion")
		}
	},
}

func init() {
	rootCmd.AddCommand(jjCompletionCmd)
}

// Fish completion that adds "jj task" subcommand completions
// This composes with jj's own completions
var fishJJCompletion = strings.TrimSpace(`
# jjtask completions for "jj task" subcommand
# Source: jjtask jj-completion fish

# Helper to check if we're completing "jj task ..."
function __fish_jj_using_task
    set -l cmd (commandline -opc)
    set -e cmd[1]
    # Skip global flags
    while set -q cmd[1]
        switch $cmd[1]
            case '-*'
                set -e cmd[1]
            case '*'
                break
        end
    end
    test "$cmd[1]" = "task"
end

function __fish_jj_task_needs_subcommand
    set -l cmd (commandline -opc)
    set -e cmd[1]
    # Skip until we find "task"
    while set -q cmd[1]
        if test "$cmd[1]" = "task"
            set -e cmd[1]
            # Skip task's flags
            while set -q cmd[1]
                switch $cmd[1]
                    case '-*'
                        set -e cmd[1]
                    case '*'
                        # Found a subcommand
                        return 1
                end
            end
            return 0
        end
        set -e cmd[1]
    end
    return 1
end

function __fish_jj_task_using_subcommand
    set -l cmd (commandline -opc)
    set -e cmd[1]
    set -l found_task 0
    set -l subcmd ""
    # Find "task" then get subcommand
    for c in $cmd
        if test $found_task -eq 1
            switch $c
                case '-*'
                    continue
                case '*'
                    set subcmd $c
                    break
            end
        else if test "$c" = "task"
            set found_task 1
        end
    end
    test -n "$subcmd"; and contains -- $subcmd $argv
end

# Dynamic completion for revisions
function __fish_jjtask_complete_revisions
    jjtask-go __complete show-desc "" 2>/dev/null | while read -l line
        switch $line
            case ':*'
                # Skip directive
            case '*'
                echo $line
        end
    end
end

# Dynamic completion for task revisions only
function __fish_jjtask_complete_task_revisions
    jjtask-go __complete flag "" 2>/dev/null | while read -l line
        switch $line
            case ':*'
                # Skip directive
            case '*'
                echo $line
        end
    end
end

# Add "task" to jj's subcommand list
complete -c jj -n "__fish_jj_needs_command" -f -a "task" -d 'Task management (jjtask)'

# Task subcommands
complete -c jj -n "__fish_jj_using_task; and __fish_jj_task_needs_subcommand" -f -a "find" -d 'List tasks'
complete -c jj -n "__fish_jj_using_task; and __fish_jj_task_needs_subcommand" -f -a "create" -d 'Create a new task'
complete -c jj -n "__fish_jj_using_task; and __fish_jj_task_needs_subcommand" -f -a "flag" -d 'Update task status'
complete -c jj -n "__fish_jj_using_task; and __fish_jj_task_needs_subcommand" -f -a "next" -d 'Review/transition tasks'
complete -c jj -n "__fish_jj_using_task; and __fish_jj_task_needs_subcommand" -f -a "finalize" -d 'Strip task prefix'
complete -c jj -n "__fish_jj_using_task; and __fish_jj_task_needs_subcommand" -f -a "parallel" -d 'Create sibling tasks'
complete -c jj -n "__fish_jj_using_task; and __fish_jj_task_needs_subcommand" -f -a "hoist" -d 'Rebase pending tasks'
complete -c jj -n "__fish_jj_using_task; and __fish_jj_task_needs_subcommand" -f -a "show-desc" -d 'Print description'
complete -c jj -n "__fish_jj_using_task; and __fish_jj_task_needs_subcommand" -f -a "desc-transform" -d 'Transform description'
complete -c jj -n "__fish_jj_using_task; and __fish_jj_task_needs_subcommand" -f -a "batch-desc" -d 'Transform multiple'
complete -c jj -n "__fish_jj_using_task; and __fish_jj_task_needs_subcommand" -f -a "checkpoint" -d 'Create checkpoint'
complete -c jj -n "__fish_jj_using_task; and __fish_jj_task_needs_subcommand" -f -a "prime" -d 'Session context'
complete -c jj -n "__fish_jj_using_task; and __fish_jj_task_needs_subcommand" -f -a "all" -d 'Run across workspaces'
complete -c jj -n "__fish_jj_using_task; and __fish_jj_task_needs_subcommand" -f -a "completion" -d 'Generate completions'

# find: flag filter
complete -c jj -n "__fish_jj_task_using_subcommand find" -f -a "pending" -d 'All non-done (default)'
complete -c jj -n "__fish_jj_task_using_subcommand find" -f -a "todo" -d 'Todo tasks'
complete -c jj -n "__fish_jj_task_using_subcommand find" -f -a "wip" -d 'Work in progress'
complete -c jj -n "__fish_jj_task_using_subcommand find" -f -a "done" -d 'Completed'
complete -c jj -n "__fish_jj_task_using_subcommand find" -f -a "blocked" -d 'Blocked'
complete -c jj -n "__fish_jj_task_using_subcommand find" -f -a "draft" -d 'Drafts'
complete -c jj -n "__fish_jj_task_using_subcommand find" -f -a "review" -d 'Needs review'
complete -c jj -n "__fish_jj_task_using_subcommand find" -f -a "all" -d 'All tasks'

# flag: revision then flag value
complete -c jj -n "__fish_jj_task_using_subcommand flag" -f -a "(__fish_jjtask_complete_task_revisions)"
complete -c jj -n "__fish_jj_task_using_subcommand flag" -f -a "draft" -d 'Placeholder'
complete -c jj -n "__fish_jj_task_using_subcommand flag" -f -a "todo" -d 'Ready to work'
complete -c jj -n "__fish_jj_task_using_subcommand flag" -f -a "wip" -d 'In progress'
complete -c jj -n "__fish_jj_task_using_subcommand flag" -f -a "blocked" -d 'Waiting'
complete -c jj -n "__fish_jj_task_using_subcommand flag" -f -a "standby" -d 'Awaits decision'
complete -c jj -n "__fish_jj_task_using_subcommand flag" -f -a "untested" -d 'Needs testing'
complete -c jj -n "__fish_jj_task_using_subcommand flag" -f -a "review" -d 'Needs review'
complete -c jj -n "__fish_jj_task_using_subcommand flag" -f -a "done" -d 'Complete'

# next: --mark-as flag and revision
complete -c jj -n "__fish_jj_task_using_subcommand next" -l mark-as -f -a "draft todo wip blocked standby untested review done" -d 'Mark with status'
complete -c jj -n "__fish_jj_task_using_subcommand next" -f -a "(__fish_jjtask_complete_task_revisions)"

# finalize: task revision
complete -c jj -n "__fish_jj_task_using_subcommand finalize" -f -a "(__fish_jjtask_complete_task_revisions)"

# show-desc: any revision
complete -c jj -n "__fish_jj_task_using_subcommand show-desc" -f -a "(__fish_jjtask_complete_revisions)"

# desc-transform: any revision
complete -c jj -n "__fish_jj_task_using_subcommand desc-transform" -f -a "(__fish_jjtask_complete_revisions)"

# create: revision for parent
complete -c jj -n "__fish_jj_task_using_subcommand create" -l draft -d 'Create as draft'
complete -c jj -n "__fish_jj_task_using_subcommand create" -f -a "(__fish_jjtask_complete_revisions)"
`) + "\n"
