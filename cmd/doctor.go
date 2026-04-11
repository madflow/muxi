package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"tmuxi/internal/tmux"
)

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check whether tmuxi can run on this machine",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "Checking if tmux is installed ==> %t\n", tmux.Installed())
			fmt.Fprintf(cmd.OutOrStdout(), "Checking if $EDITOR is set ==> %t\n", os.Getenv("EDITOR") != "")
			fmt.Fprintf(cmd.OutOrStdout(), "Checking if $SHELL is set ==> %t\n", os.Getenv("SHELL") != "")
		},
	}
}
