package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"tmuxi/internal/discovery"
)

func newListCommand() *cobra.Command {
	var newline bool
	var activeOnly bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"l", "ls"},
		Short:   "List tmuxi projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := discovery.ListProjectNames()
			if err != nil {
				return err
			}
			if activeOnly {
				active, err := activeProjects()
				if err != nil {
					return err
				}
				filtered := make([]string, 0, len(names))
				for _, name := range names {
					if _, ok := active[name]; ok {
						filtered = append(filtered, name)
					}
				}
				names = filtered
			}
			fmt.Fprintln(cmd.OutOrStdout(), "tmuxi projects:")
			if newline {
				printLines(cmd, names)
				return nil
			}
			for _, name := range names {
				fmt.Fprintln(cmd.OutOrStdout(), name)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&newline, "newline", "n", false, "print one project per line")
	cmd.Flags().BoolVarP(&activeOnly, "active", "a", false, "only print active sessions")
	return cmd
}
