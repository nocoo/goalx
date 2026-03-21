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

	out := captureStdout(t, func() {
		if err := Result(projectRoot, nil); err != nil {
			t.Fatalf("Result: %v", err)
		}
	})

	if !strings.Contains(out, "# newer summary") {
		t.Fatalf("result output missing latest summary:\n%s", out)
	}
	if strings.Contains(out, "# older summary") {
		t.Fatalf("result output should use latest saved run:\n%s", out)
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

func writeSavedResultRun(t *testing.T, projectRoot, runName string, cfg goalx.Config, files map[string]string) string {
	t.Helper()

	runDir := filepath.Join(projectRoot, ".goalx", "runs", runName)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "goalx.yaml"), data, 0o644); err != nil {
		t.Fatalf("write goalx.yaml: %v", err)
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
