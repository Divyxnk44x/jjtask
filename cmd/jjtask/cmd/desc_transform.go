package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var (
	descTransformRev   string
	descTransformStdin bool
)

var descTransformCmd = &cobra.Command{
	Use:   "desc-transform [REV] <sed-expr|command...>",
	Short: "Transform revision description",
	Long: `Transform a revision description through a command or sed expression.

Sed expressions (s/pattern/replacement/) are handled natively in Go,
supporting multiline patterns and replacements with \n for newlines.

Use --stdin to read new description content directly from stdin,
bypassing sed/command execution entirely.

Examples:
  jjtask desc-transform 's/foo/bar/'
  jjtask desc-transform xqq 's/foo/bar/'             # rev as first arg
  jjtask desc-transform 's/foo/bar\nline2/'          # multiline replacement
  jjtask desc-transform 's/old/new/g'                # global replace
  jjtask desc-transform sed 's/foo/bar/' --rev mxyz  # explicit sed
  jjtask desc-transform awk '/^##/{print}'           # external command
  echo "new content" | jjtask desc-transform --stdin # from stdin`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		rev := descTransformRev
		cmdArgs := args

		// If first arg doesn't look like a sed expr or known command, treat as rev
		if len(args) > 1 && !isSedExpr(args[0]) && !isKnownTransformCmd(args[0]) {
			rev = args[0]
			cmdArgs = args[1:]
		}

		// Get current description
		desc, err := client.GetDescription(rev)
		if err != nil {
			return fmt.Errorf("failed to get description: %w", err)
		}

		var newDesc string

		if descTransformStdin {
			// Read replacement directly from stdin
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}
			newDesc = string(data)
		} else {
			if len(cmdArgs) == 0 {
				return fmt.Errorf("requires sed expression or command")
			}

			// If single argument starting with s/ (or s + delimiter), use native Go regex
			if len(cmdArgs) == 1 && isSedExpr(cmdArgs[0]) {
				result, err := applySedExpr(desc, cmdArgs[0])
				if err != nil {
					return err
				}
				newDesc = result
			} else {
				// External command
				result, err := runExternalTransform(desc, cmdArgs)
				if err != nil {
					return err
				}
				newDesc = result
			}
		}

		if newDesc == desc {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Nothing changed.")
			return nil
		}

		return client.SetDescription(rev, newDesc)
	},
}

// isSedExpr checks if a string looks like a sed s/pattern/replacement/ expression
func isSedExpr(s string) bool {
	if len(s) < 4 || s[0] != 's' {
		return false
	}
	// s followed by a delimiter (any non-alphanumeric char)
	delim := s[1]
	isAlpha := (delim >= 'a' && delim <= 'z') || (delim >= 'A' && delim <= 'Z')
	isDigit := delim >= '0' && delim <= '9'
	return !isAlpha && !isDigit
}

// isKnownTransformCmd checks if arg is a known transform command
func isKnownTransformCmd(s string) bool {
	known := []string{"sed", "awk", "perl", "ruby", "python", "python3"}
	for _, cmd := range known {
		if s == cmd {
			return true
		}
	}
	return false
}

// applySedExpr applies a sed s/pattern/replacement/[flags] expression using Go regex
func applySedExpr(input, expr string) (string, error) {
	// Parse s<delim>pattern<delim>replacement<delim>[flags]
	if len(expr) < 4 || expr[0] != 's' {
		return "", fmt.Errorf("invalid sed expression: must start with s<delimiter>")
	}

	// Find delimiter (char after 's')
	delim := expr[1]
	parts := splitSedExpr(expr[2:], delim)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid sed expression: missing replacement")
	}

	pattern := parts[0]
	replacement := parts[1]
	flags := ""
	if len(parts) > 2 {
		flags = parts[2]
	}

	// Handle escape sequences in replacement
	replacement = strings.ReplaceAll(replacement, `\n`, "\n")
	replacement = strings.ReplaceAll(replacement, `\t`, "\t")

	// Compile regex
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid pattern: %w", err)
	}

	// Apply replacement
	if strings.Contains(flags, "g") {
		return re.ReplaceAllString(input, replacement), nil
	}
	// Replace first occurrence only
	loc := re.FindStringIndex(input)
	if loc == nil {
		return input, nil
	}
	match := input[loc[0]:loc[1]]
	replaced := re.ReplaceAllString(match, replacement)
	return input[:loc[0]] + replaced + input[loc[1]:], nil
}

// splitSedExpr splits a sed expression by delimiter, respecting escapes
func splitSedExpr(s string, delim byte) []string {
	var parts []string
	var current strings.Builder
	escaped := false

	for i := range len(s) {
		c := s[i]
		if escaped {
			current.WriteByte(c)
			escaped = false
			continue
		}
		if c == '\\' {
			current.WriteByte(c)
			escaped = true
			continue
		}
		if c == delim {
			parts = append(parts, current.String())
			current.Reset()
			continue
		}
		current.WriteByte(c)
	}
	parts = append(parts, current.String())
	return parts
}

// runExternalTransform runs an external command to transform the description
func runExternalTransform(desc string, cmdArgs []string) (string, error) {
	cmdName := cmdArgs[0]
	if _, err := exec.LookPath(cmdName); err != nil {
		return "", fmt.Errorf("command %q not found", cmdName)
	}

	transformCmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	transformCmd.Stdin = strings.NewReader(desc)
	var stdout, stderr bytes.Buffer
	transformCmd.Stdout = &stdout
	transformCmd.Stderr = &stderr

	if err := transformCmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("transform failed: %s", stderr.String())
		}
		return "", fmt.Errorf("transform failed: %w", err)
	}

	return stdout.String(), nil
}

func init() {
	rootCmd.AddCommand(descTransformCmd)
	descTransformCmd.Flags().StringVarP(&descTransformRev, "rev", "r", "@", "revision to transform")
	descTransformCmd.Flags().BoolVar(&descTransformStdin, "stdin", false, "read new description from stdin")
	_ = descTransformCmd.RegisterFlagCompletionFunc("rev", completeRevision)
}
