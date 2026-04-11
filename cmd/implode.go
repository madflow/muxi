package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"tmuxi/internal/discovery"
)

func newImplodeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "implode",
		Aliases: []string{"i"},
		Short:   "Delete all tmuxi project directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, dir := range discovery.Directories() {
				if err := os.RemoveAll(dir); err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}
