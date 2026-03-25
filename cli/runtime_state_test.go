package cli

import (
	"os"
	"strings"
	"testing"
)

func TestSaveRunRuntimeStateDoesNotPersistRecommendationField(t *testing.T) {
	runDir := t.TempDir()
	path := RunRuntimeStatePath(runDir)
	if err := SaveRunRuntimeState(path, &RunRuntimeState{
		Version:   1,
		Run:       "demo",
		Mode:      "develop",
		Active:    true,
		Phase:     "working",
		UpdatedAt: "2026-03-26T00:00:00Z",
	}); err != nil {
		t.Fatalf("SaveRunRuntimeState: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read run runtime state: %v", err)
	}
	text := string(data)
	if strings.Contains(text, `"recommendation"`) {
		t.Fatalf("run runtime state should not persist recommendation:\n%s", text)
	}
}
