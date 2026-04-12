package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"muxi/internal/discovery"
	"muxi/internal/tmux"
)

func newNewCommand() *cobra.Command {
	var local bool

	cmd := &cobra.Command{
		Use:     "new [project] [session]",
		Aliases: []string{"open", "edit", "o", "e", "n"},
		Short:   "Create or open a muxi project file",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := currentDir()
			if err != nil {
				return err
			}
			name := args[0]
			var path string
			if local {
				path = discovery.LocalDefaultPath(cwd)
			} else {
				path, err = discovery.DefaultProjectPath(name)
				if err != nil {
					return err
				}
			}

			if len(args) == 2 {
				content, err := tmux.ImportSession(name, args[1])
				if err != nil {
					return err
				}
				if err := writeProjectFile(path, content); err != nil {
					return err
				}
			} else if _, err := os.Stat(path); os.IsNotExist(err) {
				template := map[string]any{
					"name": name,
					"root": "~/",
					"windows": []any{
						map[string]any{"editor": map[string]any{"layout": "main-vertical", "panes": []any{"vim", "guard"}}},
						map[string]any{"server": "bundle exec rails s"},
						map[string]any{"logs": "tail -f log/development.log"},
					},
				}
				if err := writeProjectFile(path, template); err != nil {
					return err
				}
			}

			if editor := os.Getenv("EDITOR"); editor != "" {
				editCmd := exec.Command(editor, path)
				editCmd.Stdin = os.Stdin
				editCmd.Stdout = os.Stdout
				editCmd.Stderr = os.Stderr
				return editCmd.Run()
			}

			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&local, "local", "l", false, "create local project file at ./.tmuxinator.yml")
	return cmd
}
