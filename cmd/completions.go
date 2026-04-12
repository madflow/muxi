package cmd

import (
	"github.com/spf13/cobra"

	"muxi/internal/discovery"
)

func newCompletionsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "completions [command]",
		Short: "Print completion candidates",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "start", "stop", "edit", "open", "copy", "delete":
				names, err := discovery.ListProjectNames()
				if err != nil {
					return err
				}
				printLines(cmd, names)
			}
			return nil
		},
	}
}
