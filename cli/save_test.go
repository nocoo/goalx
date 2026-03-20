package cli

import (
	"os"
	"path/filepath"
	"testing"

	goalx "github.com/vonbai/goalx"
	"gopkg.in/yaml.v3"
)

func TestSaveUsesConfiguredResearchTargetFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectRoot := t.TempDir()
	runName := "demo"
	runDir := goalx.RunDir(projectRoot, runName)
	wtPath := WorktreePath(runDir, runName, 1)
	if err := os.MkdirAll(wtPath, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}

	cfg := goalx.Config{
		Name:      runName,
		Mode:      goalx.ModeResearch,
		Objective: "inspect",
		Parallel:  1,
		Target: goalx.TargetConfig{
			Files: []string{"notes.md"},
		},
	}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "goalx.yaml"), data, 0o644); err != nil {
		t.Fatalf("write run snapshot: %v", err)
	}

	want := "saved custom report"
	if err := os.WriteFile(filepath.Join(wtPath, "notes.md"), []byte(want+"\n"), 0o644); err != nil {
		t.Fatalf("write notes.md: %v", err)
	}

	if err := Save(projectRoot, []string{"--run", runName}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	savedPath := filepath.Join(projectRoot, ".goalx", "runs", runName, "session-1-report.md")
	got, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("read saved report: %v", err)
	}
	if string(got) != want+"\n" {
		t.Fatalf("saved report = %q, want %q", string(got), want+"\n")
	}
}
