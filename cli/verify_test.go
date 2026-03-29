package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	goalx "github.com/vonbai/goalx"
)

func TestVerifyUsesAcceptanceChecksAndWritesState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "README.md", "demo", "base commit")
	ensureSharedProofEvidence(t)

	runName := "verify-run"
	runDir := goalx.RunDir(repo, runName)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	if err := os.MkdirAll(ReportsDir(runDir), 0o755); err != nil {
		t.Fatalf("mkdir reports dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".goalx"), 0o755); err != nil {
		t.Fatalf("mkdir project .goalx: %v", err)
	}

	snapshot := []byte(`name: verify-run
mode: develop
objective: ship feature
target:
  files: ["README.md"]
local_validation:
  command: "test -f DOES-NOT-EXIST"
acceptance:
  command: "printf 'e2e ok\n'"
`)
	if err := os.WriteFile(RunSpecPath(runDir), snapshot, 0o644); err != nil {
		t.Fatalf("write run snapshot: %v", err)
	}
	goal := []byte(`{
  "version": 1,
  "required": [
    {
      "id": "req-1",
      "text": "ship feature",
      "source": "user",
      "role": "outcome",
      "state": "claimed",
      "evidence_paths": ["/tmp/e2e.txt"]
    }
  ],
  "optional": []
}`)
	if err := SaveRunMetadata(RunMetadataPath(runDir), &RunMetadata{
		Version:      1,
		Objective:    "ship feature",
		BaseRevision: strings.TrimSpace(gitOutput(t, repo, "rev-parse", "HEAD")),
	}); err != nil {
		t.Fatalf("write run metadata: %v", err)
	}
	seedRunCharterForTests(t, runDir, runName, repo)
	if err := os.WriteFile(GoalPath(runDir), goal, 0o644); err != nil {
		t.Fatalf("write goal state: %v", err)
	}

	if err := Verify(repo, []string{"--run", runName}); err != nil {
		t.Fatalf("Verify: %v", err)
	}

	stateData, err := os.ReadFile(filepath.Join(runDir, "acceptance.json"))
	if err != nil {
		t.Fatalf("read acceptance state: %v", err)
	}
	stateText := string(stateData)
	for _, want := range []string{
		`"goal_version": 1`,
		`"checks": [`,
		`"id": "chk-1"`,
		`"command": "printf 'e2e ok\n'"`,
		`"check_results": [`,
		`"exit_code": 0`,
	} {
		if !strings.Contains(stateText, want) {
			t.Fatalf("acceptance state missing %q:\n%s", want, stateText)
		}
	}
	// Framework must NOT derive status from exit code — that's the master's job
	if strings.Contains(stateText, `"status"`) {
		t.Fatalf("acceptance state must not contain derived status field:\n%s", stateText)
	}

}

func TestVerifyRunsAcceptanceInsideRunWorktree(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "README.md", "demo", "base commit")
	ensureSharedProofEvidence(t)

	runName := "verify-run"
	runDir := goalx.RunDir(repo, runName)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}

	snapshot := []byte(`name: verify-run
mode: develop
objective: ship feature
acceptance:
  command: "test -f run-worktree-only.txt"
`)
	if err := os.WriteFile(RunSpecPath(runDir), snapshot, 0o644); err != nil {
		t.Fatalf("write run snapshot: %v", err)
	}
	if err := SaveRunMetadata(RunMetadataPath(runDir), &RunMetadata{
		Version:      1,
		Objective:    "ship feature",
		BaseRevision: strings.TrimSpace(gitOutput(t, repo, "rev-parse", "HEAD")),
	}); err != nil {
		t.Fatalf("write run metadata: %v", err)
	}
	seedRunCharterForTests(t, runDir, runName, repo)
	if err := os.WriteFile(GoalPath(runDir), []byte(`{"version":1,"required":[{"id":"req-1","text":"ship feature","source":"user","role":"outcome","state":"claimed","evidence_paths":["/tmp/e2e.txt"]}],"optional":[]}`), 0o644); err != nil {
		t.Fatalf("write goal state: %v", err)
	}

	runWT := RunWorktreePath(runDir)
	if err := CreateWorktree(repo, runWT, "goalx/"+runName+"/root"); err != nil {
		t.Fatalf("CreateWorktree run root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runWT, "run-worktree-only.txt"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write run-worktree-only.txt: %v", err)
	}

	if err := Verify(repo, []string{"--run", runName}); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestVerifyRequiresAcceptanceChecks(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "README.md", "demo", "base commit")
	ensureSharedProofEvidence(t)

	runName := "verify-run"
	runDir := goalx.RunDir(repo, runName)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}

	snapshot := []byte(`name: verify-run
mode: develop
objective: ship feature
target:
  files: ["README.md"]
local_validation:
  command: "test -f DOES-NOT-EXIST"
`)
	if err := os.WriteFile(RunSpecPath(runDir), snapshot, 0o644); err != nil {
		t.Fatalf("write run snapshot: %v", err)
	}
	goal := []byte(`{
  "version": 1,
  "required": [
    {
      "id": "req-1",
      "text": "ship feature",
      "source": "user",
      "role": "outcome",
      "state": "claimed",
      "evidence_paths": ["/tmp/e2e.txt"]
    }
  ],
  "optional": []
}`)
	if err := SaveRunMetadata(RunMetadataPath(runDir), &RunMetadata{
		Version:      1,
		Objective:    "ship feature",
		BaseRevision: strings.TrimSpace(gitOutput(t, repo, "rev-parse", "HEAD")),
	}); err != nil {
		t.Fatalf("write run metadata: %v", err)
	}
	seedRunCharterForTests(t, runDir, runName, repo)
	if err := os.WriteFile(GoalPath(runDir), goal, 0o644); err != nil {
		t.Fatalf("write goal state: %v", err)
	}

	err := Verify(repo, []string{"--run", runName})
	if err == nil {
		t.Fatal("expected Verify to fail")
	}
	if !strings.Contains(err.Error(), "no acceptance checks configured") {
		t.Fatalf("Verify error = %v, want missing acceptance checks", err)
	}

	stateData, readErr := os.ReadFile(filepath.Join(runDir, "acceptance.json"))
	if readErr != nil {
		t.Fatalf("read acceptance state: %v", readErr)
	}
	stateText := string(stateData)
	for _, unwanted := range []string{
		`"command": "test -f DOES-NOT-EXIST"`,
		`"exit_code"`,
	} {
		if strings.Contains(stateText, unwanted) {
			t.Fatalf("acceptance state unexpectedly contains %q:\n%s", unwanted, stateText)
		}
	}
	if strings.Contains(stateText, `"status"`) {
		t.Fatalf("acceptance state must not contain derived status field:\n%s", stateText)
	}
}

func TestVerifyDoesNotRewriteGoalState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "README.md", "demo", "base commit")
	ensureSharedProofEvidence(t)

	runName := "verify-goal-readonly"
	runDir := goalx.RunDir(repo, runName)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}

	snapshot := []byte(`name: verify-goal-readonly
mode: develop
objective: ship feature
acceptance:
  command: "printf 'e2e ok\n'"
`)
	if err := os.WriteFile(RunSpecPath(runDir), snapshot, 0o644); err != nil {
		t.Fatalf("write run snapshot: %v", err)
	}
	goalBefore := []byte(`{
  "version": 1,
  "updated_at": "2026-03-27T00:00:00Z",
  "required": [
    {
      "id": "req-1",
      "text": "ship feature",
      "source": "user",
      "role": "outcome",
      "state": "claimed",
      "evidence_paths": ["/tmp/e2e.txt"]
    }
  ],
  "optional": []
}`)
	if err := os.WriteFile(GoalPath(runDir), goalBefore, 0o644); err != nil {
		t.Fatalf("write goal state: %v", err)
	}
	if err := SaveRunMetadata(RunMetadataPath(runDir), &RunMetadata{
		Version:      1,
		Objective:    "ship feature",
		BaseRevision: strings.TrimSpace(gitOutput(t, repo, "rev-parse", "HEAD")),
	}); err != nil {
		t.Fatalf("write run metadata: %v", err)
	}
	seedRunCharterForTests(t, runDir, runName, repo)

	if err := Verify(repo, []string{"--run", runName}); err != nil {
		t.Fatalf("Verify: %v", err)
	}

	assertFileUnchanged(t, GoalPath(runDir), goalBefore)
}

func TestVerifyResearchRequiresReportEvidenceManifest(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "README.md", "demo", "base commit")
	ensureSharedProofEvidence(t)

	runName := "verify-research-manifest-missing"
	runDir := seedResearchVerifyRun(t, repo, runName)

	if err := os.WriteFile(filepath.Join(ReportsDir(runDir), "architecture-options-comparison.md"), []byte("report\n"), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}

	err := Verify(repo, []string{"--run", runName})
	if err == nil {
		t.Fatal("expected Verify to fail")
	}
	if !strings.Contains(err.Error(), "evidence manifest") {
		t.Fatalf("Verify error = %v, want evidence manifest failure", err)
	}
}

func TestVerifyResearchRejectsMalformedReportEvidenceManifest(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "README.md", "demo", "base commit")
	ensureSharedProofEvidence(t)

	runName := "verify-research-manifest-invalid"
	runDir := seedResearchVerifyRun(t, repo, runName)

	reportPath := filepath.Join(ReportsDir(runDir), "architecture-options-comparison.md")
	if err := os.WriteFile(reportPath, []byte("report\n"), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := os.WriteFile(ReportEvidenceManifestPath(reportPath), []byte(fmt.Sprintf(`{
  "version": 1,
  "report_path": %q,
  "covers": ["ucl-research"],
  "repo_evidence_paths": [],
  "external_refs": [],
  "unexpected": true
}`, reportPath)), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := Verify(repo, []string{"--run", runName})
	if err == nil {
		t.Fatal("expected Verify to fail")
	}
	if !strings.Contains(err.Error(), "unknown field") && !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("Verify error = %v, want manifest parse failure", err)
	}
}

func TestVerifyResearchRequiresExternalRefsForComparisonClause(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "README.md", "demo", "base commit")
	ensureSharedProofEvidence(t)

	runName := "verify-research-external-refs-missing"
	runDir := seedResearchVerifyRun(t, repo, runName)

	reportPath := filepath.Join(ReportsDir(runDir), "architecture-options-comparison.md")
	evidencePath := filepath.Join(runDir, "source.txt")
	if err := os.WriteFile(reportPath, []byte("report\n"), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := os.WriteFile(evidencePath, []byte("evidence\n"), 0o644); err != nil {
		t.Fatalf("write evidence: %v", err)
	}
	if err := os.WriteFile(ReportEvidenceManifestPath(reportPath), []byte(fmt.Sprintf(`{
  "version": 1,
  "report_path": %q,
  "covers": ["ucl-research"],
  "repo_evidence_paths": [%q],
  "external_refs": []
}`, reportPath, evidencePath)), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := Verify(repo, []string{"--run", runName})
	if err == nil {
		t.Fatal("expected Verify to fail")
	}
	if !strings.Contains(err.Error(), "external_refs") {
		t.Fatalf("Verify error = %v, want external refs failure", err)
	}
}

func TestVerifyResearchPassesWithStructuredEvidence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "README.md", "demo", "base commit")
	ensureSharedProofEvidence(t)

	runName := "verify-research-structured"
	runDir := seedResearchVerifyRun(t, repo, runName)

	reportPath := filepath.Join(ReportsDir(runDir), "architecture-options-comparison.md")
	evidencePath := filepath.Join(runDir, "source.txt")
	if err := os.WriteFile(reportPath, []byte("report\n"), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := os.WriteFile(evidencePath, []byte("evidence\n"), 0o644); err != nil {
		t.Fatalf("write evidence: %v", err)
	}
	if err := os.WriteFile(ReportEvidenceManifestPath(reportPath), []byte(fmt.Sprintf(`{
  "version": 1,
  "report_path": %q,
  "covers": ["ucl-research"],
  "repo_evidence_paths": [%q],
  "external_refs": ["https://example.com/reference"]
}`, reportPath, evidencePath)), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if err := Verify(repo, []string{"--run", runName}); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func seedResearchVerifyRun(t *testing.T, repo, runName string) string {
	t.Helper()

	runDir := goalx.RunDir(repo, runName)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	if err := os.MkdirAll(ReportsDir(runDir), 0o755); err != nil {
		t.Fatalf("mkdir reports dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".goalx"), 0o755); err != nil {
		t.Fatalf("mkdir project .goalx: %v", err)
	}

	snapshot := []byte(`name: ` + runName + `
mode: research
objective: compare external reference architectures
acceptance:
  command: "printf 'research e2e ok\n'"
`)
	if err := os.WriteFile(RunSpecPath(runDir), snapshot, 0o644); err != nil {
		t.Fatalf("write run snapshot: %v", err)
	}
	contract := &ObjectiveContract{
		Version:       1,
		ObjectiveHash: "sha256:research",
		State:         objectiveContractStateLocked,
		Clauses: []ObjectiveClause{
			{
				ID:               "ucl-research",
				Text:             "compare external reference architectures",
				Kind:             objectiveClauseKindVerification,
				SourceExcerpt:    "compare external reference architectures",
				RequiredSurfaces: []ObjectiveRequiredSurface{objectiveRequiredSurfaceGoal},
			},
		},
	}
	if err := SaveObjectiveContract(ObjectiveContractPath(runDir), contract); err != nil {
		t.Fatalf("SaveObjectiveContract: %v", err)
	}
	goal := &GoalState{
		Version: 1,
		Required: []GoalItem{
			{
				ID:            "req-1",
				Text:          "compare external reference architectures",
				Source:        goalItemSourceUser,
				Role:          goalItemRoleOutcome,
				State:         goalItemStateClaimed,
				Covers:        []string{"ucl-research"},
				EvidencePaths: []string{ensureSharedProofEvidence(t)},
			},
		},
		Optional: []GoalItem{},
	}
	if err := SaveGoalState(GoalPath(runDir), goal); err != nil {
		t.Fatalf("SaveGoalState: %v", err)
	}
	if err := os.WriteFile(SummaryPath(runDir), []byte("# summary\n"), 0o644); err != nil {
		t.Fatalf("write summary: %v", err)
	}
	if err := SaveRunMetadata(RunMetadataPath(runDir), &RunMetadata{
		Version:      1,
		Objective:    "compare external reference architectures",
		BaseRevision: strings.TrimSpace(gitOutput(t, repo, "rev-parse", "HEAD")),
	}); err != nil {
		t.Fatalf("write run metadata: %v", err)
	}
	seedRunCharterForTests(t, runDir, runName, repo)

	return runDir
}

func TestVerifyRecordsAcceptanceWhenRunChangedCode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "README.md", "demo", "base commit")
	baseRevision := strings.TrimSpace(gitOutput(t, repo, "rev-parse", "HEAD"))
	ensureSharedProofEvidence(t)

	runName := "verify-code-changed"
	runDir := goalx.RunDir(repo, runName)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".goalx"), 0o755); err != nil {
		t.Fatalf("mkdir project .goalx: %v", err)
	}

	snapshot := []byte(`name: verify-code-changed
mode: develop
objective: ship feature
acceptance:
  command: "printf 'e2e ok\n'"
`)
	if err := os.WriteFile(RunSpecPath(runDir), snapshot, 0o644); err != nil {
		t.Fatalf("write run snapshot: %v", err)
	}
	goal := []byte(`{
  "version": 1,
  "required": [
    {
      "id": "req-1",
      "text": "ship feature",
      "source": "user",
      "role": "outcome",
      "state": "claimed",
      "evidence_paths": ["/tmp/e2e.txt"],
      "note": "ready for verification"
    }
  ],
  "optional": []
}`)
	if err := os.WriteFile(GoalPath(runDir), goal, 0o644); err != nil {
		t.Fatalf("write goal state: %v", err)
	}
	if err := SaveRunMetadata(RunMetadataPath(runDir), &RunMetadata{
		Version:      1,
		Objective:    "ship feature",
		BaseRevision: baseRevision,
	}); err != nil {
		t.Fatalf("write run metadata: %v", err)
	}
	seedRunCharterForTests(t, runDir, runName, repo)

	writeAndCommit(t, repo, "feature.txt", "feature", "run change")

	if err := Verify(repo, []string{"--run", runName}); err != nil {
		t.Fatalf("Verify: %v", err)
	}

	stateData, err := os.ReadFile(filepath.Join(runDir, "acceptance.json"))
	if err != nil {
		t.Fatalf("read acceptance state: %v", err)
	}
	stateText := string(stateData)
	for _, want := range []string{
		`"command": "printf 'e2e ok\n'"`,
		`"exit_code": 0`,
	} {
		if !strings.Contains(stateText, want) {
			t.Fatalf("acceptance state missing %q:\n%s", want, stateText)
		}
	}
	if strings.Contains(stateText, `"status"`) {
		t.Fatalf("acceptance state must not contain derived status field:\n%s", stateText)
	}
}

func seedRunCharterForTests(t *testing.T, runDir, runName, projectRoot string) {
	t.Helper()

	meta, err := LoadRunMetadata(RunMetadataPath(runDir))
	if err != nil {
		t.Fatalf("LoadRunMetadata: %v", err)
	}
	if meta == nil {
		t.Fatal("run metadata missing")
	}
	if meta.ProtocolVersion == 0 {
		meta.ProtocolVersion = 2
	}
	if meta.ProjectRoot == "" {
		meta.ProjectRoot = projectRoot
	}
	if meta.RunID == "" {
		meta.RunID = newRunID()
	}
	if meta.RootRunID == "" {
		meta.RootRunID = meta.RunID
	}
	if meta.Epoch == 0 {
		meta.Epoch = 1
	}
	if err := SaveRunMetadata(RunMetadataPath(runDir), meta); err != nil {
		t.Fatalf("SaveRunMetadata normalize: %v", err)
	}
	charter, err := NewRunCharter(runDir, runName, "", meta)
	if err != nil {
		t.Fatalf("NewRunCharter: %v", err)
	}
	if err := SaveRunCharter(RunCharterPath(runDir), charter); err != nil {
		t.Fatalf("SaveRunCharter: %v", err)
	}
	digest, err := hashRunCharter(charter)
	if err != nil {
		t.Fatalf("hashRunCharter: %v", err)
	}
	meta.CharterID = charter.CharterID
	meta.CharterHash = digest
	if err := SaveRunMetadata(RunMetadataPath(runDir), meta); err != nil {
		t.Fatalf("SaveRunMetadata charter linkage: %v", err)
	}
}
