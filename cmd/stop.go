package cmd

import (
	"github.com/spf13/cobra"

	"muxi/internal/runtime"
	"muxi/internal/tmux"
)

func newStopCommand() *cobra.Command {
	var projectConfig string
	var suppressWarning bool

	cmd := &cobra.Command{
		Use:     "stop [project]",
		Aliases: []string{"st"},
		Short:   "Stop a tmux session using a project config",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if len(args) > 0 {
				name = args[0]
			}
			prepared, err := loadPreparedProject(name, projectConfig, runtime.StartOptions{SuppressTmuxVersionWarning: suppressWarning})
			if err != nil {
				return err
			}
			return tmux.RunScript(runtime.BuildStopScript(prepared))
		},
	}

	cmd.Flags().StringVarP(&projectConfig, "project-config", "p", "", "path to project config file")
	cmd.Flags().BoolVar(&suppressWarning, "suppress-tmux-version-warning", false, "suppress unsupported tmux version warning")
	return cmd
}
