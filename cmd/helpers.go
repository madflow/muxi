package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"

	"muxi/internal/discovery"
	"muxi/internal/project"
	"muxi/internal/runtime"
	"muxi/internal/tmux"
)

func currentDir() (string, error) {
	return os.Getwd()
}

func loadPreparedProject(name, projectConfig string, opts runtime.StartOptions) (*runtime.PreparedProject, error) {
	cwd, err := currentDir()
	if err != nil {
		return nil, err
	}
	path, err := discovery.ResolveProject(name, projectConfig, cwd)
	if err != nil {
		return nil, err
	}
	p, err := project.Load(path)
	if err != nil {
		return nil, err
	}
	return runtime.Prepare(p, opts)
}

func writeProjectFile(path string, content any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(content)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func boolPtr(value bool) *bool {
	return &value
}

func confirmDelete() error {
	return errors.New("interactive confirmation is not implemented yet; rerun non-destructively or remove files manually")
}

func printLines(cmd *cobra.Command, lines []string) {
	for _, line := range lines {
		fmt.Fprintln(cmd.OutOrStdout(), line)
	}
}

func activeProjects() (map[string]struct{}, error) {
	sessions, err := tmux.ActiveSessions()
	if err != nil {
		return nil, err
	}
	active := make(map[string]struct{}, len(sessions))
	for _, name := range sessions {
		active[name] = struct{}{}
	}
	return active, nil
}
