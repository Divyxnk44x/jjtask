package parallel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"jjtask/internal/jj"
)

const WorkspacesDir = ".jjtask-workspaces"

// EnsureWorkspacesDir creates .jjtask-workspaces/ if it doesn't exist
func EnsureWorkspacesDir(repoRoot string) (string, error) {
	dir := filepath.Join(repoRoot, WorkspacesDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create workspaces dir: %w", err)
	}
	return dir, nil
}

// EnsureIgnored adds .jjtask-workspaces/ to .git/info/exclude
func EnsureIgnored(repoRoot string) error {
	gitDir := filepath.Join(repoRoot, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return nil // non-colocated repo, nothing to do
	}

	infoDir := filepath.Join(gitDir, "info")
	if err := os.MkdirAll(infoDir, 0o755); err != nil {
		return fmt.Errorf("create .git/info: %w", err)
	}

	excludePath := filepath.Join(infoDir, "exclude")
	pattern := WorkspacesDir + "/"

	// Read existing content
	content, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read exclude file: %w", err)
	}

	// Check if already present
	if strings.Contains(string(content), pattern) {
		return nil
	}

	// Append pattern
	f, err := os.OpenFile(excludePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open exclude file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Add newline if file doesn't end with one
	if len(content) > 0 && content[len(content)-1] != '\n' {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	if _, err := f.WriteString(pattern + "\n"); err != nil {
		return err
	}

	return nil
}

// CreateWorkspace creates a jj workspace for an agent
func CreateWorkspace(client *jj.Client, repoRoot, agentID, revision string) (string, error) {
	wsDir, err := EnsureWorkspacesDir(repoRoot)
	if err != nil {
		return "", err
	}

	agentDir := filepath.Join(wsDir, agentID)

	// Check if workspace already exists
	if _, err := os.Stat(agentDir); err == nil {
		return agentDir, nil // already exists
	}

	// Create jj workspace
	err = client.Run("workspace", "add", agentDir, "--revision", revision, "--name", agentID)
	if err != nil {
		return "", fmt.Errorf("create workspace: %w", err)
	}

	return agentDir, nil
}

// CleanupWorkspace removes a single agent workspace
func CleanupWorkspace(client *jj.Client, repoRoot, agentID string) error {
	agentDir := filepath.Join(repoRoot, WorkspacesDir, agentID)

	// Check for uncommitted changes
	hasChanges, err := workspaceHasChanges(client, agentDir)
	if err != nil {
		return fmt.Errorf("check workspace status: %w", err)
	}
	if hasChanges {
		return fmt.Errorf("workspace %s has uncommitted changes", agentID)
	}

	// Forget workspace in jj
	_ = client.Run("workspace", "forget", agentID)

	// Remove directory
	if err := os.RemoveAll(agentDir); err != nil {
		return fmt.Errorf("remove workspace dir: %w", err)
	}

	return nil
}

// CleanupAllWorkspaces removes all agent workspaces
func CleanupAllWorkspaces(client *jj.Client, repoRoot string) error {
	wsDir := filepath.Join(repoRoot, WorkspacesDir)
	entries, err := os.ReadDir(wsDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read workspaces dir: %w", err)
	}

	var errors []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if err := CleanupWorkspace(client, repoRoot, entry.Name()); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", entry.Name(), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(errors, "; "))
	}

	// Remove workspaces dir if empty
	remaining, _ := os.ReadDir(wsDir)
	if len(remaining) == 0 {
		_ = os.Remove(wsDir)
	}

	return nil
}

// ListWorkspaces returns active agent workspaces
func ListWorkspaces(repoRoot string) ([]string, error) {
	wsDir := filepath.Join(repoRoot, WorkspacesDir)
	entries, err := os.ReadDir(wsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var workspaces []string
	for _, entry := range entries {
		if entry.IsDir() {
			workspaces = append(workspaces, entry.Name())
		}
	}
	return workspaces, nil
}

func workspaceHasChanges(client *jj.Client, wsDir string) (bool, error) {
	out, err := client.Query("-R", wsDir, "diff", "--stat")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}
