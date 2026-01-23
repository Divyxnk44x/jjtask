package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print jjtask version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("jjtask version %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
