package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

var deprecatedMessages = map[string]string{
	"tabs":        "DEPRECATION: tabs has been replaced by windows.",
	"cli_args":    "DEPRECATION: cli_args has been replaced by tmux_options.",
	"pre":         "DEPRECATION: pre has been replaced by project hooks.",
	"post":        "DEPRECATION: post has been replaced by project hooks.",
	"rbenv":       "DEPRECATION: rbenv has been replaced by pre_window.",
	"rvm":         "DEPRECATION: rvm has been replaced by pre_window.",
	"sync_legacy": "DEPRECATION: synchronize: true/before is legacy; prefer synchronize: after.",
}

type Project struct {
	SourcePath          string
	Name                string
	Root                string
	SocketName          string
	SocketPath          string
	TmuxCommand         string
	TmuxOptions         string
	Pre                 string
	Post                string
	PreWindow           string
	OnProjectStart      string
	OnProjectFirstStart string
	OnProjectRestart    string
	OnProjectExit       string
	OnProjectStop       string
	StartupWindow       string
	StartupPane         string
	EnablePaneTitles    bool
	PaneTitlePosition   string
	PaneTitleFormat     string
	Attach              *bool
	Windows             []Window
	DeprecatedNotices   []string
}

type Window struct {
	Name        *string
	Root        string
	Layout      string
	Synchronize string
	Pre         string
	Commands    []string
	Panes       []Pane
}

type Pane struct {
	Title    *string
	Commands []string
}

func Load(path string) (*Project, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("failed to parse config file: expected a YAML mapping")
	}

	root := doc.Content[0]
	project := &Project{
		SourcePath:  path,
		TmuxCommand: "tmux",
	}

	if name, ok := mappingValue(root, "name"); ok {
		project.Name = nodeString(name)
	} else if legacy, ok := mappingValue(root, "project_name"); ok {
		project.Name = nodeString(legacy)
	}

	if rootNode, ok := mappingValue(root, "root"); ok {
		project.Root = expandPath(nodeString(rootNode), "")
	} else if legacy, ok := mappingValue(root, "project_root"); ok {
		project.Root = expandPath(nodeString(legacy), "")
	}

	project.SocketName = mappingString(root, "socket_name")
	project.SocketPath = mappingString(root, "socket_path")
	if command := mappingString(root, "tmux_command"); command != "" {
		project.TmuxCommand = command
	}
	project.TmuxOptions = mappingString(root, "tmux_options")
	if project.TmuxOptions == "" {
		project.TmuxOptions = mappingString(root, "cli_args")
		if project.TmuxOptions != "" {
			project.DeprecatedNotices = append(project.DeprecatedNotices, deprecatedMessages["cli_args"])
		}
	}

	project.Pre = mappingCommand(root, "pre", "; ")
	project.Post = mappingCommand(root, "post", "; ")
	project.PreWindow = mappingCommand(root, "pre_window", "; ")
	if project.PreWindow == "" {
		if rbenv := mappingString(root, "rbenv"); rbenv != "" {
			project.PreWindow = "rbenv shell " + rbenv
			project.DeprecatedNotices = append(project.DeprecatedNotices, deprecatedMessages["rbenv"])
		} else if rvm := mappingString(root, "rvm"); rvm != "" {
			project.PreWindow = "rvm use " + rvm
			project.DeprecatedNotices = append(project.DeprecatedNotices, deprecatedMessages["rvm"])
		}
	}

	project.OnProjectStart = mappingCommand(root, "on_project_start", "; ")
	project.OnProjectFirstStart = mappingCommand(root, "on_project_first_start", "; ")
	project.OnProjectRestart = mappingCommand(root, "on_project_restart", "; ")
	project.OnProjectExit = mappingCommand(root, "on_project_exit", "; ")
	project.OnProjectStop = mappingCommand(root, "on_project_stop", "; ")
	project.StartupWindow = mappingString(root, "startup_window")
	project.StartupPane = mappingString(root, "startup_pane")
	project.EnablePaneTitles = mappingBool(root, "enable_pane_titles")
	project.PaneTitlePosition = mappingString(root, "pane_title_position")
	project.PaneTitleFormat = mappingString(root, "pane_title_format")
	if attachNode, ok := mappingValue(root, "attach"); ok {
		attach := nodeBool(attachNode)
		project.Attach = &attach
	}

	windowsNode, ok := mappingValue(root, "windows")
	if !ok {
		windowsNode, ok = mappingValue(root, "tabs")
		if ok {
			project.DeprecatedNotices = append(project.DeprecatedNotices, deprecatedMessages["tabs"])
		}
	}
	if ok {
		windows, err := parseWindows(windowsNode, project.Root)
		if err != nil {
			return nil, err
		}
		project.Windows = windows
		if hasLegacySynchronize(windows) {
			project.DeprecatedNotices = append(project.DeprecatedNotices, deprecatedMessages["sync_legacy"])
		}
	}

	if project.Pre != "" {
		project.DeprecatedNotices = append(project.DeprecatedNotices, deprecatedMessages["pre"])
	}
	if project.Post != "" {
		project.DeprecatedNotices = append(project.DeprecatedNotices, deprecatedMessages["post"])
	}

	if err := project.Validate(); err != nil {
		return nil, err
	}

	return project, nil
}

func (p *Project) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("your project file didn't specify a 'project_name'")
	}
	if len(p.Windows) == 0 {
		return fmt.Errorf("your project file should include some windows")
	}
	return nil
}

func (p *Project) AttachEnabled(force *bool) bool {
	if force != nil {
		return *force
	}
	if p.Attach == nil {
		return true
	}
	return *p.Attach
}

func parseWindows(node *yaml.Node, projectRoot string) ([]Window, error) {
	resolved := deref(node)
	if resolved.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("failed to parse config file: windows must be a YAML array")
	}

	windows := make([]Window, 0, len(resolved.Content))
	for _, item := range resolved.Content {
		entry, err := parseWindow(item, projectRoot)
		if err != nil {
			return nil, err
		}
		windows = append(windows, entry)
	}

	return windows, nil
}

func parseWindow(node *yaml.Node, projectRoot string) (Window, error) {
	resolved := deref(node)
	if resolved.Kind != yaml.MappingNode || len(resolved.Content) < 2 {
		return Window{}, fmt.Errorf("failed to parse config file: each window must be a single-entry mapping")
	}

	key := deref(resolved.Content[0])
	value := deref(resolved.Content[1])

	var name *string
	if !isNull(key) {
		windowName := nodeString(key)
		name = &windowName
	}

	window := Window{Name: name}

	switch value.Kind {
	case yaml.MappingNode:
		window.Root = expandPath(mappingString(value, "root"), projectRoot)
		window.Layout = mappingString(value, "layout")
		window.Pre = mappingCommand(value, "pre", " && ")
		window.Synchronize = mappingSynchronize(value, "synchronize")
		if panesNode, ok := mappingValue(value, "panes"); ok {
			panes, err := parsePanes(panesNode)
			if err != nil {
				return Window{}, err
			}
			window.Panes = panes
		}
	case yaml.SequenceNode:
		window.Commands = nodeCommandList(value)
	default:
		if !isNull(value) {
			window.Commands = []string{nodeString(value)}
		}
	}

	if window.Root == "" {
		window.Root = projectRoot
	}

	return window, nil
}

func parsePanes(node *yaml.Node) ([]Pane, error) {
	resolved := deref(node)
	if resolved.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("failed to parse config file: panes must be a YAML array")
	}

	panes := make([]Pane, 0, len(resolved.Content))
	for _, item := range resolved.Content {
		pane, err := parsePane(item)
		if err != nil {
			return nil, err
		}
		panes = append(panes, pane)
	}

	return panes, nil
}

func parsePane(node *yaml.Node) (Pane, error) {
	resolved := deref(node)
	if isNull(resolved) {
		return Pane{}, nil
	}

	switch resolved.Kind {
	case yaml.MappingNode:
		if len(resolved.Content) < 2 {
			return Pane{}, nil
		}
		key := deref(resolved.Content[0])
		value := deref(resolved.Content[1])
		var title *string
		if !isNull(key) {
			t := nodeString(key)
			title = &t
		}
		return Pane{Title: title, Commands: nodeCommandList(value)}, nil
	case yaml.SequenceNode:
		return Pane{Commands: nodeCommandList(resolved)}, nil
	default:
		return Pane{Commands: []string{nodeString(resolved)}}, nil
	}
}

func mappingValue(node *yaml.Node, key string) (*yaml.Node, bool) {
	resolved := deref(node)
	if resolved.Kind != yaml.MappingNode {
		return nil, false
	}
	for i := 0; i+1 < len(resolved.Content); i += 2 {
		if resolved.Content[i].Value == key {
			return deref(resolved.Content[i+1]), true
		}
	}
	return nil, false
}

func mappingString(node *yaml.Node, key string) string {
	if value, ok := mappingValue(node, key); ok {
		return nodeString(value)
	}
	return ""
}

func mappingBool(node *yaml.Node, key string) bool {
	if value, ok := mappingValue(node, key); ok {
		return nodeBool(value)
	}
	return false
}

func mappingCommand(node *yaml.Node, key, joiner string) string {
	if value, ok := mappingValue(node, key); ok {
		list := nodeCommandList(value)
		return strings.Join(list, joiner)
	}
	return ""
}

func mappingSynchronize(node *yaml.Node, key string) string {
	value, ok := mappingValue(node, key)
	if !ok || isNull(value) {
		return ""
	}
	if nodeBool(value) {
		return "before"
	}
	return nodeString(value)
}

func nodeCommandList(node *yaml.Node) []string {
	resolved := deref(node)
	if isNull(resolved) {
		return nil
	}

	if resolved.Kind == yaml.SequenceNode {
		commands := make([]string, 0, len(resolved.Content))
		for _, item := range resolved.Content {
			if isNull(deref(item)) {
				continue
			}
			commands = append(commands, nodeString(item))
		}
		return commands
	}

	return []string{nodeString(resolved)}
}

func nodeString(node *yaml.Node) string {
	resolved := deref(node)
	if isNull(resolved) {
		return ""
	}

	var value any
	if err := resolved.Decode(&value); err != nil {
		return resolved.Value
	}
	return stringify(value)
}

func nodeBool(node *yaml.Node) bool {
	resolved := deref(node)
	var value bool
	if err := resolved.Decode(&value); err == nil {
		return value
	}
	text := strings.TrimSpace(strings.ToLower(nodeString(resolved)))
	return text == "true" || text == "yes" || text == "on"
}

func stringify(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprint(v)
	}
}

func expandPath(path, base string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				path = home
			} else {
				path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
			}
		}
	}

	if base != "" && !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}

	expanded, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return expanded
}

func hasLegacySynchronize(windows []Window) bool {
	for _, window := range windows {
		if window.Synchronize == "before" {
			return true
		}
	}
	return false
}

func deref(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.AliasNode && node.Alias != nil {
		return deref(node.Alias)
	}
	return node
}

func isNull(node *yaml.Node) bool {
	if node == nil {
		return true
	}
	resolved := deref(node)
	return resolved.Tag == "!!null" || (resolved.Kind == yaml.ScalarNode && resolved.Value == "")
}
