package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"jjtask/internal/parallel"
)

var (
	parallelStartMode        string
	parallelStartAgents      int
	parallelStartNames       []string
	parallelStartAssignments []string
)

var parallelStartCmd = &cobra.Command{
	Use:   "parallel-start <parent-task>",
	Short: "Start a parallel agent session",
	Long: `Start a parallel agent session for multi-agent work.

Modes:
  shared    - All agents share the same @ revision (default)
  workspace - Each agent gets a separate jj workspace

Examples:
  jjtask parallel-start --mode shared --agents 2 abc
  jjtask parallel-start --mode workspace --agents 3 xyz
  jjtask parallel-start --names agent-api,agent-ui abc
  jjtask parallel-start --assign "agent-a:src/api/**,agent-b:src/ui/**" abc`,
	Args: cobra.ExactArgs(1),
	RunE: runParallelStart,
}

func init() {
	parallelStartCmd.Flags().StringVar(&parallelStartMode, "mode", "shared", "Session mode: shared or workspace")
	parallelStartCmd.Flags().IntVar(&parallelStartAgents, "agents", 2, "Number of agents")
	parallelStartCmd.Flags().StringSliceVar(&parallelStartNames, "names", nil, "Custom agent names (comma-separated)")
	parallelStartCmd.Flags().StringSliceVar(&parallelStartAssignments, "assign", nil, "Agent assignments (agent:pattern,...)")
	rootCmd.AddCommand(parallelStartCmd)
	parallelStartCmd.ValidArgsFunction = completeRevision
}

func runParallelStart(cmd *cobra.Command, args []string) error {
	parentRev := args[0]

	// Validate mode
	if parallelStartMode != "shared" && parallelStartMode != "workspace" {
		return fmt.Errorf("invalid mode %q: must be 'shared' or 'workspace'", parallelStartMode)
	}

	// Generate agent names
	agentNames := parallelStartNames
	if len(agentNames) == 0 {
		agentNames = generateAgentNames(parallelStartAgents)
	}

	// Get the full change ID of parent to avoid ambiguity after creating children
	fullParentID, err := client.Query("log", "-r", parentRev, "--no-graph", "-T", "change_id", "--limit", "1")
	if err != nil {
		return fmt.Errorf("resolve parent revision: %w", err)
	}
	fullParentID = strings.TrimSpace(fullParentID)

	// Get parent task description to check for existing assignments
	parentDesc, err := client.GetDescription(parentRev)
	if err != nil {
		return fmt.Errorf("get parent description: %w", err)
	}

	// Check if session already exists
	existingSession, err := parallel.ParseSession(parentDesc)
	if err != nil {
		return fmt.Errorf("parse existing session: %w", err)
	}
	if existingSession != nil && len(existingSession.Agents) > 0 {
		return fmt.Errorf("parallel session already exists on %s; use parallel-stop first", parentRev)
	}

	// Build session
	session := &parallel.Session{
		Mode:    parallelStartMode,
		Started: time.Now(),
		Agents:  make([]parallel.Agent, len(agentNames)),
	}

	// Parse assignments from flag
	assignments := parseAssignments(parallelStartAssignments)

	for i, name := range agentNames {
		session.Agents[i] = parallel.Agent{
			ID:          name,
			FilePattern: assignments[name],
			Description: fmt.Sprintf("Agent %d", i+1),
		}
	}

	// Warn if no assignments in shared mode
	if parallelStartMode == "shared" {
		hasAssignments := false
		for _, a := range session.Agents {
			if a.FilePattern != "" {
				hasAssignments = true
				break
			}
		}
		if !hasAssignments {
			fmt.Println("Warning: No file assignments specified for shared mode")
			fmt.Println("Use --assign to specify: --assign \"agent-a:src/api/**,agent-b:src/ui/**\"")
			fmt.Println("Without assignments, agents may conflict by editing the same files")
			fmt.Println()
		}
	}

	// Get repo root for workspace mode
	repoRoot, err := client.Root()
	if err != nil {
		return fmt.Errorf("get repo root: %w", err)
	}

	if parallelStartMode == "workspace" {
		if err := setupWorkspaceMode(session, repoRoot, fullParentID); err != nil {
			return err
		}
	}

	// Mark parent as wip
	if err := setTaskFlag(fullParentID, "wip"); err != nil {
		return fmt.Errorf("mark parent wip: %w", err)
	}

	// Update parent description with session info
	newDesc := parallel.UpdateDescription(parentDesc, session)
	if err := client.SetDescription(fullParentID, newDesc); err != nil {
		return fmt.Errorf("update parent description: %w", err)
	}

	// Print output
	printSessionStarted(session, repoRoot)

	return nil
}

func setupWorkspaceMode(session *parallel.Session, repoRoot, parentRev string) error {
	// Ensure .jjtask-workspaces is ignored
	if err := parallel.EnsureIgnored(repoRoot); err != nil {
		return fmt.Errorf("setup gitignore: %w", err)
	}

	// Create child tasks and workspaces for each agent
	for i := range session.Agents {
		agent := &session.Agents[i]

		// Create child task
		taskTitle := fmt.Sprintf("[task:wip] %s task", agent.ID)
		if err := client.Run("new", "--no-edit", parentRev, "-m", taskTitle); err != nil {
			return fmt.Errorf("create task for %s: %w", agent.ID, err)
		}

		// Get the change ID of the new task
		taskID, err := client.Query("log", "-r", fmt.Sprintf("children(%s) & description(substring:%q)", parentRev, agent.ID), "--no-graph", "-T", "change_id.shortest()", "--limit", "1")
		if err != nil {
			return fmt.Errorf("get task ID for %s: %w", agent.ID, err)
		}
		agent.TaskID = strings.TrimSpace(taskID)

		// Create workspace
		_, err = parallel.CreateWorkspace(client, repoRoot, agent.ID, agent.TaskID)
		if err != nil {
			return fmt.Errorf("create workspace for %s: %w", agent.ID, err)
		}
	}

	return nil
}

func printSessionStarted(session *parallel.Session, _ string) {
	fmt.Printf("Parallel session started (mode: %s)\n\n", session.Mode)

	for _, agent := range session.Agents {
		fmt.Printf("Agent: %s\n", agent.ID)
		if session.Mode == "workspace" {
			fmt.Printf("  Workspace: %s/%s\n", parallel.WorkspacesDir, agent.ID)
			fmt.Printf("  Task: %s\n", agent.TaskID)
		}
		if agent.FilePattern != "" {
			fmt.Printf("  Assignment: %s\n", agent.FilePattern)
		}
		fmt.Println()
	}

	fmt.Println("To get agent context: jjtask agent-context <agent-id>")
	if session.Mode == "shared" {
		fmt.Println("Note: Agents share @ - coordinate file assignments before starting")
	}
}

func generateAgentNames(n int) []string {
	names := make([]string, n)
	for i := range n {
		names[i] = fmt.Sprintf("agent-%c", 'a'+i)
	}
	return names
}

func setTaskFlag(rev, flag string) error {
	desc, err := client.GetDescription(rev)
	if err != nil {
		return err
	}

	// Replace [task:*] with [task:flag]
	lines := strings.Split(desc, "\n")
	if len(lines) > 0 {
		lines[0] = replaceTaskFlag(lines[0], flag)
	}
	return client.SetDescription(rev, strings.Join(lines, "\n"))
}

func replaceTaskFlag(line, newFlag string) string {
	// Match [task:*] pattern
	start := strings.Index(line, "[task:")
	if start == -1 {
		return line
	}
	end := strings.Index(line[start:], "]")
	if end == -1 {
		return line
	}
	return line[:start] + "[task:" + newFlag + "]" + line[start+end+1:]
}

// parseAssignments parses "agent:pattern,agent:pattern" format
func parseAssignments(assignments []string) map[string]string {
	result := make(map[string]string)
	for _, a := range assignments {
		// Handle both comma-separated in one string and multiple flags
		parts := strings.Split(a, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if idx := strings.Index(part, ":"); idx > 0 {
				agentID := strings.TrimSpace(part[:idx])
				pattern := strings.TrimSpace(part[idx+1:])
				result[agentID] = pattern
			}
		}
	}
	return result
}
