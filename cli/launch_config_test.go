package cli

import (
	"os"
	"path/filepath"
	"testing"

	goalx "github.com/vonbai/goalx"
)

func TestBuildLaunchConfigPreservesConfiguredParallelWhenFlagOmitted(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".goalx"), 0o755); err != nil {
		t.Fatalf("mkdir .goalx: %v", err)
	}
	cfgYAML := `
preset: hybrid
parallel: 4
master:
  engine: codex
  model: best
roles:
  research:
    engine: claude-code
    model: opus
  develop:
    engine: codex
    model: fast
`
	if err := os.WriteFile(filepath.Join(projectRoot, ".goalx", "config.yaml"), []byte(cfgYAML), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	cfg, err := buildLaunchConfig(projectRoot, launchOptions{
		Objective: "audit auth",
		Mode:      goalx.ModeResearch,
	})
	if err != nil {
		t.Fatalf("buildLaunchConfig: %v", err)
	}
	if cfg.Parallel != 4 {
		t.Fatalf("parallel = %d, want 4", cfg.Parallel)
	}
}

func TestBuildLaunchConfigOverridesParallelWhenFlagProvided(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".goalx"), 0o755); err != nil {
		t.Fatalf("mkdir .goalx: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, ".goalx", "config.yaml"), []byte("parallel: 4\n"), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	cfg, err := buildLaunchConfig(projectRoot, launchOptions{
		Objective: "audit auth",
		Mode:      goalx.ModeResearch,
		Parallel:  2,
	})
	if err != nil {
		t.Fatalf("buildLaunchConfig: %v", err)
	}
	if cfg.Parallel != 2 {
		t.Fatalf("parallel = %d, want 2", cfg.Parallel)
	}
}
