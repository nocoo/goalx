package cli

import (
	"os"
	"path/filepath"
	"testing"

	goalx "github.com/vonbai/goalx"
	"gopkg.in/yaml.v3"
)

func writeSavedRunFixture(t *testing.T, projectRoot, runName string, cfg goalx.Config, files map[string]string) {
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
}

func writeRootConfigFixture(t *testing.T, projectRoot string, cfg goalx.Config) {
	t.Helper()

	goalxDir := filepath.Join(projectRoot, ".goalx")
	if err := os.MkdirAll(goalxDir, 0o755); err != nil {
		t.Fatalf("mkdir goalx dir: %v", err)
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		t.Fatalf("marshal root config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(goalxDir, "goalx.yaml"), data, 0o644); err != nil {
		t.Fatalf("write root goalx.yaml: %v", err)
	}
}
