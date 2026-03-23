package cli

import (
	"os"
	"path/filepath"
	"testing"

	goalx "github.com/vonbai/goalx"
)

func TestStorageModelUsesUserScopedProjectRegistry(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("mkdir project root: %v", err)
	}

	want := filepath.Join(home, ".goalx", "runs", goalx.ProjectID(projectRoot), "registry.json")
	if got := ProjectRegistryPath(projectRoot); got != want {
		t.Fatalf("ProjectRegistryPath = %q, want %q", got, want)
	}
}

func TestStorageModelResolvesSavedRunFromUserScope(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("mkdir project root: %v", err)
	}

	runDir := SavedRunDir(projectRoot, "saved-research")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir saved run dir: %v", err)
	}

	got, err := ResolveSavedRunLocation(projectRoot, "saved-research")
	if err != nil {
		t.Fatalf("ResolveSavedRunLocation: %v", err)
	}
	if got.Dir != runDir || got.Legacy {
		t.Fatalf("ResolveSavedRunLocation = %#v, want user-scoped dir %q", got, runDir)
	}

	if _, err := os.Stat(filepath.Join(projectRoot, ".goalx", "runs")); !os.IsNotExist(err) {
		t.Fatalf("project scoped saved runs should not exist, stat err = %v", err)
	}
}

func TestStorageModelResolvesLegacySavedRunAsFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	projectRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("mkdir project root: %v", err)
	}

	legacyDir := LegacySavedRunDir(projectRoot, "saved-research")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatalf("mkdir legacy saved run dir: %v", err)
	}

	got, err := ResolveSavedRunLocation(projectRoot, "saved-research")
	if err != nil {
		t.Fatalf("ResolveSavedRunLocation: %v", err)
	}
	if got.Dir != legacyDir || !got.Legacy {
		t.Fatalf("ResolveSavedRunLocation = %#v, want legacy dir %q", got, legacyDir)
	}
}
