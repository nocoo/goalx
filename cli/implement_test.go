package cli

import (
	"os"
	"path/filepath"
	"testing"

	goalx "github.com/vonbai/goalx"
)

func TestImplementPreservesSavedMasterConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectRoot := t.TempDir()
	writeSavedRunFixture(t, projectRoot, "research-a", goalx.Config{
		Name:      "research-a",
		Mode:      goalx.ModeResearch,
		Objective: "audit auth flow",
		Preset:    "claude",
		Parallel:  2,
		Master: goalx.MasterConfig{
			Engine: "codex",
			Model:  "gpt-5.4",
		},
	}, map[string]string{
		"summary.md":          "# summary\n",
		"session-1-report.md": "# report\n",
	})

	if err := Implement(projectRoot, []string{"--from", "research-a", "--write-config"}, nil); err != nil {
		t.Fatalf("Implement: %v", err)
	}

	cfg, err := goalx.LoadYAML[goalx.Config](filepath.Join(projectRoot, ".goalx", "goalx.yaml"))
	if err != nil {
		t.Fatalf("load goalx.yaml: %v", err)
	}
	if cfg.Master.Engine != "codex" || cfg.Master.Model != "gpt-5.4" {
		t.Fatalf("master = %s/%s, want codex/gpt-5.4", cfg.Master.Engine, cfg.Master.Model)
	}
}

func TestImplementAppliesNextConfigOverrides(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".goalx"), 0o755); err != nil {
		t.Fatalf("mkdir .goalx: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, ".goalx", "config.yaml"), []byte("target:\n  files: [cli/]\nharness:\n  command: go test ./...\n"), 0o644); err != nil {
		t.Fatalf("write base config: %v", err)
	}
	writeSavedRunFixture(t, projectRoot, "debate", goalx.Config{
		Name:      "debate",
		Mode:      goalx.ModeResearch,
		Objective: "consensus fixes",
		Preset:    "claude",
		Parallel:  2,
	}, map[string]string{
		"summary.md":          "# summary\n",
		"session-1-report.md": "# report\n",
	})

	nc := &nextConfigJSON{
		Parallel:      4,
		Engine:        "codex",
		Model:         "fast",
		Dimensions:    []string{"depth", "adversarial", "evidence", "perfectionist"},
		BudgetSeconds: 1200,
		Objective:     "custom implement objective",
	}
	if err := Implement(projectRoot, []string{"--from", "debate", "--write-config"}, nc); err != nil {
		t.Fatalf("Implement: %v", err)
	}

	cfg, err := goalx.LoadYAML[goalx.Config](filepath.Join(projectRoot, ".goalx", "goalx.yaml"))
	if err != nil {
		t.Fatalf("load goalx.yaml: %v", err)
	}
	if cfg.Parallel != 4 {
		t.Fatalf("parallel = %d, want 4", cfg.Parallel)
	}
	if cfg.Roles.Develop.Engine != "codex" || cfg.Roles.Develop.Model != "fast" {
		t.Fatalf("develop role = %s/%s, want codex/fast", cfg.Roles.Develop.Engine, cfg.Roles.Develop.Model)
	}
	if cfg.Objective != "custom implement objective" {
		t.Fatalf("objective = %q, want custom implement objective", cfg.Objective)
	}
	if cfg.Budget.MaxDuration != 20*60*1_000_000_000 {
		t.Fatalf("budget = %v, want 20m", cfg.Budget.MaxDuration)
	}
	if len(cfg.Sessions) != 4 || cfg.Sessions[3].Hint != goalx.BuiltinDimensions["perfectionist"] {
		t.Fatalf("sessions = %#v, want hinted sessions from dimensions", cfg.Sessions)
	}
}

func TestImplementResolvesNextConfigDimensionsIntoHints(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectRoot := t.TempDir()
	writeSavedRunFixture(t, projectRoot, "debate", goalx.Config{
		Name:      "debate",
		Mode:      goalx.ModeResearch,
		Objective: "consensus fixes",
		Preset:    "claude",
		Parallel:  2,
	}, map[string]string{
		"summary.md":          "# summary\n",
		"session-1-report.md": "# report\n",
	})

	nc := &nextConfigJSON{
		Parallel:   3,
		Dimensions: []string{"depth", "adversarial", "evidence"},
	}
	if err := Implement(projectRoot, []string{"--from", "debate", "--write-config"}, nc); err != nil {
		t.Fatalf("Implement: %v", err)
	}

	cfg, err := goalx.LoadYAML[goalx.Config](filepath.Join(projectRoot, ".goalx", "goalx.yaml"))
	if err != nil {
		t.Fatalf("load goalx.yaml: %v", err)
	}
	wantHints := []string{
		goalx.BuiltinDimensions["depth"],
		goalx.BuiltinDimensions["adversarial"],
		goalx.BuiltinDimensions["evidence"],
	}
	if len(cfg.Sessions) != len(wantHints) {
		t.Fatalf("sessions = %#v, want %#v", cfg.Sessions, wantHints)
	}
	for i := range wantHints {
		if cfg.Sessions[i].Hint != wantHints[i] {
			t.Fatalf("sessions[%d].hint = %q, want %q", i, cfg.Sessions[i].Hint, wantHints[i])
		}
	}
}

func TestImplementAppliesNextConfigPreset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectRoot := t.TempDir()
	writeSavedRunFixture(t, projectRoot, "debate", goalx.Config{
		Name:      "debate",
		Mode:      goalx.ModeResearch,
		Objective: "consensus fixes",
		Preset:    "claude",
		Parallel:  2,
	}, map[string]string{
		"summary.md":          "# summary\n",
		"session-1-report.md": "# report\n",
	})

	if err := Implement(projectRoot, []string{"--from", "debate", "--write-config"}, &nextConfigJSON{Preset: "claude-h"}); err != nil {
		t.Fatalf("Implement: %v", err)
	}

	cfg, err := goalx.LoadYAML[goalx.Config](filepath.Join(projectRoot, ".goalx", "goalx.yaml"))
	if err != nil {
		t.Fatalf("load goalx.yaml: %v", err)
	}
	if cfg.Preset != "claude-h" {
		t.Fatalf("preset = %q, want claude-h", cfg.Preset)
	}
	if cfg.Roles.Develop.Engine != "claude-code" || cfg.Roles.Develop.Model != "opus" {
		t.Fatalf("develop role = %s/%s, want claude-code/opus", cfg.Roles.Develop.Engine, cfg.Roles.Develop.Model)
	}
}

func TestImplementUsesSavedManifestReportArtifacts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectRoot := t.TempDir()
	writeSavedRunFixture(t, projectRoot, "debate", goalx.Config{
		Name:      "debate",
		Mode:      goalx.ModeResearch,
		Objective: "consensus fixes",
		Preset:    "claude",
		Parallel:  1,
	}, nil)
	runDir := SavedRunDir(projectRoot, "debate")
	reportPath := filepath.Join(runDir, "custom-findings.txt")
	if err := os.WriteFile(reportPath, []byte("report\n"), 0o644); err != nil {
		t.Fatalf("write custom report: %v", err)
	}
	if err := SaveArtifacts(filepath.Join(runDir, "artifacts.json"), &ArtifactsManifest{
		Run:     "debate",
		Version: 1,
		Sessions: []SessionArtifacts{
			{
				Name: "session-1",
				Mode: string(goalx.ModeResearch),
				Artifacts: []ArtifactMeta{
					{Kind: "report", Path: reportPath, RelPath: "custom-findings.txt", DurableName: "session-1-report.md"},
				},
			},
		},
	}); err != nil {
		t.Fatalf("SaveArtifacts: %v", err)
	}

	if err := Implement(projectRoot, []string{"--from", "debate", "--write-config"}, nil); err != nil {
		t.Fatalf("Implement: %v", err)
	}

	cfg, err := goalx.LoadYAML[goalx.Config](filepath.Join(projectRoot, ".goalx", "goalx.yaml"))
	if err != nil {
		t.Fatalf("load goalx.yaml: %v", err)
	}
	found := false
	for _, path := range cfg.Context.Files {
		if path == reportPath {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("context.files = %#v, want %q from artifacts manifest", cfg.Context.Files, reportPath)
	}
}

func TestImplementStartCreatesFreshCharterWithPreservedRootLineage(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectRoot := initGitRepo(t)
	writeAndCommit(t, projectRoot, "base.txt", "base", "base commit")
	sourceMeta, sourceCharter := writeSavedPhaseSourceFixture(t, projectRoot, "debate", "debate")
	installPhaseStartFakeTmux(t)
	stubLaunchRunSidecar(t)

	if err := Implement(projectRoot, []string{"--from", "debate"}, nil); err != nil {
		t.Fatalf("Implement: %v", err)
	}

	assertPhaseRunLineage(t, projectRoot, derivePhaseRunName("debate", "implement", ""), "implement", "debate", sourceMeta, sourceCharter)
}
