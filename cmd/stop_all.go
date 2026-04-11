package cmd

import (
	"github.com/spf13/cobra"

	"tmuxi/internal/discovery"
	"tmuxi/internal/shellquote"
	"tmuxi/internal/tmux"
)

func newStopAllCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop-all",
		Short: "Stop all active tmux sessions discovered by tmuxi",
		RunE: func(cmd *cobra.Command, args []string) error {
			sessions, err := tmux.ActiveSessions()
			if err != nil {
				return err
			}
			knownProjects, err := discovery.ListProjectNames()
			if err != nil {
				return err
			}
			known := make(map[string]struct{}, len(knownProjects))
			for _, name := range knownProjects {
				known[name] = struct{}{}
			}
			for _, session := range sessions {
				if session == "" {
					continue
				}
				if _, ok := known[session]; !ok {
					continue
				}
				if err := tmux.RunScript("tmux kill-session -t " + shellquote.Quote(session)); err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}
