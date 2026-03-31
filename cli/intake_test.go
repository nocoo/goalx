package cli

import (
	"os"
	"strings"
	"testing"

	goalx "github.com/vonbai/goalx"
)

func TestBuildGuidedIntakeIncludesIntentHints(t *testing.T) {
	cfg := &goalx.Config{
		Objective: "ship auth boundary",
		Context: goalx.ContextConfig{
			Files: []string{"README.md"},
			Refs:  []string{"ref:ticket-123"},
		},
		Target: goalx.TargetConfig{
			Readonly: []string{"."},
		},
	}
	meta := &RunMetadata{Intent: runIntentExplore, GuidedLaunch: true}
	intake := BuildGuidedIntake(cfg, meta)
	if intake == nil {
		t.Fatal("BuildGuidedIntake returned nil")
	}
	if !intake.Guided {
		t.Fatal("intake.Guided = false, want true")
	}
	if intake.Intent != runIntentExplore {
		t.Fatalf("intent = %q, want %q", intake.Intent, runIntentExplore)
	}
	if intake.Objective != "ship auth boundary" {
		t.Fatalf("objective = %q, want ship auth boundary", intake.Objective)
	}
	for _, want := range []string{
		"expand_evidence_before_implementation",
		"preserve_declared_readonly_boundary",
		"declared_context_is_part_of_initial_success_input",
	} {
		if !containsString(intake.WorkflowHints, want) && !containsString(intake.AntiGoals, want) && !containsString(intake.SuccessHints, want) {
			t.Fatalf("guided intake missing %q: %+v", want, intake)
		}
	}
}

func TestSaveGuidedIntakeRoundTrip(t *testing.T) {
	path := GuidedIntakePath(t.TempDir())
	intake := &GuidedIntake{
		Version:       1,
		Guided:        true,
		Objective:     "ship it",
		Intent:        runIntentDeliver,
		SuccessHints:  []string{"ship_verified_outcome"},
		AntiGoals:     []string{"do_not_stop_at_correctness_only"},
		WorkflowHints: []string{"dispatch_before_self_implementation"},
	}
	if err := SaveGuidedIntake(path, intake); err != nil {
		t.Fatalf("SaveGuidedIntake: %v", err)
	}
	loaded, err := LoadGuidedIntake(path)
	if err != nil {
		t.Fatalf("LoadGuidedIntake: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadGuidedIntake returned nil")
	}
	if !loaded.Guided || loaded.Objective != "ship it" {
		t.Fatalf("loaded intake = %+v, want guided ship it", loaded)
	}
}

func TestBuildBootstrapCompilerSourcesIncludesGuidedIntakeRunContextRef(t *testing.T) {
	repo := initGitRepo(t)
	writeAndCommit(t, repo, "README.md", "demo", "base commit")
	runDir := t.TempDir()
	if err := os.MkdirAll(ControlDir(runDir), 0o755); err != nil {
		t.Fatalf("mkdir control dir: %v", err)
	}
	if err := SaveGuidedIntake(GuidedIntakePath(runDir), &GuidedIntake{
		Version: 1,
		Guided:  true,
		Intent:  runIntentDeliver,
	}); err != nil {
		t.Fatalf("SaveGuidedIntake: %v", err)
	}

	sources, err := buildBootstrapCompilerSources(repo, runDir)
	if err != nil {
		t.Fatalf("buildBootstrapCompilerSources: %v", err)
	}
	if sources == nil {
		t.Fatal("sources is nil")
	}
	found := false
	for _, slot := range sources.SourceSlots {
		if slot.Slot != CompilerInputSlotRunContext {
			continue
		}
		for _, ref := range slot.Refs {
			if strings.Contains(ref, "guided-intake.json") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("source slots = %+v, want guided-intake run-context ref", sources.SourceSlots)
	}
}
