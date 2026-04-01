package cli

import (
	"strings"
)

func RefreshCognitionStateForRun(runDir, runName string) error {
	if strings.TrimSpace(runDir) == "" {
		return nil
	}
	scopes := []CognitionScopeState{}

	runRootScope, err := DiscoverCognitionScope("run-root", RunWorktreePath(runDir))
	if err != nil {
		return err
	}
	scopes = append(scopes, runRootScope)

	cfg, err := LoadRunSpec(runDir)
	if err != nil {
		return err
	}
	sessionsState, err := EnsureSessionsRuntimeState(runDir)
	if err != nil {
		return err
	}
	indexes, err := existingSessionIndexes(runDir)
	if err != nil {
		return err
	}
	for _, idx := range indexes {
		sessionName := SessionName(idx)
		worktreePath := resolvedSessionWorktreePath(runDir, cfg.Name, sessionName, sessionsState)
		if strings.TrimSpace(worktreePath) == "" || worktreePath == RunWorktreePath(runDir) {
			continue
		}
		scope, err := DiscoverCognitionScope(sessionName, worktreePath)
		if err != nil {
			return err
		}
		scopes = append(scopes, scope)
	}

	return SaveCognitionState(CognitionStatePath(runDir), &CognitionState{
		Version: 1,
		Scopes:  scopes,
	})
}
