package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"muxi/internal/discovery"
)

func newCopyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "copy [existing] [new]",
		Aliases: []string{"c", "cp"},
		Short:   "Copy an existing project config",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			source, ok, err := discovery.ProjectByName(args[0])
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("project %s doesn't exist", args[0])
			}
			destination, err := discovery.DefaultProjectPath(args[1])
			if err != nil {
				return err
			}
			return copyFile(source, destination)
		},
	}
	return cmd
}

func copyFile(source, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
