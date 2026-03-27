package cli

import (
	"testing"
)

func TestDurableSurfaceRegistryResolvesKnownSurfaces(t *testing.T) {
	runDir := t.TempDir()
	cases := []struct {
		name      string
		class     DurableSurfaceClass
		writeMode DurableSurfaceWriteMode
		strict    bool
		wantPath  string
	}{
		{name: "goal", class: DurableSurfaceClassStructuredState, writeMode: DurableSurfaceWriteModeReplace, strict: true, wantPath: GoalPath(runDir)},
		{name: "acceptance", class: DurableSurfaceClassStructuredState, writeMode: DurableSurfaceWriteModeReplace, strict: true, wantPath: AcceptanceStatePath(runDir)},
		{name: "coordination", class: DurableSurfaceClassStructuredState, writeMode: DurableSurfaceWriteModeReplace, strict: true, wantPath: CoordinationPath(runDir)},
		{name: "status", class: DurableSurfaceClassStructuredState, writeMode: DurableSurfaceWriteModeReplace, strict: true, wantPath: RunStatusPath(runDir)},
		{name: "goal-log", class: DurableSurfaceClassEventLog, writeMode: DurableSurfaceWriteModeAppend, strict: true, wantPath: GoalLogPath(runDir)},
		{name: "evolution", class: DurableSurfaceClassEventLog, writeMode: DurableSurfaceWriteModeAppend, strict: true, wantPath: EvolutionLogPath(runDir)},
		{name: "summary", class: DurableSurfaceClassArtifact, writeMode: DurableSurfaceWriteModeReplace, strict: false, wantPath: SummaryPath(runDir)},
		{name: "completion-proof", class: DurableSurfaceClassArtifact, writeMode: DurableSurfaceWriteModeReplace, strict: false, wantPath: CompletionStatePath(runDir)},
	}
	for _, tc := range cases {
		spec, err := LookupDurableSurface(tc.name)
		if err != nil {
			t.Fatalf("LookupDurableSurface(%q): %v", tc.name, err)
		}
		if spec.Class != tc.class {
			t.Fatalf("%s class = %s, want %s", tc.name, spec.Class, tc.class)
		}
		if spec.WriteMode != tc.writeMode {
			t.Fatalf("%s write mode = %s, want %s", tc.name, spec.WriteMode, tc.writeMode)
		}
		if spec.Strict != tc.strict {
			t.Fatalf("%s strict = %t, want %t", tc.name, spec.Strict, tc.strict)
		}
		if got := spec.Path(runDir); got != tc.wantPath {
			t.Fatalf("%s path = %s, want %s", tc.name, got, tc.wantPath)
		}
	}
}

func TestDurableSurfaceRegistryRejectsUnknownSurface(t *testing.T) {
	if _, err := LookupDurableSurface("mystery"); err == nil {
		t.Fatal("LookupDurableSurface should reject unknown surface")
	}
}
