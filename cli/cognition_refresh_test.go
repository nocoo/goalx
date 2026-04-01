package cli

import (
	"fmt"
	"testing"
)

func TestRefreshCognitionStateForRunRefreshesRunRootAndSessionScopes(t *testing.T) {
	prevLookPath := lookPathFunc
	prevStatus := gitNexusStatusFunc
	prevAnalyze := gitNexusAnalyzeFunc
	t.Cleanup(func() {
		lookPathFunc = prevLookPath
		gitNexusStatusFunc = prevStatus
		gitNexusAnalyzeFunc = prevAnalyze
	})
	lookPathFunc = func(name string) (string, error) {
		switch name {
		case "git", "gitnexus":
			return "/usr/bin/" + name, nil
		default:
			return "", fmt.Errorf("missing")
		}
	}

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "base.txt", "base", "base commit")
	runName, runDir := writeLifecycleRunFixture(t, repo)

	statusCalls := map[string]int{}
	gitNexusStatusFunc = func(invocationKind, scopePath string) (string, error) {
		statusCalls[scopePath]++
		if statusCalls[scopePath] == 1 {
			return "Repository not indexed.\nRun: gitnexus analyze\n", nil
		}
		return fmt.Sprintf("Repository: %s\nIndexed: 4/1/2026, 12:00:00 AM\nIndexed commit: abc1234\nCurrent commit: abc1234\nStatus: ✅ up-to-date\n", scopePath), nil
	}

	analyzeCalls := map[string]int{}
	gitNexusAnalyzeFunc = func(invocationKind, scopePath string) error {
		analyzeCalls[scopePath]++
		return nil
	}

	if err := RefreshCognitionStateForRun(runDir, runName); err != nil {
		t.Fatalf("RefreshCognitionStateForRun: %v", err)
	}

	state, err := LoadCognitionState(CognitionStatePath(runDir))
	if err != nil {
		t.Fatalf("LoadCognitionState: %v", err)
	}
	if state == nil || len(state.Scopes) < 2 {
		t.Fatalf("cognition state = %#v, want run-root and session scope", state)
	}

	runRootFound := false
	sessionFound := false
	for _, scope := range state.Scopes {
		for _, provider := range scope.Providers {
			if provider.Name != "gitnexus" {
				continue
			}
			if provider.IndexState != "fresh" {
				t.Fatalf("scope %s gitnexus provider = %+v, want fresh", scope.Scope, provider)
			}
			switch scope.Scope {
			case "run-root":
				runRootFound = true
			case "session-1":
				sessionFound = true
			}
		}
	}
	if !runRootFound || !sessionFound {
		t.Fatalf("cognition scopes = %#v, want gitnexus for run-root and session-1", state.Scopes)
	}
	if analyzeCalls[RunWorktreePath(runDir)] != 1 {
		t.Fatalf("run-root analyze calls = %d, want 1", analyzeCalls[RunWorktreePath(runDir)])
	}
	if analyzeCalls[WorktreePath(runDir, runName, 1)] != 1 {
		t.Fatalf("session-1 analyze calls = %d, want 1", analyzeCalls[WorktreePath(runDir, runName, 1)])
	}
}
