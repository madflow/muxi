package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"muxi/internal/runtime"
)

func newDebugCommand() *cobra.Command {
	var attach bool
	var customName string
	var projectConfig string
	var appendMode bool
	var noPreWindow bool

	cmd := &cobra.Command{
		Use:   "debug [project]",
		Short: "Print the tmux commands generated for a project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if len(args) > 0 {
				name = args[0]
			}
			prepared, err := loadPreparedProject(name, projectConfig, runtime.StartOptions{
				CustomName:  customName,
				ForceAttach: debugAttachPointer(cmd, attach),
				Append:      appendMode,
				NoPreWindow: noPreWindow,
			})
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), runtime.BuildStartScript(prepared, runtime.StartOptions{
				CustomName:  customName,
				ForceAttach: debugAttachPointer(cmd, attach),
				Append:      appendMode,
				NoPreWindow: noPreWindow,
			}))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&attach, "attach", "a", false, "attach to the tmux session after creation")
	cmd.Flags().StringVarP(&customName, "name", "n", "", "give the session a different name")
	cmd.Flags().StringVarP(&projectConfig, "project-config", "p", "", "path to project config file")
	cmd.Flags().BoolVar(&appendMode, "append", false, "append windows to the current session")
	cmd.Flags().BoolVar(&noPreWindow, "no-pre-window", false, "skip pre_window commands")
	return cmd
}

func debugAttachPointer(cmd *cobra.Command, attach bool) *bool {
	if !cmd.Flags().Lookup("attach").Changed {
		return nil
	}
	return boolPtr(attach)
}
