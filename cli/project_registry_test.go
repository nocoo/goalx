package cli

import (
	"testing"
	"time"

	goalx "github.com/vonbai/goalx"
)

func TestProjectRegistryFocusedRunFallsBackToRunScopedTruth(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "README.md", "demo", "base commit")

	activeCfg := &goalx.Config{
		Name:      "alpha",
		Mode:      goalx.ModeDevelop,
		Objective: "ship alpha",
	}
	activeRun := writeRunSpecFixture(t, repo, activeCfg)
	if err := SaveControlRunState(ControlRunStatePath(activeRun), &ControlRunState{
		Version:        1,
		LifecycleState: "active",
	}); err != nil {
		t.Fatalf("SaveControlRunState: %v", err)
	}
	if err := RenewControlLease(activeRun, "sidecar", "run_alpha", 1, time.Minute, "process", 4242); err != nil {
		t.Fatalf("RenewControlLease: %v", err)
	}

	if err := SaveProjectRegistry(repo, &ProjectRegistry{
		Version:    1,
		FocusedRun: "ghost",
		ActiveRuns: map[string]ProjectRunRef{
			"ghost": {Name: "ghost", State: "active"},
		},
	}); err != nil {
		t.Fatalf("SaveProjectRegistry: %v", err)
	}

	got, err := ResolveDefaultRunName(repo)
	if err != nil {
		t.Fatalf("ResolveDefaultRunName: %v", err)
	}
	if got != "alpha" {
		t.Fatalf("ResolveDefaultRunName = %q, want alpha", got)
	}
}
