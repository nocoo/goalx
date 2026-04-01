package cli

import (
	"strings"
)

func RefreshCognitionStateForRun(runDir, runName string) error {
	if strings.TrimSpace(runDir) == "" {
		return nil
	}
	scopes := []CognitionScopeState{}

	runRootScope, err := RefreshCognitionScope("run-root", RunWorktreePath(runDir))
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
		scope, err := RefreshCognitionScope(sessionName, worktreePath)
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

func RefreshCognitionScope(scopeName, scopePath string) (CognitionScopeState, error) {
	scopePath = strings.TrimSpace(scopePath)
	if scopePath == "" {
		return CognitionScopeState{}, nil
	}
	providers := []CognitionProvider{
		repoNativeCognitionProvider{},
		gitNexusCognitionProvider{},
	}
	scope := CognitionScopeState{
		Scope:        strings.TrimSpace(scopeName),
		WorktreePath: scopePath,
		Providers:    []CognitionProviderState{},
	}
	for _, provider := range providers {
		state, err := provider.Refresh(scopePath)
		if err != nil {
			return CognitionScopeState{}, err
		}
		scope.Providers = append(scope.Providers, state)
	}
	return scope, nil
}
