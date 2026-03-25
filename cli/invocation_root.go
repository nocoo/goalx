package cli

import (
	"os"
	"path/filepath"
	"strings"
)

// CanonicalProjectRoot maps a GoalX run worktree cwd back to the source project
// root recorded in durable run metadata. Non-run paths are returned unchanged.
func CanonicalProjectRoot(projectRoot string) string {
	abs, err := filepath.Abs(strings.TrimSpace(projectRoot))
	if err != nil || abs == "" {
		return projectRoot
	}
	runDir, ok := enclosingRunDirFromWorktree(abs)
	if !ok {
		return abs
	}
	if meta, err := LoadRunMetadata(RunMetadataPath(runDir)); err == nil && meta != nil && strings.TrimSpace(meta.ProjectRoot) != "" {
		return filepath.Clean(meta.ProjectRoot)
	}
	if identity, err := LoadControlRunIdentity(ControlRunIdentityPath(runDir)); err == nil && identity != nil && strings.TrimSpace(identity.ProjectRoot) != "" {
		return filepath.Clean(identity.ProjectRoot)
	}
	return abs
}

func enclosingRunDirFromWorktree(path string) (string, bool) {
	for current := filepath.Clean(path); ; current = filepath.Dir(current) {
		parent := filepath.Dir(current)
		if filepath.Base(parent) == "worktrees" {
			runDir := filepath.Dir(parent)
			if strings.TrimSpace(runDir) != "" {
				if _, err := os.Stat(RunMetadataPath(runDir)); err == nil {
					return runDir, true
				}
				if _, err := os.Stat(RunSpecPath(runDir)); err == nil {
					return runDir, true
				}
			}
			return "", false
		}
		next := filepath.Dir(current)
		if next == current {
			return "", false
		}
	}
}
