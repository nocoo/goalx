package cli

import (
	"os"
	"path/filepath"
	"testing"

	goalx "github.com/vonbai/goalx"
)

func TestInferHarnessFromMarkerFiles(t *testing.T) {
	t.Run("go", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}

		if got := InferHarness(root); got != "go build ./... && go test ./... -count=1 && go vet ./..." {
			t.Fatalf("InferHarness(go) = %q", got)
		}
	})

	t.Run("node without test script", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"demo","scripts":{"build":"tsc"}}`), 0o644); err != nil {
			t.Fatalf("write package.json: %v", err)
		}

		if got := InferHarness(root); got != "npx tsc --noEmit 2>/dev/null || true" {
			t.Fatalf("InferHarness(package.json without test) = %q", got)
		}
	})
}

func TestInferTargetPrefersSourceDirectories(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}

	got := InferTarget(root)
	if len(got) != 1 || got[0] != "src/" {
		t.Fatalf("InferTarget(src) = %#v", got)
	}
}

func TestImplementUsesInferredHarnessAndTargetWhenConfigIsEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".goalx"), 0o755); err != nil {
		t.Fatalf("mkdir project .goalx: %v", err)
	}
	if err := os.Mkdir(filepath.Join(projectRoot, "src"), 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "package.json"), []byte(`{"name":"demo","scripts":{"test":"vitest"}}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	writeSavedRunFixture(t, projectRoot, "research-a", goalx.Config{
		Name:      "research-a",
		Mode:      goalx.ModeResearch,
		Objective: "audit auth flow",
		Preset:    "codex",
		Parallel:  2,
	}, map[string]string{
		"summary.md":          "# summary\n",
		"session-1-report.md": "# report\n",
	})

	if err := Implement(projectRoot, nil, nil); err != nil {
		t.Fatalf("Implement: %v", err)
	}

	cfg, err := goalx.LoadYAML[goalx.Config](filepath.Join(projectRoot, ".goalx", "goalx.yaml"))
	if err != nil {
		t.Fatalf("load goalx.yaml: %v", err)
	}
	if cfg.Harness.Command != "npm test" {
		t.Fatalf("harness.command = %q, want npm test", cfg.Harness.Command)
	}
	if len(cfg.Target.Files) != 1 || cfg.Target.Files[0] != "src/" {
		t.Fatalf("target.files = %#v, want [src/]", cfg.Target.Files)
	}
}
