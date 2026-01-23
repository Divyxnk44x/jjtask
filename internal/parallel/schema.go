package parallel

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Session holds parallel agent session info parsed from task description
type Session struct {
	Mode    string    // "shared" or "workspace"
	Started time.Time // session start time
	Agents  []Agent
}

// Agent represents a single agent's assignment
type Agent struct {
	ID          string // e.g. "agent-a"
	FilePattern string // glob pattern like "src/api/**"
	Description string // what this agent is doing
	TaskID      string // change ID for workspace mode, empty for shared
}

var agentLineRe = regexp.MustCompile(`^- ([\w-]+):\s*([^|]+?)\s*\|\s*([^|]+?)\s*(?:\|\s*task:(\w+))?$`)

// ParseSession extracts parallel session info from task description
func ParseSession(description string) (*Session, error) {
	lines := strings.Split(description, "\n")

	var inParallelSection, inAgentsSection bool
	session := &Session{}
	var parseErrors []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect section headers
		if strings.HasPrefix(trimmed, "## Parallel Session") {
			inParallelSection = true
			inAgentsSection = false
			continue
		}
		if strings.HasPrefix(trimmed, "### Agents") {
			inAgentsSection = true
			continue
		}
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
			if inParallelSection && strings.HasPrefix(trimmed, "## ") {
				inParallelSection = false
			}
			if inAgentsSection && (strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ")) {
				inAgentsSection = false
			}
			continue
		}

		if !inParallelSection {
			continue
		}

		// Parse key: value lines in Parallel Session
		if !inAgentsSection {
			if strings.HasPrefix(trimmed, "mode:") {
				session.Mode = strings.TrimSpace(strings.TrimPrefix(trimmed, "mode:"))
			} else if strings.HasPrefix(trimmed, "started:") {
				ts := strings.TrimSpace(strings.TrimPrefix(trimmed, "started:"))
				t, err := time.Parse(time.RFC3339, ts)
				if err == nil {
					session.Started = t
				}
			}
		}

		// Parse agent lines
		if inAgentsSection && strings.HasPrefix(trimmed, "- ") {
			match := agentLineRe.FindStringSubmatch(trimmed)
			if match != nil {
				agent := Agent{
					ID:          match[1],
					FilePattern: strings.TrimSpace(match[2]),
					Description: strings.TrimSpace(match[3]),
				}
				if len(match) > 4 && match[4] != "" {
					agent.TaskID = match[4]
				}
				session.Agents = append(session.Agents, agent)
			} else {
				parseErrors = append(parseErrors, fmt.Sprintf("invalid agent line: %s", trimmed))
			}
		}
	}

	if session.Mode == "" && len(session.Agents) == 0 {
		return nil, nil // no parallel session found
	}

	if len(parseErrors) > 0 {
		return session, fmt.Errorf("parse warnings: %s", strings.Join(parseErrors, "; "))
	}

	return session, nil
}

// FormatSession generates markdown for a parallel session
func FormatSession(session *Session) string {
	var b strings.Builder

	b.WriteString("## Parallel Session\n")
	b.WriteString(fmt.Sprintf("mode: %s\n", session.Mode))
	if !session.Started.IsZero() {
		b.WriteString(fmt.Sprintf("started: %s\n", session.Started.Format(time.RFC3339)))
	}
	b.WriteString("\n### Agents\n")

	for _, agent := range session.Agents {
		if agent.TaskID != "" {
			b.WriteString(fmt.Sprintf("- %s: %s | %s | task:%s\n", agent.ID, agent.FilePattern, agent.Description, agent.TaskID))
		} else {
			b.WriteString(fmt.Sprintf("- %s: %s | %s\n", agent.ID, agent.FilePattern, agent.Description))
		}
	}

	return b.String()
}

// UpdateDescription inserts or replaces parallel session in description
func UpdateDescription(description string, session *Session) string {
	sessionText := FormatSession(session)

	// Find existing parallel session section
	lines := strings.Split(description, "\n")
	var result []string
	var inParallelSection bool
	var inserted bool

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "## Parallel Session") {
			inParallelSection = true
			continue
		}

		if inParallelSection {
			// Look for next ## section to end parallel section
			if strings.HasPrefix(trimmed, "## ") {
				inParallelSection = false
				if !inserted {
					result = append(result, sessionText)
					inserted = true
				}
				result = append(result, line)
			}
			continue
		}

		// Insert before ## Requirements or ## Acceptance if we haven't yet
		if !inserted && (strings.HasPrefix(trimmed, "## Requirements") || strings.HasPrefix(trimmed, "## Acceptance")) {
			result = append(result, sessionText)
			result = append(result, "")
			inserted = true
		}

		result = append(result, line)

		// If at end and haven't inserted, do it now
		if i == len(lines)-1 && !inserted {
			result = append(result, "")
			result = append(result, sessionText)
			inserted = true
		}
	}

	return strings.Join(result, "\n")
}

// GetAgentByID finds an agent by ID
func (s *Session) GetAgentByID(id string) *Agent {
	for i := range s.Agents {
		if s.Agents[i].ID == id {
			return &s.Agents[i]
		}
	}
	return nil
}

// OtherAgents returns all agents except the given one
func (s *Session) OtherAgents(excludeID string) []Agent {
	var others []Agent
	for _, a := range s.Agents {
		if a.ID != excludeID {
			others = append(others, a)
		}
	}
	return others
}
