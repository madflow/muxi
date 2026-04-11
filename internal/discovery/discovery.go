package discovery

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var localDefaults = []string{".tmuxinator.yml", ".tmuxinator.yaml"}

func LocalProject(cwd string) (string, bool) {
	for _, name := range localDefaults {
		path := filepath.Join(cwd, name)
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	return "", false
}

func ResolveProject(name, projectConfig, cwd string) (string, error) {
	if strings.TrimSpace(projectConfig) != "" {
		if _, err := os.Stat(projectConfig); err != nil {
			return "", fmt.Errorf("project config (%s) doesn't exist", projectConfig)
		}
		return projectConfig, nil
	}

	if strings.TrimSpace(name) == "" {
		if local, ok := LocalProject(cwd); ok {
			return local, nil
		}
		return "", fmt.Errorf("project file at ./.tmuxinator.yml doesn't exist")
	}

	path, ok, err := ProjectByName(name)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("project %s doesn't exist", name)
	}
	return path, nil
}

func ProjectByName(name string) (string, bool, error) {
	for _, directory := range searchDirectories() {
		projects, err := allProjects(directory, true)
		if err != nil {
			return "", false, err
		}
		for _, project := range projects {
			base := strings.TrimSuffix(filepath.Base(project), filepath.Ext(project))
			if base == name {
				return project, true, nil
			}
		}
	}
	return "", false, nil
}

func DefaultProjectPath(name string) (string, error) {
	dir, err := ConfigDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".yml"), nil
}

func LocalDefaultPath(cwd string) string {
	return filepath.Join(cwd, localDefaults[0])
}

func ConfigDirectory() (string, error) {
	if env := environmentDirectory(); env != "" {
		return env, nil
	}

	xdg := xdgDirectory()
	if isDir(xdg) {
		return xdg, nil
	}

	home := homeDirectory()
	if isDir(home) {
		return home, nil
	}

	if err := os.MkdirAll(xdg, 0o755); err != nil {
		return "", err
	}
	return xdg, nil
}

func Directories() []string {
	if env := environmentDirectory(); env != "" {
		return []string{env}
	}

	dirs := make([]string, 0, 2)
	if isDir(xdgDirectory()) {
		dirs = append(dirs, xdgDirectory())
	}
	if isDir(homeDirectory()) {
		dirs = append(dirs, homeDirectory())
	}
	return dirs
}

func ListProjectNames() ([]string, error) {
	var names []string
	for _, dir := range Directories() {
		projects, err := allProjects(dir, false)
		if err != nil {
			return nil, err
		}
		for _, path := range projects {
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				continue
			}
			names = append(names, strings.TrimSuffix(filepath.ToSlash(rel), ".yml"))
		}
	}
	sort.Strings(names)
	return names, nil
}

func environmentDirectory() string {
	dir := strings.TrimSpace(os.Getenv("TMUXINATOR_CONFIG"))
	if dir == "" {
		return ""
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return dir
	}
	return dir
}

func xdgDirectory() string {
	base := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "tmuxinator")
}

func homeDirectory() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".tmuxinator")
}

func searchDirectories() []string {
	return []string{environmentDirectory(), xdgDirectory(), homeDirectory()}
}

func allProjects(root string, includeYAML bool) ([]string, error) {
	if strings.TrimSpace(root) == "" || !isDir(root) {
		return nil, nil
	}

	projects := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".yml" || (includeYAML && ext == ".yaml") {
			projects = append(projects, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(projects)
	return projects, nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
