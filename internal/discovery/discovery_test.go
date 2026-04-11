package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalProjectPrefersYMLThenYAML(t *testing.T) {
	tempDir := t.TempDir()
	ymlPath := filepath.Join(tempDir, ".tmuxinator.yml")
	yamlPath := filepath.Join(tempDir, ".tmuxinator.yaml")
	if err := os.WriteFile(yamlPath, []byte("name: yaml\nwindows:\n  - editor: vim\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ymlPath, []byte("name: yml\nwindows:\n  - editor: vim\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	path, ok := LocalProject(tempDir)
	if !ok {
		t.Fatalf("expected local project")
	}
	if path != ymlPath {
		t.Fatalf("path = %q, want %q", path, ymlPath)
	}
}

func TestResolveProjectUsesExplicitConfigFirst(t *testing.T) {
	tempDir := t.TempDir()
	projectPath := filepath.Join(tempDir, "custom.yml")
	if err := os.WriteFile(projectPath, []byte("name: custom\nwindows:\n  - editor: vim\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveProject("ignored", projectPath, tempDir)
	if err != nil {
		t.Fatalf("ResolveProject() error = %v", err)
	}
	if resolved != projectPath {
		t.Fatalf("resolved = %q, want %q", resolved, projectPath)
	}
}
