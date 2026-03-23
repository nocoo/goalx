package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNextDefaultsToAutoFirst(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("mkdir project root: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Next(projectRoot, nil); err != nil {
			t.Fatalf("Next: %v", err)
		}
	})
	if !strings.Contains(out, "goalx auto \"your objective\"") {
		t.Fatalf("next output missing auto-first quickstart:\n%s", out)
	}
	if strings.Contains(out, "goalx init") || strings.Contains(out, "goalx start") {
		t.Fatalf("next output still promotes init/start:\n%s", out)
	}
}

func TestNextPromptsFocusForMultipleActiveRuns(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("mkdir project root: %v", err)
	}
	if err := SaveProjectRegistry(projectRoot, &ProjectRegistry{
		Version: 1,
		ActiveRuns: map[string]ProjectRunRef{
			"alpha": {Name: "alpha", State: "active"},
			"beta":  {Name: "beta", State: "active"},
		},
	}); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Next(projectRoot, nil); err != nil {
			t.Fatalf("Next: %v", err)
		}
	})
	if !strings.Contains(out, "goalx focus --run NAME") {
		t.Fatalf("next output missing focus guidance:\n%s", out)
	}
}

func TestNextSuggestsExplicitPhaseFromSavedResearchRun(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(projectRoot, ".goalx", "runs", "research-a"), 0o755); err != nil {
		t.Fatalf("mkdir saved run: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, ".goalx", "runs", "research-a", "run-spec.yaml"), []byte("name: research-a\nmode: research\nobjective: audit auth\n"), 0o644); err != nil {
		t.Fatalf("write run-spec: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Next(projectRoot, nil); err != nil {
			t.Fatalf("Next: %v", err)
		}
	})
	for _, want := range []string{
		"goalx debate --from research-a",
		"goalx implement --from research-a",
		"goalx explore --from research-a",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("next output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "goalx start") {
		t.Fatalf("next output should not suggest config-first start:\n%s", out)
	}
}
