package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"jjtask/internal/workspace"
)

var (
	findFormat string
	findStatus string
)

type TaskItem struct {
	ChangeID    string `json:"change_id"`
	Flag        string `json:"flag"`
	Title       string `json:"title"`
	Empty       bool   `json:"empty"`
	WorkingCopy bool   `json:"working_copy"`
	Repo        string `json:"repo,omitempty"`
}

type FindOutput struct {
	Tasks []TaskItem `json:"tasks"`
	Count int        `json:"count"`
}

var findRevset string

var findCmd = &cobra.Command{
	Use:   "find [--status STATUS] [--revisions REVSET]",
	Short: "List tasks",
	Long: `List tasks matching a status filter or custom revset.

Without arguments, shows pending tasks. With --status, shows tasks
matching that status. Use -r for custom revsets.

Status options: pending, todo, wip, done, blocked, standby, untested, draft, review, all

Examples:
  jjtask find                    # pending tasks (default)
  jjtask find --status todo      # todo tasks only
  jjtask find -s wip             # work in progress
  jjtask find -s done            # completed tasks
  jjtask find -s all             # all tasks including done
  jjtask find -r 'tasks() & mine()'`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var revset string
		customRevset := findRevset != ""

		if customRevset {
			// Custom revset via -r flag - intersect with tasks() to only show task revisions
			revset = fmt.Sprintf("(%s) & tasks()", findRevset)
		} else {
			taskRevset := "tasks_pending()"
			status := findStatus
			if status != "" {
				switch status {
				case "pending":
					taskRevset = "tasks_pending()"
				case "todo":
					taskRevset = "tasks_todo()"
				case "wip":
					taskRevset = "tasks_wip()"
				case "done":
					taskRevset = "tasks_done()"
				case "blocked":
					taskRevset = "tasks_blocked()"
				case "standby":
					taskRevset = "tasks_standby()"
				case "untested":
					taskRevset = "tasks_untested()"
				case "draft":
					taskRevset = "tasks_draft()"
				case "review":
					taskRevset = "tasks_review()"
				case "all":
					taskRevset = "tasks()"
				default:
					return fmt.Errorf("unknown status %q", status)
				}
			}
			// Show tasks + @ (jj handles elision for gaps in history)
			if status == "done" || status == "all" {
				revset = taskRevset
			} else {
				revset = fmt.Sprintf("%s | @", taskRevset)
			}
		}

		repos, workspaceRoot, err := workspace.GetRepos()
		if err != nil {
			return err
		}

		isMulti := len(repos) > 1

		if findFormat == "json" {
			return findJSON(repos, workspaceRoot, revset, isMulti)
		}

		// Text output
		if hint := workspace.ContextHint(); hint != "" {
			fmt.Println(hint)
			fmt.Println()
		}

		PrintTasksWithRevset(repos, workspaceRoot, revset)
		return nil
	},
}

// PrintTasksWithRevset outputs tasks matching revset across repos
func PrintTasksWithRevset(repos []workspace.Repo, workspaceRoot, revset string) {
	isMulti := len(repos) > 1

	for _, repo := range repos {
		repoPath := workspace.ResolveRepoPath(repo, workspaceRoot)

		if isMulti {
			displayPath := workspace.RelativePath(repoPath)
			fmt.Printf("=== %s: jj -R %s log ===\n", workspace.DisplayName(repo), displayPath)
		}

		jjArgs := []string{}
		if globals.Color != "" {
			jjArgs = append(jjArgs, "--color", globals.Color)
		} else if client.IsTTY {
			jjArgs = append(jjArgs, "--color=always")
		}
		jjArgs = append(jjArgs, "-R", repoPath, "log", "-r", revset, "-T", "task_log")

		jjCmd := exec.Command("jj", jjArgs...)
		jjCmd.Env = append(os.Environ(), "JJ_ALLOW_TASK=1", "JJ_NO_HINTS=1")

		output, err := jjCmd.Output()
		if err != nil {
			if isMulti {
				if client.IsTTY {
					fmt.Println("~  \033[32m(no tasks)\033[0m")
				} else {
					fmt.Println("~  (no tasks)")
				}
			}
		} else {
			outStr := strings.TrimRight(string(output), "\n")
			if outStr != "" {
				fmt.Println(outStr)
			} else if isMulti {
				if client.IsTTY {
					fmt.Println("~  \033[32m(no tasks)\033[0m")
				} else {
					fmt.Println("~  (no tasks)")
				}
			}
		}

		if isMulti {
			fmt.Println()
		}
	}
}

func init() {
	findCmd.Flags().StringVarP(&findStatus, "status", "s", "", "Filter by status (pending, todo, wip, done, blocked, standby, untested, draft, review, all)")
	findCmd.Flags().StringVarP(&findRevset, "revisions", "r", "", "Custom revset to filter tasks")
	findCmd.Flags().StringVar(&findFormat, "format", "text", "Output format: text or json")
	rootCmd.AddCommand(findCmd)
	_ = findCmd.RegisterFlagCompletionFunc("status", completeFindFlag)
}

func findJSON(repos []workspace.Repo, workspaceRoot, revset string, isMulti bool) error {
	var output FindOutput
	taskFlagRe := regexp.MustCompile(`\[task:(\w+)\]`)

	// Template: change_id \t empty \t working_copy \t description_first_line
	tmpl := `change_id.shortest() ++ "\t" ++ if(empty, "true", "false") ++ "\t" ++ if(self.contained_in("@"), "true", "false") ++ "\t" ++ description.first_line() ++ "\n"`

	for _, repo := range repos {
		repoPath := workspace.ResolveRepoPath(repo, workspaceRoot)

		jjArgs := []string{"--color=never", "-R", repoPath, "log", "-r", revset, "--no-graph", "-T", tmpl}
		jjCmd := exec.Command("jj", jjArgs...)
		jjCmd.Env = append(os.Environ(), "JJ_ALLOW_TASK=1", "JJ_NO_HINTS=1")

		out, err := jjCmd.Output()
		if err != nil {
			continue
		}

		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "\t", 4)
			if len(parts) < 4 {
				continue
			}

			item := TaskItem{
				ChangeID:    parts[0],
				Empty:       parts[1] == "true",
				WorkingCopy: parts[2] == "true",
			}

			firstLine := parts[3]
			if match := taskFlagRe.FindStringSubmatch(firstLine); match != nil {
				item.Flag = match[1]
				item.Title = strings.TrimSpace(taskFlagRe.ReplaceAllString(firstLine, ""))
			} else {
				item.Title = firstLine
			}

			if isMulti {
				item.Repo = workspace.DisplayName(repo)
			}

			output.Tasks = append(output.Tasks, item)
		}
	}

	output.Count = len(output.Tasks)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}
