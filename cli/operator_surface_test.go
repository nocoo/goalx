package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOperatorSurfaceConsistency(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	repoRoot := filepath.Dir(wd)
	files := []string{
		filepath.Join(repoRoot, "README.md"),
		filepath.Join(repoRoot, "skill", "SKILL.md"),
		filepath.Join(repoRoot, "skill", "references", "advanced-control.md"),
		filepath.Join(repoRoot, "deploy", "README.md"),
	}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile %s: %v", path, err)
		}
		text := string(data)
		switch filepath.Base(path) {
		case "README.md":
			if !strings.Contains(text, "--guided") {
				t.Fatalf("%s missing --guided guidance", path)
			}
			if strings.Contains(path, filepath.Join("deploy", "README.md")) {
				if !strings.Contains(text, "goalx schema") {
					t.Fatalf("%s missing goalx schema guidance", path)
				}
			} else if !strings.Contains(text, "goalx schema") {
				t.Fatalf("%s missing goalx schema guidance", path)
			}
		case "SKILL.md":
			if !strings.Contains(text, "--guided") || !strings.Contains(text, "goalx schema") {
				t.Fatalf("%s missing guided/schema guidance", path)
			}
		case "advanced-control.md":
			if !strings.Contains(text, "--guided") {
				t.Fatalf("%s missing guided guidance", path)
			}
			if strings.Contains(text, "sidecar") {
				t.Fatalf("%s should not mention removed sidecar control plane", path)
			}
		}
	}
}
