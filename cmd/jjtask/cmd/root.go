package cmd

import (
	"github.com/spf13/cobra"

	"jjtask/internal/jj"
)

var (
	client  *jj.Client
	globals jj.GlobalFlags
)

var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "jjtask",
	Short:   "Task management for jj repositories",
	Long:    "jjtask provides structured task management using jj revisions with [task:*] flags.",
	Version: Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		client = jj.NewWithGlobals(globals)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global jj flags
	rootCmd.PersistentFlags().StringVarP(&globals.Repository, "repository", "R", "", "Path to repository")
	rootCmd.PersistentFlags().StringVar(&globals.AtOperation, "at-operation", "", "Operation to load repo at")
	rootCmd.PersistentFlags().StringVar(&globals.Color, "color", "", "When to colorize output")
	rootCmd.PersistentFlags().StringArrayVar(&globals.Config, "config", nil, "Additional config value")
	rootCmd.PersistentFlags().StringVar(&globals.ConfigFile, "config-file", "", "Additional config file")
	rootCmd.PersistentFlags().BoolVar(&globals.IgnoreWorkingCopy, "ignore-working-copy", false, "Don't snapshot working copy")
	rootCmd.PersistentFlags().BoolVar(&globals.IgnoreImmutable, "ignore-immutable", false, "Allow rewriting immutable commits")
	rootCmd.PersistentFlags().BoolVar(&globals.Debug, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&globals.Quiet, "quiet", false, "Silence non-primary output")
	rootCmd.PersistentFlags().BoolVar(&globals.NoPager, "no-pager", false, "Disable pager")

	// Silence usage on error
	rootCmd.SilenceUsage = true

	// Disable default completion command (we provide our own with better docs)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}
