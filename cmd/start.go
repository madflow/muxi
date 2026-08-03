package cmd

import (
	"muxi/internal/runtime"
	"muxi/internal/tmux"

	"github.com/spf13/cobra"
)

func newStartCommand() *cobra.Command {
	var attach bool
	var customName string
	var projectConfig string
	var appendMode bool
	var noPreWindow bool

	cmd := &cobra.Command{
		Use:     "start [project]",
		Aliases: []string{"s"},
		Short:   "Start a tmux session from a project config",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(cmd, args)
		},
	}

	cmd.Flags().BoolVarP(&attach, "attach", "a", false, "attach to the tmux session after creation")
	cmd.Flags().StringVarP(&customName, "name", "n", "", "give the session a different name")
	cmd.Flags().StringVarP(&projectConfig, "project-config", "p", "", "path to project config file")
	cmd.Flags().BoolVar(&appendMode, "append", false, "append windows to the current session")
	cmd.Flags().BoolVar(&noPreWindow, "no-pre-window", false, "skip pre_window commands")

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(withStartOptions(cmd.Context(), startCommandOptions{
			attach:        attach,
			attachChanged: cmd.Flags().Lookup("attach").Changed,
			customName:    customName,
			projectConfig: projectConfig,
			appendMode:    appendMode,
			noPreWindow:   noPreWindow,
		}))
		return nil
	}

	return cmd
}

type startCommandOptions struct {
	attach        bool
	attachChanged bool
	customName    string
	projectConfig string
	appendMode    bool
	noPreWindow   bool
}

func runStart(cmd *cobra.Command, args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}
	opts := startOptionsFromContext(cmd.Context())
	prepared, err := loadPreparedProject(name, opts.projectConfig, runtime.StartOptions{
		CustomName:  opts.customName,
		ForceAttach: attachPointer(opts),
		Append:      opts.appendMode,
		NoPreWindow: opts.noPreWindow,
	})
	if err != nil {
		return err
	}
	if !tmux.Installed() {
		return tmux.RunScript(runtime.BuildStartScript(prepared, runtime.StartOptions{
			CustomName:  opts.customName,
			ForceAttach: attachPointer(opts),
			Append:      opts.appendMode,
			NoPreWindow: opts.noPreWindow,
		}))
	}
	return tmux.RunScript(runtime.BuildStartScript(prepared, runtime.StartOptions{
		CustomName:  opts.customName,
		ForceAttach: attachPointer(opts),
		Append:      opts.appendMode,
		NoPreWindow: opts.noPreWindow,
	}))
}

func attachPointer(opts startCommandOptions) *bool {
	if !opts.attachChanged {
		return nil
	}
	return boolPtr(opts.attach)
}
