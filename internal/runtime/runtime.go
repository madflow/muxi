package runtime

import (
	"fmt"
	"strings"

	"muxi/internal/project"
	"muxi/internal/shellquote"
	"muxi/internal/tmux"
)

type StartOptions struct {
	CustomName                 string
	ForceAttach                *bool
	Append                     bool
	NoPreWindow                bool
	SuppressTmuxVersionWarning bool
}

type PreparedProject struct {
	Project       *project.Project
	Name          string
	TmuxBase      string
	SessionExists bool
	BaseIndex     int
	PaneBaseIndex int
	Version       tmux.Version
	Attach        bool
	Warnings      []string
}

func Prepare(p *project.Project, opts StartOptions) (*PreparedProject, error) {
	tmuxBase := strings.TrimSpace(p.TmuxCommand)
	if tmuxBase == "" {
		tmuxBase = "tmux"
	}
	if p.TmuxOptions != "" {
		tmuxBase += " " + strings.TrimSpace(p.TmuxOptions)
	}
	if p.SocketPath != "" {
		tmuxBase += " -S " + shellquote.Quote(p.SocketPath)
	} else if p.SocketName != "" {
		tmuxBase += " -L " + shellquote.Quote(p.SocketName)
	}

	name := p.Name
	if opts.Append {
		current, err := tmux.CurrentSessionName(tmuxBase)
		if err != nil || strings.TrimSpace(current) == "" {
			return nil, fmt.Errorf("cannot append to a session that does not exist")
		}
		name = current
	} else if strings.TrimSpace(opts.CustomName) != "" {
		name = opts.CustomName
	}

	sessionExists, err := tmux.SessionExists(tmuxBase, name)
	if err != nil {
		return nil, err
	}
	if opts.Append && !sessionExists {
		return nil, fmt.Errorf("cannot append to a session that does not exist")
	}

	baseIndex, paneBaseIndex, err := tmux.BaseIndexes(tmuxBase)
	if err != nil {
		return nil, err
	}
	if opts.Append {
		lastWindow, err := tmux.LastWindowIndex(tmuxBase)
		if err != nil {
			return nil, err
		}
		baseIndex = lastWindow + 1
	}

	prepared := &PreparedProject{
		Project:       p,
		Name:          name,
		TmuxBase:      tmuxBase,
		SessionExists: sessionExists,
		BaseIndex:     baseIndex,
		PaneBaseIndex: paneBaseIndex,
		Version:       tmux.DetectVersion(),
		Attach:        p.AttachEnabled(opts.ForceAttach),
		Warnings:      append([]string(nil), p.DeprecatedNotices...),
	}

	if p.EnablePaneTitles {
		if prepared.Version.Number > 0 && prepared.Version.Number < 2.6 {
			prepared.Warnings = append(prepared.Warnings, "WARNING: pane titles require tmux >= 2.6.")
		}
		if pos := p.PaneTitlePosition; pos != "" && pos != "top" && pos != "bottom" && pos != "off" {
			prepared.Warnings = append(prepared.Warnings, "WARNING: invalid pane_title_position, expected top, bottom, or off.")
		}
	}
	if !prepared.Version.Supported && !opts.SuppressTmuxVersionWarning {
		prepared.Warnings = append(prepared.Warnings, tmux.UnsupportedVersionMessage())
	}

	return prepared, nil
}

func BuildStartScript(prepared *PreparedProject, opts StartOptions) string {
	p := prepared.Project
	lines := make([]string, 0, 128)
	if !opts.Append {
		lines = append(lines, "unset RBENV_VERSION")
		lines = append(lines, "unset RBENV_DIR")
		lines = append(lines, prepared.TmuxBase+" start-server")
	}

	lines = append(lines, "cd "+shellquote.Quote(defaultRoot(p.Root)))
	appendIf(&lines, p.OnProjectStart)

	if opts.Append || !prepared.SessionExists {
		appendIf(&lines, p.Pre)
		appendIf(&lines, p.OnProjectFirstStart)
		if !opts.Append {
			lines = append(lines, newSessionCommand(prepared))
		}
		if prepared.Version.Number > 0 && prepared.Version.Number < 1.7 && p.Root != "" {
			lines = append(lines, prepared.TmuxBase+" set-option -t "+shellquote.Quote(prepared.Name)+" default-path "+shellquote.Quote(p.Root)+" 1>/dev/null")
		}
		lines = append(lines, warningLines(prepared.Warnings)...)
		for index, window := range p.Windows {
			lines = append(lines, newWindowCommand(prepared, window, index))
		}
		for index, window := range p.Windows {
			lines = append(lines, buildWindowCommands(prepared, window, index, opts.NoPreWindow)...)
		}
		lines = append(lines, prepared.TmuxBase+" select-window -t "+shellquote.Quote(startupWindow(prepared)))
		lines = append(lines, prepared.TmuxBase+" select-pane -t "+shellquote.Quote(startupPane(prepared)))
	} else {
		appendIf(&lines, p.OnProjectRestart)
	}

	if prepared.Attach && !opts.Append {
		lines = append(lines, prepared.TmuxBase+" -u attach-session -t "+shellquote.Quote(prepared.Name))
	}

	appendIf(&lines, p.Post)
	appendIf(&lines, p.OnProjectExit)

	return strings.Join(lines, "\n") + "\n"
}

func BuildStopScript(prepared *PreparedProject) string {
	lines := []string{
		"if " + prepared.TmuxBase + " has-session -t " + shellquote.Quote(prepared.Name) + " 2>/dev/null; then",
		"  cd " + shellquote.Quote(defaultRoot(prepared.Project.Root)),
	}
	if prepared.Project.OnProjectStop != "" {
		lines = append(lines, "  "+prepared.Project.OnProjectStop)
	}
	lines = append(lines,
		"  "+prepared.TmuxBase+" kill-session -t "+shellquote.Quote(prepared.Name),
		"fi",
	)
	return strings.Join(lines, "\n") + "\n"
}

func buildWindowCommands(prepared *PreparedProject, window project.Window, index int, noPreWindow bool) []string {
	target := windowTarget(prepared, index)
	lines := make([]string, 0, 32)
	if window.Synchronize == "before" {
		lines = append(lines, prepared.TmuxBase+" set-window-option -t "+shellquote.Quote(target)+" synchronize-panes on")
	}
	if prepared.Project.EnablePaneTitles && prepared.Version.Number >= 2.6 {
		position := prepared.Project.PaneTitlePosition
		if position == "" || (position != "top" && position != "bottom" && position != "off") {
			position = "top"
		}
		format := prepared.Project.PaneTitleFormat
		if format == "" {
			format = "#{pane_index}: #{pane_title}"
		}
		lines = append(lines,
			prepared.TmuxBase+" set-window-option -t "+shellquote.Quote(target)+" pane-border-status "+shellquote.Quote(position),
			prepared.TmuxBase+" set-window-option -t "+shellquote.Quote(target)+" pane-border-format "+shellquote.Quote(format),
		)
	}

	if len(window.Panes) == 0 {
		if !noPreWindow && prepared.Project.PreWindow != "" {
			lines = append(lines, sendWindowKeys(prepared, index, prepared.Project.PreWindow))
		}
		for _, command := range window.Commands {
			lines = append(lines, sendWindowKeys(prepared, index, command))
		}
	} else {
		for paneIndex, pane := range window.Panes {
			if prepared.Project.EnablePaneTitles && prepared.Version.Number >= 2.6 && pane.Title != nil {
				lines = append(lines, prepared.TmuxBase+" select-pane -t "+shellquote.Quote(paneTarget(prepared, index, paneIndex))+" -T "+shellquote.Quote(*pane.Title))
			}
			if !noPreWindow && prepared.Project.PreWindow != "" {
				lines = append(lines, sendPaneKeys(prepared, index, paneIndex, prepared.Project.PreWindow))
			}
			if window.Pre != "" {
				lines = append(lines, sendPaneKeys(prepared, index, paneIndex, window.Pre))
			}
			for _, command := range pane.Commands {
				lines = append(lines, sendPaneKeys(prepared, index, paneIndex, command))
			}
			if paneIndex < len(window.Panes)-1 {
				lines = append(lines, splitPaneCommand(prepared, window, index))
				lines = append(lines, prepared.TmuxBase+" select-layout -t "+shellquote.Quote(target)+" tiled")
			}
		}
		if window.Layout != "" {
			lines = append(lines, prepared.TmuxBase+" select-layout -t "+shellquote.Quote(target)+" "+shellquote.Quote(window.Layout))
		}
		lines = append(lines, prepared.TmuxBase+" select-pane -t "+shellquote.Quote(paneTarget(prepared, index, 0)))
	}

	if window.Synchronize == "after" {
		lines = append(lines, prepared.TmuxBase+" set-window-option -t "+shellquote.Quote(target)+" synchronize-panes on")
	}
	return lines
}

func newSessionCommand(prepared *PreparedProject) string {
	windowNameOpt := ""
	if len(prepared.Project.Windows) > 0 && prepared.Project.Windows[0].Name != nil {
		windowNameOpt = " -n " + shellquote.Quote(*prepared.Project.Windows[0].Name)
	}
	return prepared.TmuxBase + " new-session -d -s " + shellquote.Quote(prepared.Name) + windowNameOpt
}

func newWindowCommand(prepared *PreparedProject, window project.Window, index int) string {
	parts := []string{prepared.TmuxBase, "new-window"}
	if root := windowRoot(window); root != "" {
		parts = append(parts, defaultPathFlag(prepared.Version), shellquote.Quote(root))
	}
	parts = append(parts, "-k", "-t", shellquote.Quote(windowTarget(prepared, index)))
	if window.Name != nil {
		parts = append(parts, "-n", shellquote.Quote(*window.Name))
	}
	return strings.Join(parts, " ")
}

func splitPaneCommand(prepared *PreparedProject, window project.Window, index int) string {
	parts := []string{prepared.TmuxBase, "splitw"}
	if root := windowRoot(window); root != "" {
		parts = append(parts, defaultPathFlag(prepared.Version), shellquote.Quote(root))
	}
	parts = append(parts, "-t", shellquote.Quote(windowTarget(prepared, index)))
	return strings.Join(parts, " ")
}

func sendWindowKeys(prepared *PreparedProject, windowIndex int, command string) string {
	return prepared.TmuxBase + " send-keys -t " + shellquote.Quote(windowTarget(prepared, windowIndex)) + " " + shellquote.Quote(command) + " C-m"
}

func sendPaneKeys(prepared *PreparedProject, windowIndex, paneIndex int, command string) string {
	return prepared.TmuxBase + " send-keys -t " + shellquote.Quote(paneTarget(prepared, windowIndex, paneIndex)) + " " + shellquote.Quote(command) + " C-m"
}

func startupWindow(prepared *PreparedProject) string {
	value := prepared.Project.StartupWindow
	if value == "" {
		value = fmt.Sprintf("%d", prepared.BaseIndex)
	}
	return prepared.Name + ":" + value
}

func startupPane(prepared *PreparedProject) string {
	value := prepared.Project.StartupPane
	if value == "" {
		value = fmt.Sprintf("%d", prepared.PaneBaseIndex)
	}
	return startupWindow(prepared) + "." + value
}

func windowTarget(prepared *PreparedProject, index int) string {
	return fmt.Sprintf("%s:%d", prepared.Name, index+prepared.BaseIndex)
}

func paneTarget(prepared *PreparedProject, windowIndex, paneIndex int) string {
	return fmt.Sprintf("%s:%d.%d", prepared.Name, windowIndex+prepared.BaseIndex, paneIndex+prepared.PaneBaseIndex)
}

func windowRoot(window project.Window) string {
	return defaultRoot(window.Root)
}

func defaultRoot(root string) string {
	if strings.TrimSpace(root) == "" {
		return "."
	}
	return root
}

func appendIf(lines *[]string, command string) {
	if strings.TrimSpace(command) != "" {
		*lines = append(*lines, command)
	}
}

func warningLines(warnings []string) []string {
	lines := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		if strings.TrimSpace(warning) == "" {
			continue
		}
		lines = append(lines, `printf '%s\n' `+shellquote.Quote(warning)+` >&2`)
	}
	return lines
}

func defaultPathFlag(version tmux.Version) string {
	if version.Number > 0 && version.Number < 1.8 {
		return "default-path"
	}
	return "-c"
}
