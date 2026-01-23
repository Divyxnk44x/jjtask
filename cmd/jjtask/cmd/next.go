package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

var nextMarkAs string
var nextFormat string

type NextTaskBrief struct {
	ChangeID string `json:"change_id"`
	Flag     string `json:"flag"`
	Title    string `json:"title"`
}

type NextOutput struct {
	Revision       string          `json:"revision"`
	ChangeID       string          `json:"change_id"`
	Description    string          `json:"description"`
	IsTask         bool            `json:"is_task"`
	CurrentFlag    string          `json:"current_flag,omitempty"`
	MarkedAs       string          `json:"marked_as,omitempty"`
	NextTasks      []NextTaskBrief `json:"next_tasks,omitempty"`
	StaleTasks     []string        `json:"stale_tasks,omitempty"`
	AvailableFlags []string        `json:"available_flags,omitempty"`
}

var nextCmd = &cobra.Command{
	Use:   "next [rev]",
	Short: "Review current task or transition to next",
	Long: `Review the current task specification, optionally marking it
with a new status and transitioning to the next task.

Without --mark-as, shows the current task's full description.
With --mark-as, updates the status and shows next task options.

Examples:
  jjtask next                    # Review current task
  jjtask next --mark-as done     # Mark current done, show next
  jjtask next --mark-as wip xyz  # Mark xyz as wip`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rev := "@"
		if len(args) == 1 {
			rev = args[0]
		}

		if nextFormat == "json" {
			return nextJSON(cmd, rev)
		}

		// Text output mode
		if nextMarkAs != "" {
			if !slices.Contains(validFlags, nextMarkAs) {
				return fmt.Errorf("invalid flag %q, must be one of: %s", nextMarkAs, strings.Join(validFlags, ", "))
			}

			desc, err := client.GetDescription(rev)
			if err != nil {
				return fmt.Errorf("failed to get description: %w", err)
			}

			if !strings.HasPrefix(desc, "[task:") {
				return fmt.Errorf("revision %s is not a task", rev)
			}

			_ = client.Run("--ignore-working-copy", "log", "-r", rev, "-n1", "--no-graph", "-T", "description")

			fmt.Printf("\nMarking %s as %s...\n", rev, nextMarkAs)

			flagArgs := []string{rev, nextMarkAs}
			if err := flagCmd.RunE(cmd, flagArgs); err != nil {
				return err
			}

			fmt.Println("\nNext tasks:")
			if err := client.Run("log", "-r", "tasks_next()", "-T", "task_log"); err != nil {
				fmt.Println("  (no ready tasks)")
			}

			stale, err := client.Query("log", "-r", "tasks_stale()", "--no-graph", "-T", "change_id.shortest() ++ \" \"")
			if err == nil && strings.TrimSpace(stale) != "" {
				fmt.Printf("\nStale tasks: %s- consider: jjtask hoist\n", stale)
			}

			return nil
		}

		desc, err := client.GetDescription(rev)
		if err != nil {
			return fmt.Errorf("failed to get description: %w", err)
		}

		fmt.Println(desc)

		if strings.HasPrefix(desc, "[task:") {
			fmt.Println("\n---")
			fmt.Println("Transitions: jjtask next --mark-as <flag> [rev]")
			fmt.Println("Flags: draft, todo, wip, untested, standby, review, blocked, done")
		}

		return nil
	},
}

func init() {
	nextCmd.Flags().StringVar(&nextMarkAs, "mark-as", "", "Mark task with new status flag")
	nextCmd.Flags().StringVar(&nextFormat, "format", "text", "Output format: text or json")
	rootCmd.AddCommand(nextCmd)

	nextCmd.ValidArgsFunction = completeTaskRevision
	_ = nextCmd.RegisterFlagCompletionFunc("mark-as", completeTaskFlag)
}

func nextJSON(cmd *cobra.Command, rev string) error {
	taskFlagRe := regexp.MustCompile(`\[task:(\w+)\]`)

	desc, err := client.GetDescription(rev)
	if err != nil {
		return fmt.Errorf("failed to get description: %w", err)
	}

	changeID, err := client.Query("log", "-r", rev, "--no-graph", "-T", "change_id.shortest()")
	if err != nil {
		return fmt.Errorf("get change ID for %s: %w", rev, err)
	}
	changeID = strings.TrimSpace(changeID)

	output := NextOutput{
		Revision:    rev,
		ChangeID:    changeID,
		Description: desc,
		IsTask:      strings.HasPrefix(desc, "[task:"),
	}

	if output.IsTask {
		if match := taskFlagRe.FindStringSubmatch(desc); match != nil {
			output.CurrentFlag = match[1]
		}
		output.AvailableFlags = validFlags
	}

	// Handle --mark-as in JSON mode
	if nextMarkAs != "" {
		if !slices.Contains(validFlags, nextMarkAs) {
			return fmt.Errorf("invalid flag %q, must be one of: %s", nextMarkAs, strings.Join(validFlags, ", "))
		}
		if !output.IsTask {
			return fmt.Errorf("revision %s is not a task", rev)
		}

		flagArgs := []string{rev, nextMarkAs}
		if err := flagCmd.RunE(cmd, flagArgs); err != nil {
			return err
		}
		output.MarkedAs = nextMarkAs
	}

	// Get next tasks
	tmpl := `change_id.shortest() ++ "\t" ++ description.first_line() ++ "\n"`
	nextOut, err := client.Query("log", "-r", "tasks_next()", "--no-graph", "-T", tmpl)
	if err == nil && strings.TrimSpace(nextOut) != "" {
		for _, line := range strings.Split(strings.TrimSpace(nextOut), "\n") {
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) < 2 {
				continue
			}
			brief := NextTaskBrief{ChangeID: parts[0]}
			if match := taskFlagRe.FindStringSubmatch(parts[1]); match != nil {
				brief.Flag = match[1]
				brief.Title = strings.TrimSpace(taskFlagRe.ReplaceAllString(parts[1], ""))
			} else {
				brief.Title = parts[1]
			}
			output.NextTasks = append(output.NextTasks, brief)
		}
	}

	// Get stale tasks
	staleOut, err := client.Query("log", "-r", "tasks_stale()", "--no-graph", "-T", "change_id.shortest() ++ \"\\n\"")
	if err == nil && strings.TrimSpace(staleOut) != "" {
		for _, id := range strings.Split(strings.TrimSpace(staleOut), "\n") {
			if id != "" {
				output.StaleTasks = append(output.StaleTasks, id)
			}
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}
