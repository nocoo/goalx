package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ar "github.com/vonbai/autoresearch"
	"gopkg.in/yaml.v3"
)

func TestReviewUsesConfiguredResearchTargetFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectRoot := t.TempDir()
	runName := "demo"
	runDir := ar.RunDir(projectRoot, runName)
	wtPath := WorktreePath(runDir, runName, 1)
	if err := os.MkdirAll(wtPath, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}

	cfg := ar.Config{
		Name:      runName,
		Mode:      ar.ModeResearch,
		Objective: "inspect",
		Parallel:  1,
		Target: ar.TargetConfig{
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

	want := "custom report body"
	if err := os.WriteFile(filepath.Join(wtPath, "notes.md"), []byte(want+"\n"), 0o644); err != nil {
		t.Fatalf("write notes.md: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Review(projectRoot, []string{"--run", runName}); err != nil {
			t.Fatalf("Review: %v", err)
		}
	})

	if !strings.Contains(out, want) {
		t.Fatalf("review output missing configured target file contents:\n%s", out)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return buf.String()
}
