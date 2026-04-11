package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var supportedVersions = []string{
	"3.6a", "3.6", "3.5a", "3.5", "3.4", "3.3a", "3.3", "3.2a", "3.2",
	"3.1c", "3.1b", "3.1a", "3.1", "3.0a", "3.0", "2.9a", "2.9", "2.8",
	"2.7", "2.6", "2.5", "2.4", "2.3", "2.2", "2.1", "2.0", "1.9", "1.8",
	"1.7", "1.6", "1.5",
}

type Version struct {
	Raw       string
	Number    float64
	Supported bool
}

func Installed() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

func DetectVersion() Version {
	output, err := exec.Command("tmux", "-V").Output()
	if err != nil {
		return Version{}
	}
	raw := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(output)), "tmux "))
	number := parseVersionNumber(raw)
	return Version{Raw: raw, Number: number, Supported: isSupported(raw, number)}
}

func UnsupportedVersionMessage() string {
	items := append([]string(nil), supportedVersions...)
	sort.Strings(items)
	return "WARNING: You are running tmuxi with an unsupported version of tmux. Please consider using a supported version: (" + strings.Join(items, ", ") + ")"
}

func ActiveSessions() ([]string, error) {
	output, err := exec.Command("tmux", "list-sessions", "-F", "#S").Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) == 0 {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var sessions []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			sessions = append(sessions, line)
		}
	}
	return sessions, nil
}

func RunScript(script string) error {
	cmd := exec.Command(shell(), "-lc", script)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Output(script string) (string, error) {
	cmd := exec.Command(shell(), "-lc", script)
	data, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return strings.TrimSpace(string(exitErr.Stderr)), err
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func shell() string {
	if value := strings.TrimSpace(os.Getenv("SHELL")); value != "" {
		return value
	}
	return "/bin/bash"
}

func parseVersionNumber(raw string) float64 {
	re := regexp.MustCompile(`\d+(?:\.\d+)?`)
	match := re.FindString(raw)
	if match == "" {
		return 0
	}
	value, err := strconv.ParseFloat(match, 64)
	if err != nil {
		return 0
	}
	return value
}

func isSupported(raw string, number float64) bool {
	for _, item := range supportedVersions {
		if raw == item {
			return true
		}
	}
	if number == 0 {
		return false
	}
	formatted := strconv.FormatFloat(number, 'f', -1, 64)
	for _, item := range supportedVersions {
		if item == formatted {
			return true
		}
	}
	return false
}

func CurrentSessionName(tmuxBase string) (string, error) {
	return Output(tmuxBase + ` display-message -p '#S'`)
}

func SessionExists(tmuxBase, name string) (bool, error) {
	output, err := Output(tmuxBase + " ls 2>/dev/null")
	if err != nil && output == "" {
		return false, nil
	}
	prefix := name + ":"
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			return true, nil
		}
	}
	return false, nil
}

func BaseIndexes(tmuxBase string) (int, int, error) {
	output, err := Output(tmuxBase + " start-server; " + tmuxBase + " show-option -g base-index; " + tmuxBase + " show-window-option -g pane-base-index")
	if err != nil && output == "" {
		return 0, 0, err
	}
	baseIndex := 0
	paneBaseIndex := 0
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 {
			continue
		}
		value, convErr := strconv.Atoi(fields[1])
		if convErr != nil {
			continue
		}
		switch fields[0] {
		case "base-index":
			baseIndex = value
		case "pane-base-index":
			paneBaseIndex = value
		}
	}
	return baseIndex, paneBaseIndex, nil
}

func LastWindowIndex(tmuxBase string) (int, error) {
	output, err := Output(tmuxBase + " list-windows -F '#I'")
	if err != nil {
		return 0, err
	}
	last := 0
	for _, line := range strings.Split(output, "\n") {
		value, convErr := strconv.Atoi(strings.TrimSpace(line))
		if convErr == nil {
			last = value
		}
	}
	return last, nil
}

func ImportSession(name, session string) (map[string]any, error) {
	version := DetectVersion()
	if version.Number > 0 && version.Number < 1.6 {
		return nil, fmt.Errorf("creating projects from sessions is unsupported for tmux version 1.5 or lower")
	}

	windowsRaw, err := exec.Command(shell(), "-lc", `tmux list-windows -t `+quote(session)+` -F '#W #{window_layout} #{window_active} #{pane_current_path}'`).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("session %q doesn't exist", session)
	}
	panesRaw, err := exec.Command(shell(), "-lc", `tmux list-panes -s -t `+quote(session)+` -F '#W #{pane_current_path}'`).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("session %q doesn't exist", session)
	}
	optionsRaw, err := exec.Command(shell(), "-lc", `tmux show-options -t `+quote(session)).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("session %q doesn't exist", session)
	}

	panePaths := map[string][]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(panesRaw)), "\n") {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		windowName := parts[0]
		panePaths[windowName] = append(panePaths[windowName], strings.Join(parts[1:], " "))
	}

	projectRoot := ""
	for _, line := range strings.Split(string(optionsRaw), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "default-path ") {
			projectRoot = strings.Trim(strings.TrimPrefix(line, "default-path "), `"`)
			break
		}
	}

	windows := make([]any, 0)
	for _, line := range strings.Split(strings.TrimSpace(string(windowsRaw)), "\n") {
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}
		windowName := parts[0]
		layout := parts[1]
		active := parts[2]
		path := strings.Join(parts[3:], " ")
		if projectRoot == "" && active == "1" {
			projectRoot = path
		}
		paneCommands := make([]any, 0, len(panePaths[windowName]))
		for _, panePath := range panePaths[windowName] {
			paneCommands = append(paneCommands, "cd "+panePath)
		}
		windows = append(windows, map[string]any{
			windowName: map[string]any{
				"layout": layout,
				"panes":  paneCommands,
			},
		})
	}

	return map[string]any{
		"name":         name,
		"project_root": projectRoot,
		"windows":      windows,
	}, nil
}

func quote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
