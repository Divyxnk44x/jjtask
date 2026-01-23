package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var showDescFormat string

type ShowDescOutput struct {
	Revision    string `json:"revision"`
	ChangeID    string `json:"change_id"`
	Description string `json:"description"`
	FirstLine   string `json:"first_line"`
	TaskFlag    string `json:"task_flag,omitempty"`
}

var showDescCmd = &cobra.Command{
	Use:   "show-desc [rev]",
	Short: "Print revision description",
	Long: `Print the description for a revision (default @).

Examples:
  jjtask show-desc
  jjtask show-desc mxyz`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rev := "@"
		if len(args) == 1 {
			rev = args[0]
		}

		desc, err := client.GetDescription(rev)
		if err != nil {
			return err
		}

		if showDescFormat == "json" {
			changeID, _ := client.Query("log", "-r", rev, "--no-graph", "-T", "change_id.shortest()")
			changeID = strings.TrimSpace(changeID)

			lines := strings.SplitN(desc, "\n", 2)
			firstLine := ""
			if len(lines) > 0 {
				firstLine = lines[0]
			}

			var taskFlag string
			if re := regexp.MustCompile(`\[task:(\w+)\]`); re.MatchString(firstLine) {
				if match := re.FindStringSubmatch(firstLine); match != nil {
					taskFlag = match[1]
				}
			}

			output := ShowDescOutput{
				Revision:    rev,
				ChangeID:    changeID,
				Description: desc,
				FirstLine:   firstLine,
				TaskFlag:    taskFlag,
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(output)
		}

		fmt.Print(desc)
		return nil
	},
}

func init() {
	showDescCmd.Flags().StringVar(&showDescFormat, "format", "text", "Output format: text or json")
	rootCmd.AddCommand(showDescCmd)
	showDescCmd.ValidArgsFunction = completeRevision
}
