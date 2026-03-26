package cli

import (
	"testing"

	goalx "github.com/vonbai/goalx"
)

func TestGlobalRunRegistryLookupHydratesRunIDFromRunScopedMetadata(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "README.md", "demo", "base commit")

	cfg := &goalx.Config{
		Name:      "alpha",
		Mode:      goalx.ModeResearch,
		Objective: "audit alpha",
	}
	runDir := writeRunSpecFixture(t, repo, cfg)
	meta, err := EnsureRunMetadata(runDir, repo, cfg.Objective)
	if err != nil {
		t.Fatalf("EnsureRunMetadata: %v", err)
	}

	reg := &GlobalRunRegistry{
		Version: 1,
		Runs: map[string]GlobalRunRef{
			globalRunKey(repo, cfg.Name): {
				Name:        cfg.Name,
				ProjectID:   goalx.ProjectID(repo),
				ProjectRoot: repo,
				RunDir:      runDir,
				State:       "active",
			},
		},
	}
	if err := SaveGlobalRunRegistry(reg); err != nil {
		t.Fatalf("SaveGlobalRunRegistry: %v", err)
	}

	matches, err := LookupGlobalRuns(meta.RunID)
	if err != nil {
		t.Fatalf("LookupGlobalRuns: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("matches len = %d, want 1", len(matches))
	}
	if matches[0].RunID != meta.RunID {
		t.Fatalf("matches[0].RunID = %q, want %q", matches[0].RunID, meta.RunID)
	}
}
