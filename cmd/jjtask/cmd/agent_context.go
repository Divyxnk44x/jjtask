package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"jjtask/internal/parallel"
)

var agentContextFormat string

var agentContextCmd = &cobra.Command{
	Use:   "agent-context <agent-id>",
	Short: "Get context for a parallel agent",
	Long: `Output context information for an agent in a parallel session.

Shows the agent's assignment, files to avoid (other agents), and DAG state.

Examples:
  jjtask agent-context agent-a
  jjtask agent-context agent-b --format json`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentContext,
}

func init() {
	agentContextCmd.Flags().StringVar(&agentContextFormat, "format", "text", "Output format: text or json")
	rootCmd.AddCommand(agentContextCmd)
}

type AgentContextOutput struct {
	AgentID      string   `json:"agent_id"`
	Mode         string   `json:"mode"`
	Workspace    string   `json:"workspace,omitempty"`
	TaskID       string   `json:"task_id,omitempty"`
	Assignment   string   `json:"assignment"`
	AvoidFiles   []string `json:"avoid_files"`
	OtherAgents  []string `json:"other_agents"`
	ParentTask   string   `json:"parent_task"`
	ParentTaskID string   `json:"parent_task_id"`
}

func runAgentContext(cmd *cobra.Command, args []string) error {
	agentID := args[0]

	// Find parent task with parallel session
	session, parentRev, parentDesc, err := findParallelSession()
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("no parallel session found; start one with: jjtask parallel-start")
	}

	// Find this agent
	agent := session.GetAgentByID(agentID)
	if agent == nil {
		var available []string
		for _, a := range session.Agents {
			available = append(available, a.ID)
		}
		return fmt.Errorf("agent %q not in session; available: %s", agentID, strings.Join(available, ", "))
	}

	// Build output
	output := AgentContextOutput{
		AgentID:      agentID,
		Mode:         session.Mode,
		Assignment:   agent.FilePattern,
		ParentTaskID: parentRev,
	}

	// Extract parent title
	lines := strings.Split(parentDesc, "\n")
	if len(lines) > 0 {
		output.ParentTask = strings.TrimSpace(lines[0])
	}

	// Workspace path
	if session.Mode == "workspace" && agent.TaskID != "" {
		output.TaskID = agent.TaskID
		repoRoot, err := client.Root()
		if err != nil {
			return fmt.Errorf("get repo root: %w", err)
		}
		output.Workspace = filepath.Join(repoRoot, parallel.WorkspacesDir, agentID)
	}

	// Other agents' assignments
	for _, other := range session.OtherAgents(agentID) {
		output.OtherAgents = append(output.OtherAgents, other.ID)
		if other.FilePattern != "" {
			output.AvoidFiles = append(output.AvoidFiles, fmt.Sprintf("%s (%s)", other.FilePattern, other.ID))
		}
	}

	if agentContextFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	// Text output
	printAgentContext(output, session, parentRev)
	return nil
}

func printAgentContext(ctx AgentContextOutput, _ *parallel.Session, parentRev string) {
	fmt.Printf("## Agent Context: %s\n\n", ctx.AgentID)
	fmt.Printf("Mode: %s\n", ctx.Mode)

	if ctx.Workspace != "" {
		fmt.Printf("Workspace: %s\n", ctx.Workspace)
	}
	if ctx.TaskID != "" {
		fmt.Printf("Task: %s\n", ctx.TaskID)
	}

	fmt.Println()

	if ctx.Assignment != "" {
		fmt.Println("### Your Assignment")
		fmt.Println(ctx.Assignment)
		fmt.Println()
	}

	if len(ctx.AvoidFiles) > 0 {
		fmt.Println("### Files to AVOID (other agents)")
		for _, f := range ctx.AvoidFiles {
			fmt.Printf("- %s\n", f)
		}
		fmt.Println()
	}

	fmt.Println("### Parent Task")
	fmt.Printf("%s %s\n", parentRev, ctx.ParentTask)
	fmt.Println()

	// Show DAG
	dagOutput, err := client.Query("log", "-r", fmt.Sprintf("(%s):: & tasks()", parentRev), "-T", `
separate(" ",
  if(self.contained_in("@"), "←", ""),
  change_id.shortest(),
  description.first_line().substr(0,60)
) ++ "\n"
`)
	if err == nil && strings.TrimSpace(dagOutput) != "" {
		fmt.Println("### DAG State")
		fmt.Println(dagOutput)
	}
}

func findParallelSession() (session *parallel.Session, parentRev, parentDesc string, err error) {
	// Strategy 1: look at @ and its ancestors for a parallel session
	revsToCheck := []string{"@", "@-", "@--"}

	// Also check if we're in a workspace subdirectory
	cwd, _ := os.Getwd()
	if strings.Contains(cwd, parallel.WorkspacesDir) {
		// In a workspace, check the parent workspace's tasks
		revsToCheck = append([]string{"@-"}, revsToCheck...)
	}

	for _, rev := range revsToCheck {
		desc, err := client.GetDescription(rev)
		if err != nil {
			continue
		}

		session, parseErr := parallel.ParseSession(desc)
		if parseErr != nil {
			continue
		}
		if session != nil && len(session.Agents) > 0 {
			// Get the actual change ID
			revID, _ := client.Query("log", "-r", rev, "--no-graph", "-T", "change_id.shortest()")
			return session, strings.TrimSpace(revID), desc, nil
		}
	}

	// Strategy 2: look for any task with "## Parallel Session" in description
	taskRevs, err := client.Query("log", "-r", "tasks()", "--no-graph", "-T", "change_id.shortest() ++ \"\\n\"")
	if err == nil {
		for _, rev := range strings.Split(strings.TrimSpace(taskRevs), "\n") {
			if rev == "" {
				continue
			}
			desc, err := client.GetDescription(rev)
			if err != nil {
				continue
			}
			session, parseErr := parallel.ParseSession(desc)
			if parseErr != nil {
				continue
			}
			if session != nil && len(session.Agents) > 0 {
				return session, rev, desc, nil
			}
		}
	}

	return nil, "", "", nil
}
