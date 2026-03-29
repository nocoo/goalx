package cli

import (
	"os"
	"strings"
	"testing"
)

func TestLoadGoalStateRejectsUnknownFields(t *testing.T) {
	path := t.TempDir() + "/goal.json"
	payload := []byte(`{
  "version": 2,
  "required_items": [
    {
      "id": "req-1",
      "text": "ship feature",
      "state": "done"
    }
  ],
  "improvements": []
}`)
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("write goal state: %v", err)
	}

	_, err := LoadGoalState(path)
	if err == nil {
		t.Fatal("expected LoadGoalState to fail")
	}
	for _, want := range []string{"parse goal state", "unknown field", "goalx schema goal"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("LoadGoalState error = %v, want %q", err, want)
		}
	}
}

func TestLoadGoalStateRejectsEmptyFile(t *testing.T) {
	path := t.TempDir() + "/goal.json"
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("write empty goal state: %v", err)
	}

	_, err := LoadGoalState(path)
	if err == nil {
		t.Fatal("expected LoadGoalState to fail")
	}
	if !strings.Contains(err.Error(), "goal state is empty") {
		t.Fatalf("LoadGoalState error = %v, want empty-file error", err)
	}
}

func TestLoadGoalStateRejectsInvalidItemState(t *testing.T) {
	path := t.TempDir() + "/goal.json"
	payload := []byte(`{
  "version": 1,
  "required": [
    {
      "id": "req-1",
      "text": "ship feature",
      "source": "user",
      "role": "outcome",
      "state": "done"
    }
  ],
  "optional": []
}`)
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("write goal state: %v", err)
	}

	_, err := LoadGoalState(path)
	if err == nil {
		t.Fatal("expected LoadGoalState to fail")
	}
	if !strings.Contains(err.Error(), `invalid goal item state "done"`) {
		t.Fatalf("LoadGoalState error = %v, want invalid-state error", err)
	}
}

func TestLoadGoalStateRejectsMissingItemSource(t *testing.T) {
	path := t.TempDir() + "/goal.json"
	payload := []byte(`{
  "version": 1,
  "required": [
    {
      "id": "req-1",
      "text": "ship feature",
      "role": "outcome",
      "state": "open"
    }
  ],
  "optional": []
}`)
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("write goal state: %v", err)
	}

	_, err := LoadGoalState(path)
	if err == nil {
		t.Fatal("expected LoadGoalState to fail")
	}
	for _, want := range []string{"goal item source is required", "goalx schema goal"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("LoadGoalState error = %v, want %q", err, want)
		}
	}
}

func TestLoadGoalStateRejectsMissingItemRole(t *testing.T) {
	path := t.TempDir() + "/goal.json"
	payload := []byte(`{
  "version": 1,
  "required": [
    {
      "id": "req-1",
      "text": "ship feature",
      "source": "user",
      "state": "open"
    }
  ],
  "optional": []
}`)
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("write goal state: %v", err)
	}

	_, err := LoadGoalState(path)
	if err == nil {
		t.Fatal("expected LoadGoalState to fail")
	}
	for _, want := range []string{"goal item role is required", "goalx schema goal"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("LoadGoalState error = %v, want %q", err, want)
		}
	}
}

func TestLoadGoalStateRejectsInvalidItemRole(t *testing.T) {
	path := t.TempDir() + "/goal.json"
	payload := []byte(`{
  "version": 1,
  "required": [
    {
      "id": "req-1",
      "text": "ship feature",
      "source": "user",
      "role": "task",
      "state": "open"
    }
  ],
  "optional": []
}`)
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("write goal state: %v", err)
	}

	_, err := LoadGoalState(path)
	if err == nil {
		t.Fatal("expected LoadGoalState to fail")
	}
	if !strings.Contains(err.Error(), `invalid goal item role "task"`) {
		t.Fatalf("LoadGoalState error = %v, want invalid-role error", err)
	}
}

func TestEnsureGoalStateDoesNotRewriteExistingGoal(t *testing.T) {
	runDir := t.TempDir()
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

	state, err := EnsureGoalState(runDir)
	if err != nil {
		t.Fatalf("EnsureGoalState: %v", err)
	}
	if state == nil || len(state.Required) != 1 {
		t.Fatalf("EnsureGoalState returned %#v, want one required item", state)
	}

	assertFileUnchanged(t, GoalPath(runDir), goalBefore)
}

func TestSaveGoalStateWritesExplicitSourceAndRole(t *testing.T) {
	path := t.TempDir() + "/goal.json"
	state := &GoalState{
		Version: 1,
		Required: []GoalItem{
			{
				ID:     "req-1",
				Text:   "ship feature",
				Source: "user",
				Role:   "outcome",
				Covers: []string{"ucl-1"},
				State:  "open",
			},
		},
		Optional: []GoalItem{
			{
				ID:     "opt-1",
				Text:   "improve latency",
				Source: "master",
				Role:   "guardrail",
				Covers: []string{"ucl-2"},
				State:  "open",
			},
		},
	}

	if err := SaveGoalState(path, state); err != nil {
		t.Fatalf("SaveGoalState: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read goal state: %v", err)
	}
	for _, want := range []string{`"source": "user"`, `"role": "outcome"`, `"source": "master"`, `"role": "guardrail"`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("saved goal state missing %q:\n%s", want, string(data))
		}
	}
}

func TestSaveGoalStateWithLockedObjectiveContractRequiresRequiredCovers(t *testing.T) {
	runDir := t.TempDir()
	contract := &ObjectiveContract{
		Version:       1,
		ObjectiveHash: "sha256:demo",
		State:         objectiveContractStateLocked,
		Clauses: []ObjectiveClause{
			{
				ID:               "ucl-1",
				Text:             "ship feature",
				Kind:             objectiveClauseKindDelivery,
				SourceExcerpt:    "ship feature",
				RequiredSurfaces: []ObjectiveRequiredSurface{objectiveRequiredSurfaceGoal},
			},
			{
				ID:               "ucl-2",
				Text:             "second feature",
				Kind:             objectiveClauseKindDelivery,
				SourceExcerpt:    "second feature",
				RequiredSurfaces: []ObjectiveRequiredSurface{objectiveRequiredSurfaceGoal},
			},
		},
	}
	if err := SaveObjectiveContract(ObjectiveContractPath(runDir), contract); err != nil {
		t.Fatalf("SaveObjectiveContract: %v", err)
	}
	err := SaveGoalState(GoalPath(runDir), &GoalState{
		Version: 1,
		Required: []GoalItem{
			{ID: "req-1", Text: "ship feature", Source: goalItemSourceUser, Role: goalItemRoleOutcome, State: goalItemStateOpen},
		},
	})
	if err == nil {
		t.Fatal("SaveGoalState should reject missing covers under locked objective contract")
	}
	if !strings.Contains(err.Error(), "missing covers") {
		t.Fatalf("SaveGoalState error = %v, want missing covers", err)
	}
}

func TestSaveGoalStateWithLockedObjectiveContractRejectsUnknownClauseReference(t *testing.T) {
	runDir := t.TempDir()
	contract := &ObjectiveContract{
		Version:       1,
		ObjectiveHash: "sha256:demo",
		State:         objectiveContractStateLocked,
		Clauses: []ObjectiveClause{
			{
				ID:               "ucl-1",
				Text:             "ship feature",
				Kind:             objectiveClauseKindDelivery,
				SourceExcerpt:    "ship feature",
				RequiredSurfaces: []ObjectiveRequiredSurface{objectiveRequiredSurfaceGoal},
			},
			{
				ID:               "ucl-2",
				Text:             "second feature",
				Kind:             objectiveClauseKindDelivery,
				SourceExcerpt:    "second feature",
				RequiredSurfaces: []ObjectiveRequiredSurface{objectiveRequiredSurfaceGoal},
			},
		},
	}
	if err := SaveObjectiveContract(ObjectiveContractPath(runDir), contract); err != nil {
		t.Fatalf("SaveObjectiveContract: %v", err)
	}
	err := SaveGoalState(GoalPath(runDir), &GoalState{
		Version: 1,
		Required: []GoalItem{
			{ID: "req-1", Text: "ship feature", Source: goalItemSourceUser, Role: goalItemRoleOutcome, Covers: []string{"ucl-missing"}, State: goalItemStateOpen},
		},
	})
	if err == nil {
		t.Fatal("SaveGoalState should reject unknown clause coverage")
	}
	if !strings.Contains(err.Error(), "unknown objective clause") {
		t.Fatalf("SaveGoalState error = %v, want unknown clause", err)
	}
}

func TestSaveGoalStateWithLockedObjectiveContractRejectsOptionalOnlyCoverage(t *testing.T) {
	runDir := t.TempDir()
	contract := &ObjectiveContract{
		Version:       1,
		ObjectiveHash: "sha256:demo",
		State:         objectiveContractStateLocked,
		Clauses: []ObjectiveClause{
			{
				ID:               "ucl-1",
				Text:             "ship feature",
				Kind:             objectiveClauseKindDelivery,
				SourceExcerpt:    "ship feature",
				RequiredSurfaces: []ObjectiveRequiredSurface{objectiveRequiredSurfaceGoal},
			},
			{
				ID:               "ucl-2",
				Text:             "second feature",
				Kind:             objectiveClauseKindDelivery,
				SourceExcerpt:    "second feature",
				RequiredSurfaces: []ObjectiveRequiredSurface{objectiveRequiredSurfaceGoal},
			},
		},
	}
	if err := SaveObjectiveContract(ObjectiveContractPath(runDir), contract); err != nil {
		t.Fatalf("SaveObjectiveContract: %v", err)
	}
	err := SaveGoalState(GoalPath(runDir), &GoalState{
		Version: 1,
		Required: []GoalItem{
			{ID: "req-1", Text: "second feature", Source: goalItemSourceUser, Role: goalItemRoleOutcome, Covers: []string{"ucl-2"}, State: goalItemStateOpen},
		},
		Optional: []GoalItem{
			{ID: "opt-1", Text: "ship feature", Source: goalItemSourceMaster, Role: goalItemRoleOutcome, Covers: []string{"ucl-1"}, State: goalItemStateOpen},
		},
	})
	if err == nil {
		t.Fatal("SaveGoalState should reject optional-only contract coverage")
	}
	if !strings.Contains(err.Error(), "requires required goal coverage") {
		t.Fatalf("SaveGoalState error = %v, want required coverage failure", err)
	}
}

func TestValidateGoalStateForVerificationRequiresApprovalRefForWaivedItems(t *testing.T) {
	state := &GoalState{
		Version: 1,
		Required: []GoalItem{
			{
				ID:     "req-1",
				Text:   "ship feature",
				Source: goalItemSourceUser,
				Role:   goalItemRoleOutcome,
				Covers: []string{"ucl-1"},
				State:  goalItemStateWaived,
			},
		},
	}

	_, err := ValidateGoalStateForVerification(state)
	if err == nil {
		t.Fatal("ValidateGoalStateForVerification should reject waived items without approval_ref")
	}
	if !strings.Contains(err.Error(), "approval_ref") {
		t.Fatalf("ValidateGoalStateForVerification error = %v, want approval_ref hint", err)
	}
}
