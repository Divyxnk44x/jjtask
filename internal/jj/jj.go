package jj

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
