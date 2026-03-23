package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	goalx "github.com/vonbai/goalx"
	"gopkg.in/yaml.v3"
)

func TestResultPrintsLatestResearchSummary(t *testing.T) {
	projectRoot := t.TempDir()

	olderDir := writeSavedResultRun(t, projectRoot, "older-run", goalx.Config{
		Name: "older-run",
		Mode: goalx.ModeResearch,
		Target: goalx.TargetConfig{
			Files: []string{"report.md"},
		},
	}, map[string]string{
		"summary.md": "# older summary\n",
	})
	newerDir := writeSavedResultRun(t, projectRoot, "newer-run", goalx.Config{
		Name: "newer-run",
		Mode: goalx.ModeResearch,
		Target: goalx.TargetConfig{
			Files: []string{"report.md"},
		},
	}, map[string]string{
		"summary.md": "# newer summary\n",
	})

	oldTime := time.Now().Add(-time.Hour)
	newTime := time.Now()
	if err := os.Chtimes(olderDir, oldTime, oldTime); err != nil {
		t.Fatalf("chtimes older run: %v", err)
	}
	if err := os.Chtimes(newerDir, newTime, newTime); err != nil {
		t.Fatalf("chtimes newer run: %v", err)
	}

	err := Result(projectRoot, nil)
	if err == nil {
		t.Fatalf("Result should require an explicit run when multiple saved runs exist")
	}
	if !strings.Contains(err.Error(), "multiple saved runs") {
		t.Fatalf("Result error = %v, want multiple saved runs", err)
	}
}

func TestResultPrintsDevelopBranchSummary(t *testing.T) {
	projectRoot := initGitRepo(t)
	writeAndCommit(t, projectRoot, "README.md", "base\n", "base commit")

	headBranch := currentBranchName(t, projectRoot)
	branch := "goalx/dev-run/1"
	runGit(t, projectRoot, "checkout", "-b", branch)
	writeAndCommit(t, projectRoot, "README.md", "base\nupdated\n", "feat: update readme")
	runGit(t, projectRoot, "checkout", headBranch)

	runDir := writeSavedResultRun(t, projectRoot, "dev-run", goalx.Config{
		Name: "dev-run",
		Mode: goalx.ModeDevelop,
		Target: goalx.TargetConfig{
			Files: []string{"README.md"},
		},
	}, nil)

	selection := map[string]string{
		"kept":   "session-1",
		"branch": branch,
	}
	data, err := json.Marshal(selection)
	if err != nil {
		t.Fatalf("marshal selection: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "selection.json"), data, 0o644); err != nil {
		t.Fatalf("write selection.json: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Result(projectRoot, []string{"dev-run"}); err != nil {
			t.Fatalf("Result: %v", err)
		}
	})

	for _, want := range []string{
		"session-1",
		"feat: update readme",
		"README.md |",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("result output missing %q:\n%s", want, out)
		}
	}
}

func TestResultPrintsSmartResearchSummaryByDefault(t *testing.T) {
	projectRoot := t.TempDir()

	writeSavedResultRun(t, projectRoot, "smart-run", goalx.Config{
		Name: "smart-run",
		Mode: goalx.ModeResearch,
		Target: goalx.TargetConfig{
			Files: []string{"report.md"},
		},
	}, map[string]string{
		"summary.md": strings.TrimSpace(`
# Summary

## Key Findings
- finding 1
- finding 2
- finding 3
- finding 4
- finding 5
- finding 6

## Priority Fix List
- P0: config.go
- P1: cli/result.go

## Recommendation
implement

## Appendix
hidden details
`) + "\n",
	})

	out := captureStdout(t, func() {
		if err := Result(projectRoot, []string{"smart-run"}); err != nil {
			t.Fatalf("Result: %v", err)
		}
	})

	for _, want := range []string{
		"=== Research Result ===",
		"Recommendation: implement",
		"- finding 1",
		"... (1 more lines)",
		"- P0: config.go",
		"Full report: goalx result --full",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("result output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "hidden details") {
		t.Fatalf("smart result output should omit appendix details:\n%s", out)
	}
}

func TestResultPrintsFullResearchSummaryWithFullFlag(t *testing.T) {
	projectRoot := t.TempDir()

	summary := strings.TrimSpace(`
# Summary

## Key Findings
- finding 1

## Recommendation
done

## Appendix
hidden details
`) + "\n"
	writeSavedResultRun(t, projectRoot, "smart-run", goalx.Config{
		Name: "smart-run",
		Mode: goalx.ModeResearch,
		Target: goalx.TargetConfig{
			Files: []string{"report.md"},
		},
	}, map[string]string{
		"summary.md": summary,
	})

	out := captureStdout(t, func() {
		if err := Result(projectRoot, []string{"smart-run", "--full"}); err != nil {
			t.Fatalf("Result: %v", err)
		}
	})

	if out != summary {
		t.Fatalf("full result output mismatch:\nwant:\n%s\ngot:\n%s", summary, out)
	}
}

func TestResultFallsBackToSavedManifestReportWhenSummaryMissing(t *testing.T) {
	projectRoot := t.TempDir()

	runDir := writeSavedResultRun(t, projectRoot, "report-only-run", goalx.Config{
		Name: "report-only-run",
		Mode: goalx.ModeResearch,
		Target: goalx.TargetConfig{
			Files: []string{"report.md"},
		},
	}, nil)
	reportPath := filepath.Join(runDir, "custom-findings.txt")
	if err := os.WriteFile(reportPath, []byte("# report only\n\nuse this\n"), 0o644); err != nil {
		t.Fatalf("write custom report: %v", err)
	}
	if err := SaveArtifacts(filepath.Join(runDir, "artifacts.json"), &ArtifactsManifest{
		Run:     "report-only-run",
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

	out := captureStdout(t, func() {
		if err := Result(projectRoot, []string{"report-only-run"}); err != nil {
			t.Fatalf("Result: %v", err)
		}
	})

	if !strings.Contains(out, "# report only") {
		t.Fatalf("result output missing manifest-backed report:\n%s", out)
	}
}

func TestResultReadsLegacyProjectScopedSavedRun(t *testing.T) {
	projectRoot := t.TempDir()
	runDir := LegacySavedRunDir(projectRoot, "legacy-run")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir legacy run dir: %v", err)
	}
	cfg := goalx.Config{
		Name:   "legacy-run",
		Mode:   goalx.ModeResearch,
		Target: goalx.TargetConfig{Files: []string{"report.md"}},
	}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(RunSpecPath(runDir), data, 0o644); err != nil {
		t.Fatalf("write run spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "summary.md"), []byte("# legacy summary\n"), 0o644); err != nil {
		t.Fatalf("write summary: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Result(projectRoot, []string{"legacy-run"}); err != nil {
			t.Fatalf("Result: %v", err)
		}
	})
	if !strings.Contains(out, "legacy summary") {
		t.Fatalf("result output missing legacy summary:\n%s", out)
	}
}

func TestResultHelpPrintsUsage(t *testing.T) {
	err := Result(t.TempDir(), []string{"--help"})
	if err == nil || !strings.Contains(err.Error(), "usage: goalx result [NAME] [--full]") {
		t.Fatalf("Result --help error = %v", err)
	}
}

func writeSavedResultRun(t *testing.T, projectRoot, runName string, cfg goalx.Config, files map[string]string) string {
	t.Helper()

	runDir := SavedRunDir(projectRoot, runName)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(RunSpecPath(runDir), data, 0o644); err != nil {
		t.Fatalf("write run-spec.yaml: %v", err)
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(runDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	return runDir
}

func currentBranchName(t *testing.T, repo string) string {
	t.Helper()

	cmd := exec.Command("git", "-C", repo, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git current branch: %v\n%s", err, string(out))
	}
	return strings.TrimSpace(string(out))
}
