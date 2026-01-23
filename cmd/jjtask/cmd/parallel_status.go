package cmd

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"jjtask/internal/parallel"
)

var parallelStatusCmd = &cobra.Command{
	Use:   "parallel-status [parent-task]",
	Short: "Show status of parallel agent session",
	Long: `View status of all agents in a parallel session.

Shows each agent's progress, file changes, and potential conflicts.

Examples:
  jjtask parallel-status       # auto-detect from current context
  jjtask parallel-status abc   # explicit parent task`,
	Args: cobra.MaximumNArgs(1),
	RunE: runParallelStatus,
}

func init() {
	rootCmd.AddCommand(parallelStatusCmd)
	parallelStatusCmd.ValidArgsFunction = completeRevision
}

type agentStatus struct {
	ID           string
	TaskID       string
	Flag         string
	FilesChanged int
	LinesAdded   int
	LinesRemoved int
}

func runParallelStatus(cmd *cobra.Command, args []string) error {
	var session *parallel.Session
	var parentRev, parentDesc string
	var err error

	if len(args) > 0 {
		parentRev = args[0]
		parentDesc, err = client.GetDescription(parentRev)
		if err != nil {
			return fmt.Errorf("get description: %w", err)
		}
		session, err = parallel.ParseSession(parentDesc)
		if err != nil {
			return fmt.Errorf("parse session from %s: %w", parentRev, err)
		}
	} else {
		session, parentRev, parentDesc, err = findParallelSession()
		if err != nil {
			return err
		}
	}

	if session == nil {
		return fmt.Errorf("no parallel session found")
	}

	// Get parent title
	parentTitle := ""
	lines := strings.Split(parentDesc, "\n")
	if len(lines) > 0 {
		parentTitle = strings.TrimSpace(lines[0])
	}

	// Print header
	fmt.Printf("Parallel Session: %s %s\n\n", parentRev, parentTitle)
	fmt.Printf("Mode: %s\n", session.Mode)
	if !session.Started.IsZero() {
		fmt.Printf("Started: %s ago\n", formatDuration(time.Since(session.Started)))
	}
	fmt.Println()

	// Collect agent statuses
	var statuses []agentStatus
	for _, agent := range session.Agents {
		status := agentStatus{
			ID:     agent.ID,
			TaskID: agent.TaskID,
		}

		// Get task flag and file stats
		if session.Mode == "workspace" && agent.TaskID != "" {
			status.Flag = getTaskFlag(agent.TaskID)
			status.FilesChanged, status.LinesAdded, status.LinesRemoved = getFileStats(agent.TaskID)
		} else {
			status.Flag = getTaskFlag(parentRev)
			// For shared mode, would need to filter by file pattern
		}

		statuses = append(statuses, status)
	}

	// Print status table
	fmt.Printf("%-12s %-10s %-12s %s\n", "Agent", "Status", "Task", "Changes")
	fmt.Printf("%-12s %-10s %-12s %s\n", "-----", "------", "----", "-------")
	for _, s := range statuses {
		task := s.TaskID
		if task == "" {
			task = "(shared)"
		}
		changes := fmt.Sprintf("%d files (+%d/-%d)", s.FilesChanged, s.LinesAdded, s.LinesRemoved)
		if s.FilesChanged == 0 {
			changes = "no changes"
		}
		fmt.Printf("%-12s %-10s %-12s %s\n", s.ID, s.Flag, task, changes)
	}
	fmt.Println()

	// Check for conflicts
	if session.Mode == "workspace" {
		fileConflicts, err := parallel.FindFileConflicts(client, session)
		if err != nil {
			fmt.Printf("Warning: could not check conflicts: %v\n", err)
		}
		if len(fileConflicts) > 0 {
			fmt.Println("File Conflicts:")
			for _, c := range fileConflicts {
				fmt.Printf("  %s modified by: %s\n", c.File, strings.Join(c.Agents, ", "))
			}
		} else {
			fmt.Println("File Conflicts: none")
		}
		fmt.Println()
	} else {
		// Check pattern overlaps for shared mode
		warnings := parallel.CheckPatternOverlaps(session)
		if len(warnings) > 0 {
			fmt.Println("Pattern Overlap Warnings:")
			for _, w := range warnings {
				fmt.Printf("  - %s\n", w)
			}
			fmt.Println()
		}
	}

	// Show DAG
	dagOutput, err := client.Query("log", "-r", fmt.Sprintf("(%s):: & tasks()", parentRev))
	if err == nil && strings.TrimSpace(dagOutput) != "" {
		fmt.Println("DAG:")
		fmt.Println(dagOutput)
	}

	return nil
}

func getTaskFlag(rev string) string {
	desc, err := client.GetDescription(rev)
	if err != nil {
		return "unknown"
	}
	lines := strings.Split(desc, "\n")
	if len(lines) == 0 {
		return "none"
	}
	// Extract [task:FLAG]
	re := regexp.MustCompile(`\[task:(\w+)\]`)
	match := re.FindStringSubmatch(lines[0])
	if match != nil {
		return match[1]
	}
	return "none"
}

func getFileStats(rev string) (files, added, removed int) {
	out, err := client.Query("diff", "-r", rev, "--stat")
	if err != nil || strings.TrimSpace(out) == "" {
		return 0, 0, 0
	}

	// Parse stat output - last line usually has summary
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "file") {
			// Match patterns like "3 files changed, 120 insertions(+), 5 deletions(-)"
			re := regexp.MustCompile(`(\d+) file`)
			if m := re.FindStringSubmatch(line); m != nil {
				_, _ = fmt.Sscanf(m[1], "%d", &files)
			}
		}
		if strings.Contains(line, "|") {
			files++
			// Count +/- from individual file lines
			plusCount := strings.Count(line, "+")
			minusCount := strings.Count(line, "-") - 1 // minus one for the separator
			added += plusCount
			removed += minusCount
		}
	}
	return
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}
