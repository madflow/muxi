package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadParsesLegacyAndStructuredWindows(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "sample.yml")
	content := []byte(`
name: sample
root: ./repo
cli_args: -f ~/.tmux.conf
pre: echo boot
post: echo done
enable_pane_titles: true
windows:
  - editor:
      root: api
      layout: main-vertical
      synchronize: after
      pre:
        - echo before
        - source env
      panes:
        - editor:
          - vim
        - guard
  - logs:
      - tail -f log/development.log
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if p.Name != "sample" {
		t.Fatalf("Name = %q, want sample", p.Name)
	}
	if p.TmuxOptions != "-f ~/.tmux.conf" {
		t.Fatalf("TmuxOptions = %q", p.TmuxOptions)
	}
	if len(p.Windows) != 2 {
		t.Fatalf("len(Windows) = %d, want 2", len(p.Windows))
	}
	if p.Windows[0].Layout != "main-vertical" {
		t.Fatalf("Layout = %q", p.Windows[0].Layout)
	}
	if p.Windows[0].Synchronize != "after" {
		t.Fatalf("Synchronize = %q", p.Windows[0].Synchronize)
	}
	if got := p.Windows[0].Pre; got != "echo before && source env" {
		t.Fatalf("Window pre = %q", got)
	}
	if len(p.Windows[0].Panes) != 2 {
		t.Fatalf("len(Panes) = %d, want 2", len(p.Windows[0].Panes))
	}
	if p.Windows[0].Panes[0].Title == nil || *p.Windows[0].Panes[0].Title != "editor" {
		t.Fatalf("expected titled pane")
	}
	if len(p.DeprecatedNotices) == 0 {
		t.Fatalf("expected deprecation notices")
	}
}

func TestLoadSupportsNamelessWindowsAndNullPanes(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "sample.yml")
	content := []byte(`
name: sample
windows:
  - ~: echo unnamed
  - work:
      panes:
        -
        - top
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if p.Windows[0].Name != nil {
		t.Fatalf("expected nameless window")
	}
	if len(p.Windows[1].Panes) != 2 {
		t.Fatalf("len(Panes) = %d, want 2", len(p.Windows[1].Panes))
	}
	if len(p.Windows[1].Panes[0].Commands) != 0 {
		t.Fatalf("expected empty first pane")
	}
}
