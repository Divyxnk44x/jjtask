package parallel

import (
	"fmt"
	"path/filepath"
	"strings"

	"jjtask/internal/jj"
)

// Conflict represents a file modified by multiple agents
type Conflict struct {
	File   string
	Agents []string
}

// GetModifiedFiles returns files modified in a revision
func GetModifiedFiles(client *jj.Client, rev string) ([]string, error) {
	out, err := client.Query("diff", "-r", rev, "--name-only")
	if err != nil {
		return nil, err
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// FindFileConflicts detects files modified by multiple agents
func FindFileConflicts(client *jj.Client, session *Session) ([]Conflict, error) {
	if session.Mode != "workspace" {
		return nil, nil // Can only check workspace mode
	}

	// Map of file -> agents that modified it
	fileAgents := make(map[string][]string)

	var errs []error
	for _, agent := range session.Agents {
		if agent.TaskID == "" {
			continue
		}

		files, err := GetModifiedFiles(client, agent.TaskID)
		if err != nil {
			errs = append(errs, fmt.Errorf("get files for %s: %w", agent.ID, err))
			continue
		}

		for _, file := range files {
			fileAgents[file] = append(fileAgents[file], agent.ID)
		}
	}
	if len(errs) > 0 && len(fileAgents) == 0 {
		return nil, fmt.Errorf("failed to get modified files: %v", errs)
	}

	var conflicts []Conflict
	for file, agents := range fileAgents {
		if len(agents) > 1 {
			conflicts = append(conflicts, Conflict{
				File:   file,
				Agents: agents,
			})
		}
	}

	return conflicts, nil
}

// PatternsOverlap checks if two glob patterns could match the same files
func PatternsOverlap(patternA, patternB string) bool {
	if patternA == "" || patternB == "" {
		return false
	}

	// Simple heuristics for common cases
	// Full overlap detection would require glob expansion

	// Direct prefix match
	if strings.HasPrefix(patternA, patternB) || strings.HasPrefix(patternB, patternA) {
		return true
	}

	// Strip trailing ** and check prefix
	baseA := strings.TrimSuffix(patternA, "**")
	baseA = strings.TrimSuffix(baseA, "*")
	baseA = strings.TrimSuffix(baseA, "/")

	baseB := strings.TrimSuffix(patternB, "**")
	baseB = strings.TrimSuffix(baseB, "*")
	baseB = strings.TrimSuffix(baseB, "/")

	if baseA != "" && baseB != "" {
		if strings.HasPrefix(baseA, baseB) || strings.HasPrefix(baseB, baseA) {
			return true
		}
	}

	return false
}

// CheckPatternOverlaps returns warnings for overlapping agent assignments
func CheckPatternOverlaps(session *Session) []string {
	var warnings []string

	for i, agentA := range session.Agents {
		for j, agentB := range session.Agents {
			if i >= j {
				continue
			}
			if PatternsOverlap(agentA.FilePattern, agentB.FilePattern) {
				warnings = append(warnings, fmt.Sprintf(
					"%s (%s) and %s (%s) may overlap",
					agentA.ID, agentA.FilePattern,
					agentB.ID, agentB.FilePattern,
				))
			}
		}
	}

	return warnings
}

// FileMatchesPattern checks if a file path matches a glob pattern
func FileMatchesPattern(file, pattern string) bool {
	if pattern == "" {
		return false
	}

	// Handle ** patterns
	if strings.Contains(pattern, "**") {
		// Convert ** to match any path
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := parts[0]
			suffix := strings.TrimPrefix(parts[1], "/")

			if !strings.HasPrefix(file, prefix) {
				return false
			}

			if suffix == "" {
				return true
			}

			// Check suffix match
			matched, _ := filepath.Match(suffix, filepath.Base(file))
			return matched
		}
	}

	// Simple glob match
	matched, _ := filepath.Match(pattern, file)
	if matched {
		return true
	}

	// Try matching just the filename
	matched, _ = filepath.Match(pattern, filepath.Base(file))
	return matched
}

// FormatConflicts returns a formatted string of conflicts for display
func FormatConflicts(conflicts []Conflict) string {
	if len(conflicts) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Conflicts detected:\n")
	for _, c := range conflicts {
		b.WriteString(fmt.Sprintf("  %s modified by: %s\n", c.File, strings.Join(c.Agents, ", ")))
	}
	return b.String()
}
