package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	goalx "github.com/vonbai/goalx"
)

func TestRelaunchMasterIgnoresLegacyHandoffFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "base.txt", "base", "base commit")

	logPath := installFakeTmux(t, "master")
	runName, runDir := writeLifecycleRunFixture(t, repo)
	cfg, err := LoadRunSpec(runDir)
	if err != nil {
		t.Fatalf("LoadRunSpec: %v", err)
	}
	legacyPath := filepath.Join(ControlDir(runDir), "handoffs", "master.json")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("mkdir legacy handoff dir: %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("write malformed legacy handoff: %v", err)
	}

	err = relaunchMaster(repo, runDir, goalx.TmuxSessionName(repo, runName), cfg)
	if err != nil {
		t.Fatalf("relaunchMaster: %v", err)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read tmux log: %v", err)
	}
	logText := string(logData)
	for _, want := range []string{
		"kill-window -t " + goalx.TmuxSessionName(repo, runName) + ":master",
		"new-window -t " + goalx.TmuxSessionName(repo, runName) + " -n master -c " + RunWorktreePath(runDir),
		filepath.Join(runDir, "master.md"),
	} {
		if !strings.Contains(logText, want) {
			t.Fatalf("tmux log missing %q:\n%s", want, logText)
		}
	}
}
