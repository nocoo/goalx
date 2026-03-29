package cli

import (
	"strings"
	"testing"

	goalx "github.com/vonbai/goalx"
)

func TestEnsureAcceptanceStateBootstrapsChecksFromExplicitAcceptanceCommand(t *testing.T) {
	runDir := t.TempDir()
	cfg := &goalx.Config{
		Acceptance: goalx.AcceptanceConfig{
			Command: "printf 'gate ok\\n'",
		},
	}

	state, err := EnsureAcceptanceState(runDir, cfg, 3)
	if err != nil {
		t.Fatalf("EnsureAcceptanceState: %v", err)
	}
	if state.GoalVersion != 3 {
		t.Fatalf("goal_version = %d, want 3", state.GoalVersion)
	}
	if len(state.Checks) != 1 {
		t.Fatalf("checks = %#v, want one bootstrap check", state.Checks)
	}
	if state.Checks[0].Command != "printf 'gate ok\\n'" {
		t.Fatalf("bootstrap check command = %q, want explicit acceptance command", state.Checks[0].Command)
	}
	if state.Checks[0].State != acceptanceCheckStateActive {
		t.Fatalf("bootstrap check state = %q, want active", state.Checks[0].State)
	}
}

func TestSaveAcceptanceStateWithLockedObjectiveContractRejectsMissingAcceptanceCoverage(t *testing.T) {
	runDir := t.TempDir()
	if err := SaveObjectiveContract(ObjectiveContractPath(runDir), &ObjectiveContract{
		Version:       1,
		ObjectiveHash: "sha256:demo",
		State:         objectiveContractStateLocked,
		Clauses: []ObjectiveClause{
			{
				ID:               "ucl-verify",
				Text:             "live verification",
				Kind:             objectiveClauseKindVerification,
				SourceExcerpt:    "live verification",
				RequiredSurfaces: []ObjectiveRequiredSurface{objectiveRequiredSurfaceAcceptance},
			},
		},
	}); err != nil {
		t.Fatalf("SaveObjectiveContract: %v", err)
	}

	err := SaveAcceptanceState(AcceptanceStatePath(runDir), &AcceptanceState{
		Version:     2,
		GoalVersion: 1,
		Checks: []AcceptanceCheck{
			{ID: "chk-1", Label: "build", Command: "go test ./...", State: acceptanceCheckStateActive},
		},
	})
	if err == nil {
		t.Fatal("SaveAcceptanceState should reject missing acceptance clause coverage")
	}
	if !strings.Contains(err.Error(), "missing covers") {
		t.Fatalf("SaveAcceptanceState error = %v, want missing covers failure", err)
	}
}

func TestParseAcceptanceStateRejectsActiveCheckWithoutCommand(t *testing.T) {
	_, err := parseAcceptanceState([]byte(`{
  "version": 2,
  "goal_version": 1,
  "checks": [
    {
      "id": "chk-1",
      "label": "build",
      "state": "active"
    }
  ]
}`))
	if err == nil {
		t.Fatal("parseAcceptanceState should reject active checks without command")
	}
	if !strings.Contains(err.Error(), "command") {
		t.Fatalf("parseAcceptanceState error = %v, want command hint", err)
	}
}
