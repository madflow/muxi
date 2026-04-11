package cmd

import "github.com/spf13/cobra"

func newLocalCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "local",
		Aliases: []string{"."},
		Short:   "Start the local .tmuxinator.yml project",
		RunE:    runLocal,
	}
	return cmd
}

func runLocal(cmd *cobra.Command, _ []string) error {
	ctx := withStartOptions(cmd.Context(), startCommandOptions{})
	cmd.SetContext(ctx)
	return runStart(cmd, nil)
}
