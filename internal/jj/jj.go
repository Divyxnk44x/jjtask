package jj

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/term"
)

// GlobalFlags holds jj global flags parsed from command line
type GlobalFlags struct {
	Repository        string
	AtOperation       string
	Color             string
	Config            []string
	ConfigFile        string
	IgnoreWorkingCopy bool
	IgnoreImmutable   bool
	Debug             bool
	Quiet             bool
	NoPager           bool
}

// Client wraps jj subprocess calls
type Client struct {
	Globals GlobalFlags
	IsTTY   bool
}

// New creates a jj client with default settings
func New() *Client {
	return &Client{
		IsTTY: isTerminal(),
	}
}

// NewWithGlobals creates a jj client with parsed global flags
func NewWithGlobals(globals GlobalFlags) *Client {
	return &Client{
		Globals: globals,
		IsTTY:   isTerminal(),
	}
}

func isTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// buildQueryArgs constructs args for internal queries (no color)
func (c *Client) buildQueryArgs(args []string) []string {
	var result []string

	if c.Globals.Repository != "" {
		result = append(result, "-R", c.Globals.Repository)
	}
	if c.Globals.AtOperation != "" {
		result = append(result, "--at-operation", c.Globals.AtOperation)
	}
	for _, cfg := range c.Globals.Config {
		result = append(result, "--config", cfg)
	}
	if c.Globals.ConfigFile != "" {
		result = append(result, "--config-file", c.Globals.ConfigFile)
	}
	if c.Globals.Quiet {
		result = append(result, "--quiet")
	}
	if c.Globals.NoPager {
		result = append(result, "--no-pager")
	}

	result = append(result, args...)
	return result
}

// buildArgs constructs jj command args with global flags
func (c *Client) buildArgs(args []string) []string {
	var result []string

	if c.Globals.Repository != "" {
		result = append(result, "-R", c.Globals.Repository)
	}
	if c.Globals.AtOperation != "" {
		result = append(result, "--at-operation", c.Globals.AtOperation)
	}
	if c.Globals.Color != "" {
		result = append(result, "--color", c.Globals.Color)
	} else if c.IsTTY {
		result = append(result, "--color=always")
	}
	for _, cfg := range c.Globals.Config {
		result = append(result, "--config", cfg)
	}
	if c.Globals.ConfigFile != "" {
		result = append(result, "--config-file", c.Globals.ConfigFile)
	}
	if c.Globals.IgnoreWorkingCopy {
		result = append(result, "--ignore-working-copy")
	}
	if c.Globals.IgnoreImmutable {
		result = append(result, "--ignore-immutable")
	}
	if c.Globals.Debug {
		result = append(result, "--debug")
	}
	if c.Globals.Quiet {
		result = append(result, "--quiet")
	}
	if c.Globals.NoPager {
		result = append(result, "--no-pager")
	}

	return append(result, args...)
}

// Run executes jj with given args, inheriting stdin/stdout/stderr
func (c *Client) Run(args ...string) error {
	cmd := exec.Command("jj", c.buildArgs(args)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "JJ_ALLOW_TASK=1", "JJ_NO_HINTS=1")
	return cmd.Run()
}

// Query executes jj for internal queries (with --ignore-working-copy, --color=never)
func (c *Client) Query(args ...string) (string, error) {
	queryArgs := append([]string{"--ignore-working-copy", "--color=never"}, args...)
	cmd := exec.Command("jj", c.buildQueryArgs(queryArgs)...)
	cmd.Env = append(os.Environ(), "JJ_ALLOW_TASK=1", "JJ_NO_HINTS=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return "", err
	}
	return stdout.String(), nil
}

// Pipe executes jj with stdin from input string
func (c *Client) Pipe(input string, args ...string) error {
	cmd := exec.Command("jj", c.buildArgs(args)...)
	cmd.Stdin = strings.NewReader(input)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "JJ_ALLOW_TASK=1", "JJ_NO_HINTS=1")
	return cmd.Run()
}

// PipeQuiet executes jj with stdin, capturing output
func (c *Client) PipeQuiet(input string, args ...string) (string, error) {
	cmd := exec.Command("jj", c.buildArgs(args)...)
	cmd.Stdin = strings.NewReader(input)
	cmd.Env = append(os.Environ(), "JJ_ALLOW_TASK=1", "JJ_NO_HINTS=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return "", err
	}
	return stdout.String(), nil
}

// Root returns the repository root directory
func (c *Client) Root() (string, error) {
	// Prefer JJ_WORKSPACE_ROOT from jj util exec
	if root := os.Getenv("JJ_WORKSPACE_ROOT"); root != "" {
		return root, nil
	}
	out, err := c.Query("root")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// GetDescription returns the description for a revision
func (c *Client) GetDescription(rev string) (string, error) {
	out, err := c.Query("log", "-r", rev, "-n1", "--no-graph", "-T", "description")
	if err != nil {
		return "", err
	}
	return out, nil
}

// SetDescription sets the description for a revision
func (c *Client) SetDescription(rev, desc string) error {
	return c.Pipe(desc, "describe", "-r", rev, "--stdin")
}

// IsValidRev checks if a revision is valid
func (c *Client) IsValidRev(rev string) bool {
	_, err := c.Query("log", "-r", rev, "--no-graph", "-T", "change_id", "--limit", "1")
	return err == nil
}

// FindConfigDir locates jjtask config directory
func FindConfigDir() string {
	// Find the jjtask installation directory
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	// Resolve symlinks
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return ""
	}
	// Go up from bin/ to root, then config/conf.d
	root := filepath.Dir(filepath.Dir(exe))
	confDir := filepath.Join(root, "config", "conf.d")
	if _, err := os.Stat(confDir); err == nil {
		return confDir
	}
	return ""
}

// SetupEnv configures environment for jjtask
func SetupEnv() {
	_ = os.Setenv("JJ_ALLOW_TASK", "1")
	_ = os.Setenv("JJ_NO_HINTS", "1")

	// Auto-set JJ_CONFIG for agent mode (non-TTY)
	if os.Getenv("JJ_CONFIG") == "" && !isTerminal() {
		if confDir := FindConfigDir(); confDir != "" {
			_ = os.Setenv("JJ_CONFIG", confDir)
		}
	}
}

// GetActiveRevisions returns change IDs of WIP tasks only
func (c *Client) GetActiveRevisions() ([]string, error) {
	out, err := c.Query("log", "-r", "tasks_wip()", "--no-graph", "-T", `change_id.shortest() ++ "\n"`)
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// isRootChangeID checks if a change ID is the root commit (all z's)
func isRootChangeID(id string) bool {
	for _, c := range id {
		if c != 'z' {
			return false
		}
	}
	return true
}

// GetParents returns the parent change IDs of a revision
func (c *Client) GetParents(rev string) ([]string, error) {
	out, err := c.Query("log", "-r", "parents("+rev+")", "--no-graph", "-T", `change_id.shortest() ++ "\n"`)
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// RemoveFromMerge removes a revision from @'s parents, preserving @ content
func (c *Client) RemoveFromMerge(task string) error {
	parents, err := c.GetParents("@")
	if err != nil {
		return fmt.Errorf("getting parents: %w", err)
	}

	// Filter out the task and root
	var remaining []string
	for _, p := range parents {
		if p != task && !isRootChangeID(p) {
			remaining = append(remaining, p)
		}
	}

	if len(remaining) == 0 {
		return nil
	}

	if len(remaining) == 1 {
		if err := c.Run("squash", "--into", remaining[0], "--keep-emptied"); err != nil {
			return fmt.Errorf("squashing into parent: %w", err)
		}
		return c.Run("edit", remaining[0])
	}

	// Multiple parents - rebase @ onto remaining parents
	args := []string{"rebase", "-r", "@"}
	for _, p := range remaining {
		args = append(args, "-o", p)
	}
	return c.Run(args...)
}

// IsAncestorOf checks if rev is an ancestor of target
func (c *Client) IsAncestorOf(rev, target string) (bool, error) {
	out, err := c.Query("log", "-r", rev+"::"+target, "--no-graph", "-T", "change_id.shortest()", "--limit", "1")
	if err != nil {
		// Empty result means not an ancestor
		return false, nil
	}
	return strings.TrimSpace(out) != "", nil
}

// AddToMerge adds a revision as a new parent of @, preserving @ content
func (c *Client) AddToMerge(task string) error {
	return c.AddMultipleToMerge([]string{task})
}

// AddMultipleToMerge adds multiple revisions as new parents of @ in a single rebase
func (c *Client) AddMultipleToMerge(tasks []string) error {
	if len(tasks) == 0 {
		return nil
	}

	atID, err := c.Query("log", "-r", "@", "--no-graph", "-T", "change_id.shortest()")
	if err != nil {
		return fmt.Errorf("getting @ ID: %w", err)
	}
	atID = strings.TrimSpace(atID)

	parents, err := c.GetParents("@")
	if err != nil {
		return fmt.Errorf("getting parents: %w", err)
	}

	// Build new parent list: existing parents + new tasks (excluding @ itself, root, and duplicates)
	seen := make(map[string]bool)
	var newParents []string
	for _, p := range parents {
		if !isRootChangeID(p) && !seen[p] {
			seen[p] = true
			newParents = append(newParents, p)
		}
	}
	for _, task := range tasks {
		if task != atID && !seen[task] {
			seen[task] = true
			newParents = append(newParents, task)
		}
	}

	// If nothing changed, skip
	if len(newParents) == len(parents) {
		allPresent := true
		for _, t := range tasks {
			if t != atID && !slices.Contains(parents, t) {
				allPresent = false
				break
			}
		}
		if allPresent {
			return nil
		}
	}

	if len(newParents) == 0 {
		return nil
	}

	// Single rebase with all parents
	args := []string{"rebase", "-r", "@"}
	for _, p := range newParents {
		args = append(args, "-o", p)
	}
	return c.Run(args...)
}
