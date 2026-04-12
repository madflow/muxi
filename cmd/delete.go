package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"muxi/internal/discovery"
)

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete [project]...",
		Aliases: []string{"d", "rm"},
		Short:   "Delete one or more project configs",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, name := range args {
				path, ok, err := discovery.ProjectByName(name)
				if err != nil {
					return err
				}
				if !ok {
					continue
				}
				if err := os.Remove(path); err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}
