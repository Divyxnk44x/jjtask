package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	createDraft bool
	createChain bool
)

var createCmd = &cobra.Command{
	Use:   "create [parent] <title> [description]",
	Short: "Create a new task revision",
	Long: `Create a new task revision as direct child of @ (or specified parent).

By default, creates a direct child of @. Use --chain to auto-chain from
the deepest pending descendant instead.

Examples:
  jjtask create "Fix bug"                      # direct child of @
  jjtask create xyz "Fix bug"                  # direct child of xyz
  jjtask create @ "Fix bug" "Description"      # explicit @ with description
  jjtask create --chain "Next step"            # chain from deepest pending
  jjtask create --draft "Future work"          # draft task`,
	Args: cobra.RangeArgs(1, 3),
	RunE: runCreate,
}

func init() {
	createCmd.Flags().BoolVar(&createDraft, "draft", false, "Create with [task:draft] flag")
	createCmd.Flags().BoolVar(&createChain, "chain", false, "Auto-chain from deepest pending descendant")
	rootCmd.AddCommand(createCmd)
	createCmd.ValidArgsFunction = completeRevision
}

func runCreate(cmd *cobra.Command, args []string) error {
	var title, desc, parent string

	// Parse args: [parent] <title> [description]
	// Heuristic: if first arg looks like a revset (short alphanumeric, @, or contains revision chars),
	// treat it as parent. Otherwise it's the title.
	switch len(args) {
	case 1:
		title = args[0]
	case 2:
		if looksLikeRevset(args[0]) {
			parent = args[0]
			title = args[1]
		} else {
			title = args[0]
			desc = args[1]
		}
	case 3:
		parent = args[0]
		title = args[1]
		desc = args[2]
	}

	if parent == "" {
		parent = "@"
	}

	// Check if @ is a WIP task when using explicit parent (not @)
	if parent != "@" {
		checkWipSuggestion(cmd)
	}

	// Auto-chain: find deepest pending descendant (only with --chain flag)
	if createChain {
		leaf := findDeepestPendingDescendant(parent)
		if leaf != "" {
			parent = leaf
		}
	}

	flag := "todo"
	if createDraft {
		flag = "draft"
	}

	message := fmt.Sprintf("[task:%s] %s", flag, title)
	if desc != "" {
		message = message + "\n\n" + desc
	}

	err := client.Run("new", "--no-edit", parent, "-m", message)
	if err != nil {
		return err
	}

	// Get the created revision's change ID
	out, err := client.Query("log", "-r", "children("+parent+") & description(substring:\"[task:\") & heads(all())", "--no-graph", "-T", "change_id.shortest()", "--limit", "1")
	if err != nil {
		fmt.Printf("Created task [task:%s] %s (could not resolve ID: %v)\n", flag, title, err)
		return nil
	}
	changeID := strings.TrimSpace(out)
	if changeID == "" {
		fmt.Printf("Created task [task:%s] %s\n", flag, title)
	} else {
		fmt.Printf("Created new commit %s (empty) [task:%s] %s\n", changeID, flag, title)
	}

	return nil
}

// looksLikeRevset returns true if s looks like a jj revision specifier rather than a task title
func looksLikeRevset(s string) bool {
	if s == "" {
		return false
	}

	// @ or @- or @-N
	if s[0] == '@' {
		return true
	}

	// Revset operators: ::x, x::, x::y, x+, x-, x..y, etc.
	for _, op := range []string{"::", "..", "~", "&", "|", "+", "-"} {
		if strings.Contains(s, op) {
			return true
		}
	}

	// Function calls: root(), ancestors(x), mine(), etc.
	if strings.Contains(s, "(") && strings.Contains(s, ")") {
		return true
	}

	// Short alphanumeric strings (1-12 chars, lowercase letters and numbers only) are likely change IDs
	if len(s) >= 1 && len(s) <= 12 {
		for _, c := range s {
			if (c < 'a' || c > 'z') && (c < '0' || c > '9') {
				return false
			}
		}
		return true
	}

	return false
}

// findDeepestPendingDescendant finds the deepest pending task descendant of rev
func findDeepestPendingDescendant(rev string) string {
	// Get all pending descendants, sorted by depth (most ancestors = deepest)
	// We want the leaf of the chain - a task with no pending children
	out, err := client.Query("log",
		"-r", fmt.Sprintf("(%s | descendants(%s)) & tasks_pending()", rev, rev),
		"--no-graph",
		"-T", `change_id.shortest() ++ "\n"`,
	)
	if err != nil {
		return ""
	}

	candidates := strings.Split(strings.TrimSpace(out), "\n")
	if len(candidates) == 0 || (len(candidates) == 1 && candidates[0] == "") {
		return ""
	}

	// Find the leaf - a candidate with no pending children
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}

		// Check if this candidate has any pending children
		children, err := client.Query("log",
			"-r", fmt.Sprintf("children(%s) & tasks_pending()", candidate),
			"--no-graph",
			"-T", "change_id.shortest()",
			"--limit", "1",
		)
		if err != nil {
			continue
		}

		if strings.TrimSpace(children) == "" {
			// No pending children - this is the leaf
			return candidate
		}
	}

	// No leaf found, return last candidate
	return strings.TrimSpace(candidates[len(candidates)-1])
}

// checkWipSuggestion suggests chaining to @ if @ is a WIP task
func checkWipSuggestion(cmd *cobra.Command) {
	atDesc, err := client.GetDescription("@")
	if err != nil {
		return
	}
	if !strings.HasPrefix(atDesc, "[task:wip]") {
		return
	}

	atID, err := client.Query("log", "-r", "@", "--no-graph", "-T", "change_id.shortest()")
	if err != nil {
		return
	}
	atID = strings.TrimSpace(atID)

	stderr := cmd.ErrOrStderr()
	_, _ = fmt.Fprintln(stderr)
	_, _ = fmt.Fprintf(stderr, "Note: Current revision (%s) is a WIP task.\n", atID)
	_, _ = fmt.Fprintln(stderr, "Consider: `jjtask create \"title\"` to auto-chain from @")
	_, _ = fmt.Fprintln(stderr)
}
