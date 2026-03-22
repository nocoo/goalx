package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func existingSessionIndexes(runDir string) ([]int, error) {
	state, err := EnsureSessionsRuntimeState(runDir)
	if err != nil {
		return nil, fmt.Errorf("read session runtime state: %w", err)
	}

	indexes := make([]int, 0, len(state.Sessions))
	for name := range state.Sessions {
		idx, err := parseSessionIndex(name)
		if err == nil && idx > 0 {
			indexes = append(indexes, idx)
		}
	}
	indexes = append(indexes, discoverSessionIndexesFromFS(runDir)...)
	slices.Sort(indexes)
	indexes = slices.Compact(indexes)
	return indexes, nil
}

func nextSessionIndex(runDir string) (int, error) {
	indexes, err := existingSessionIndexes(runDir)
	if err != nil {
		return 0, err
	}
	if len(indexes) == 0 {
		return 1, nil
	}
	return indexes[len(indexes)-1] + 1, nil
}

func hasSessionIndex(runDir string, idx int) (bool, error) {
	indexes, err := existingSessionIndexes(runDir)
	if err != nil {
		return false, err
	}
	return slices.Contains(indexes, idx), nil
}

func discoverSessionIndexesFromFS(runDir string) []int {
	var indexes []int
	appendFromDir := func(dir string, transform func(string) string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, entry := range entries {
			name := transform(entry.Name())
			idx, err := parseSessionIndex(name)
			if err == nil && idx > 0 {
				indexes = append(indexes, idx)
			}
		}
	}

	appendFromDir(filepath.Join(runDir, "journals"), func(name string) string {
		return strings.TrimSuffix(name, ".jsonl")
	})
	appendFromDir(filepath.Join(runDir, "guidance"), func(name string) string {
		return strings.TrimSuffix(name, ".md")
	})
	appendFromDir(ControlDir(runDir), func(name string) string {
		return strings.TrimSuffix(name, "-guidance.json")
	})
	appendFromDir(filepath.Join(runDir, "worktrees"), func(name string) string {
		if i := strings.LastIndex(name, "-"); i >= 0 {
			return "session-" + name[i+1:]
		}
		return name
	})
	return indexes
}
