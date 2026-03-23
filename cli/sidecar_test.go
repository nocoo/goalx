package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	goalx "github.com/vonbai/goalx"
)

func TestRunSidecarTickRenewsLease(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "README.md", "base", "base commit")
	cfg := &goalx.Config{
		Name:      "sidecar-run",
		Mode:      goalx.ModeDevelop,
		Objective: "ship feature",
		Master:    goalx.MasterConfig{Engine: "codex", Model: "codex"},
	}
	runDir := writeRunSpecFixture(t, repo, cfg)
	meta, err := EnsureRunMetadata(runDir, repo, cfg.Objective)
	if err != nil {
		t.Fatalf("EnsureRunMetadata: %v", err)
	}
	if _, err := EnsureRuntimeState(runDir, cfg); err != nil {
		t.Fatalf("EnsureRuntimeState: %v", err)
	}
	if err := EnsureControlState(runDir); err != nil {
		t.Fatalf("EnsureControlState: %v", err)
	}

	fakeBin := t.TempDir()
	tmuxPath := filepath.Join(fakeBin, "tmux")
	if err := os.WriteFile(tmuxPath, []byte("#!/bin/sh\nif [ \"$1\" = \"has-session\" ]; then exit 1; fi\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := runSidecarTick(repo, cfg.Name, runDir, meta.RunID, meta.Epoch, 2*time.Minute, 4242); err != nil {
		t.Fatalf("runSidecarTick: %v", err)
	}

	lease, err := LoadControlLease(ControlLeasePath(runDir, "sidecar"))
	if err != nil {
		t.Fatalf("LoadControlLease: %v", err)
	}
	if lease.Holder != "sidecar" {
		t.Fatalf("lease holder = %q, want sidecar", lease.Holder)
	}
	if lease.RunID != meta.RunID {
		t.Fatalf("lease run id = %q, want %q", lease.RunID, meta.RunID)
	}
	if lease.Epoch != meta.Epoch {
		t.Fatalf("lease epoch = %d, want %d", lease.Epoch, meta.Epoch)
	}
	if lease.PID != 4242 {
		t.Fatalf("lease pid = %d, want 4242", lease.PID)
	}
	if lease.RenewedAt == "" || lease.ExpiresAt == "" {
		t.Fatalf("lease timestamps missing: %+v", lease)
	}
	heartbeat, err := LoadHeartbeatState(HeartbeatStatePath(runDir))
	if err != nil {
		t.Fatalf("LoadHeartbeatState: %v", err)
	}
	if heartbeat.Seq != 1 {
		t.Fatalf("heartbeat seq = %d, want 1", heartbeat.Seq)
	}
}

func TestStopTerminatesSidecar(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "base.txt", "base", "base commit")
	runName, runDir := writeLifecycleRunFixture(t, repo)
	cfg, err := LoadRunSpec(runDir)
	if err != nil {
		t.Fatalf("LoadRunSpec: %v", err)
	}
	if err := RegisterActiveRun(repo, cfg); err != nil {
		t.Fatalf("RegisterActiveRun: %v", err)
	}

	origStopSidecar := stopRunSidecar
	defer func() { stopRunSidecar = origStopSidecar }()
	var gotRunDir string
	stopRunSidecar = func(runDir string) error {
		gotRunDir = runDir
		return nil
	}

	if err := Stop(repo, []string{"--run", runName}); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if gotRunDir != runDir {
		t.Fatalf("stopRunSidecar runDir = %q, want %q", gotRunDir, runDir)
	}
}

func TestDropTerminatesSidecarBeforeRemovingRunDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "base.txt", "base", "base commit")

	runName := "drop-run"
	runDir := goalx.RunDir(repo, runName)
	for _, dir := range []string{
		filepath.Join(runDir, "journals"),
		filepath.Join(runDir, "worktrees"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(RunSpecPath(runDir), []byte("name: drop-run\nmode: research\nobjective: demo\ntarget:\n  files: [\"report.md\"]\nharness:\n  command: \"test -f base.txt\"\n"), 0o644); err != nil {
		t.Fatalf("write run snapshot: %v", err)
	}

	origStopSidecar := stopRunSidecar
	defer func() { stopRunSidecar = origStopSidecar }()
	var gotRunDir string
	stopRunSidecar = func(runDir string) error {
		gotRunDir = runDir
		return nil
	}

	if err := Drop(repo, []string{"--run", runName}); err != nil {
		t.Fatalf("Drop: %v", err)
	}
	if gotRunDir != runDir {
		t.Fatalf("stopRunSidecar runDir = %q, want %q", gotRunDir, runDir)
	}
}

func TestSidecarRenewsLeaseUntilContextStops(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "README.md", "base", "base commit")
	cfg := &goalx.Config{
		Name:      "sidecar-run",
		Mode:      goalx.ModeDevelop,
		Objective: "ship feature",
		Master:    goalx.MasterConfig{Engine: "codex", Model: "codex"},
	}
	runDir := writeRunSpecFixture(t, repo, cfg)
	meta, err := EnsureRunMetadata(runDir, repo, cfg.Objective)
	if err != nil {
		t.Fatalf("EnsureRunMetadata: %v", err)
	}
	if _, err := EnsureRuntimeState(runDir, cfg); err != nil {
		t.Fatalf("EnsureRuntimeState: %v", err)
	}
	if err := EnsureControlState(runDir); err != nil {
		t.Fatalf("EnsureControlState: %v", err)
	}
	fakeBin := t.TempDir()
	tmuxPath := filepath.Join(fakeBin, "tmux")
	if err := os.WriteFile(tmuxPath, []byte("#!/bin/sh\nif [ \"$1\" = \"has-session\" ]; then exit 1; fi\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runSidecarLoop(ctx, repo, cfg.Name, runDir, meta.RunID, meta.Epoch, 50*time.Millisecond)
	}()

	time.Sleep(120 * time.Millisecond)
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("runSidecarLoop: %v", err)
	}

	lease, err := LoadControlLease(ControlLeasePath(runDir, "sidecar"))
	if err != nil {
		t.Fatalf("LoadControlLease: %v", err)
	}
	if lease.RenewedAt == "" {
		t.Fatalf("sidecar lease renewed_at empty: %+v", lease)
	}
}

func TestSidecarStopsWhenRunIdentityChanges(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "README.md", "base", "base commit")
	cfg := &goalx.Config{
		Name:      "sidecar-run",
		Mode:      goalx.ModeDevelop,
		Objective: "ship feature",
		Master:    goalx.MasterConfig{Engine: "codex", Model: "codex"},
	}
	runDir := writeRunSpecFixture(t, repo, cfg)
	meta, err := EnsureRunMetadata(runDir, repo, cfg.Objective)
	if err != nil {
		t.Fatalf("EnsureRunMetadata: %v", err)
	}
	if _, err := EnsureRuntimeState(runDir, cfg); err != nil {
		t.Fatalf("EnsureRuntimeState: %v", err)
	}
	if err := EnsureControlState(runDir); err != nil {
		t.Fatalf("EnsureControlState: %v", err)
	}

	fakeBin := t.TempDir()
	tmuxPath := filepath.Join(fakeBin, "tmux")
	if err := os.WriteFile(tmuxPath, []byte("#!/bin/sh\nif [ \"$1\" = \"has-session\" ]; then exit 1; fi\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- runSidecarLoop(ctx, repo, cfg.Name, runDir, meta.RunID, meta.Epoch, 50*time.Millisecond)
	}()

	time.Sleep(60 * time.Millisecond)
	meta.RunID = newRunID()
	if err := SaveRunMetadata(RunMetadataPath(runDir), meta); err != nil {
		t.Fatalf("SaveRunMetadata: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runSidecarLoop: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("sidecar did not stop after run identity changed")
	}
}
