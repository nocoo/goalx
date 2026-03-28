package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const keepHelpText = `usage: goalx keep [--run NAME] [session-name]

Without a session name:
- merge the run worktree branch into the source root
- require source-root HEAD to still descend from the run base revision
- skip the merge when the run tree is already integrated

With session-N:
- merge that develop session branch into the run worktree
- only committed session branch history is merged
- dirty session worktrees must be committed first
- require a recorded session parent/base ref to define the merge boundary
- if conflicts or partial adoption are required, inspect the session worktree and merge manually
- record selection.json so the kept session is durable
- this does not merge into the source root yet`

// Keep merges or preserves a specific session from a run.
func Keep(projectRoot string, args []string) error {
	if printUsageIfHelp(args, keepHelpText) {
		return nil
	}
	runName, rest, err := extractRunFlag(args)
	if err != nil {
		return err
	}
	if len(rest) > 1 {
		return fmt.Errorf("usage: goalx keep [--run NAME] [session-name]")
	}

	rc, err := ResolveRun(projectRoot, runName)
	if err != nil {
		return err
	}

	if len(rest) == 0 {
		meta, err := LoadRunMetadata(RunMetadataPath(rc.RunDir))
		if err != nil {
			return fmt.Errorf("load run metadata: %w", err)
		}
		if meta != nil && strings.TrimSpace(meta.BaseRevision) != "" {
			ok, err := gitIsAncestor(rc.ProjectRoot, meta.BaseRevision, "HEAD")
			if err != nil {
				return fmt.Errorf("check run base revision ancestry: %w", err)
			}
			if !ok {
				return fmt.Errorf("source root HEAD does not descend from run base revision %s; switch back to the run base branch or merge manually", meta.BaseRevision)
			}
		}
		runBranch := fmt.Sprintf("goalx/%s/root", rc.Config.Name)
		integrated, err := gitTreesEqual(rc.ProjectRoot, "HEAD", runBranch)
		if err != nil {
			return fmt.Errorf("compare %s with source root: %w", runBranch, err)
		}
		if integrated {
			fmt.Printf("Run worktree already integrated into source root.\n")
			return nil
		}
		if err := MergeWorktree(rc.ProjectRoot, runBranch); err != nil {
			return fmt.Errorf("merge %s: %w", runBranch, err)
		}
		fmt.Printf("Merged run worktree into source root.\n")
		return nil
	}

	runWT := RunWorktreePath(rc.RunDir)
	sessionName := rest[0]
	idx, err := parseSessionIndex(sessionName)
	if err != nil {
		return err
	}
	ok, err := hasSessionIndex(rc.RunDir, idx)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("session %q out of range for run %q", sessionName, rc.Name)
	}

	sessionState, err := EnsureSessionsRuntimeState(rc.RunDir)
	if err != nil {
		return fmt.Errorf("load session runtime state: %w", err)
	}
	branch := resolvedSessionBranch(rc.RunDir, rc.Config.Name, sessionName, sessionState)
	if branch == "" {
		branch = fmt.Sprintf("goalx/%s/%d", rc.Config.Name, idx)
	}
	wtPath := resolvedSessionWorktreePath(rc.RunDir, rc.Config.Name, sessionName, sessionState)
	if wtPath == "" {
		wtPath = WorktreePath(rc.RunDir, rc.Config.Name, idx)
	}
	if info, err := os.Stat(wtPath); err == nil && info.IsDir() {
		selectionBaseSelector, selectionBaseBranch, err := requireSessionKeepBoundary(rc.RunDir, rc.Config.Name, sessionName, wtPath, branch)
		if err != nil {
			return err
		}
		integrated, err := gitTreesEqual(runWT, "HEAD", branch)
		if err != nil {
			return fmt.Errorf("compare %s with run worktree: %w", branch, err)
		}
		if integrated {
			fmt.Printf("Session %s already integrated into run worktree.\n", sessionName)
			return writeKeepSelection(rc.RunDir, sessionName, branch, selectionBaseSelector, selectionBaseBranch)
		}
		if err := MergeWorktree(runWT, branch); err != nil {
			return fmt.Errorf("merge %s: %w", branch, err)
		}
		fmt.Printf("Merged %s into run worktree.\n", branch)
		if err := writeKeepSelection(rc.RunDir, sessionName, branch, selectionBaseSelector, selectionBaseBranch); err != nil {
			return err
		}
	} else {
		fmt.Printf("Session %s has no worktree (changes already in run worktree).\n", sessionName)
		if err := writeKeepSelection(rc.RunDir, sessionName, branch, "", ""); err != nil {
			return err
		}
	}
	return nil
}

func parseSessionIndex(name string) (int, error) {
	// Expect "session-N"
	if len(name) > 8 && name[:8] == "session-" {
		n, err := strconv.Atoi(name[8:])
		if err == nil && n > 0 {
			return n, nil
		}
	}
	return 0, fmt.Errorf("invalid session name %q (expected session-N)", name)
}

func requireSessionKeepBoundary(runDir, runName, sessionName, worktreePath, branch string) (string, string, error) {
	identity, err := RequireSessionIdentity(runDir, sessionName)
	if err != nil {
		return "", "", fmt.Errorf("load %s identity: %w", sessionName, err)
	}
	baseSelector := strings.TrimSpace(identity.BaseBranchSelector)
	baseBranch := strings.TrimSpace(identity.BaseBranch)
	if baseBranch == "" {
		return "", "", fmt.Errorf("session %s has no recorded parent/base ref; this session boundary is not mergeable through goalx keep", sessionName)
	}
	dirtyPaths, err := dirtyWorktreePaths(worktreePath)
	if err != nil {
		return "", "", err
	}
	if len(dirtyPaths) > 0 {
		return "", "", fmt.Errorf("session %s has uncommitted changes (%s); commit them before goalx keep so the merge boundary is sealed", sessionName, summarizeDirtyPaths(dirtyPaths))
	}
	lineage, err := snapshotWorktreeLineage(worktreePath, baseSelector, baseBranch, "")
	if err != nil {
		return "", "", err
	}
	if lineage == nil || lineage.AheadCommits == 0 {
		return "", "", fmt.Errorf("session %s has no committed branch changes relative to recorded base %s; goalx keep only merges committed session history", sessionName, baseBranch)
	}
	sessionState, err := EnsureSessionsRuntimeState(runDir)
	if err == nil {
		if resolved := resolvedSessionBranch(runDir, runName, sessionName, sessionState); resolved != "" && resolved != branch {
			return "", "", fmt.Errorf("session %s branch mismatch: runtime=%s keep=%s", sessionName, resolved, branch)
		}
	}
	return baseSelector, baseBranch, nil
}

func dirtyWorktreePaths(worktreePath string) ([]string, error) {
	statusOut, err := exec.Command("git", "-C", worktreePath, "status", "--porcelain").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git status in %s: %w: %s", worktreePath, err, statusOut)
	}
	dirtyPaths := make([]string, 0)
	for _, line := range strings.Split(string(statusOut), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		path := parsePorcelainPath(line)
		if isAllowedLocalConfigPath(path) {
			continue
		}
		dirtyPaths = append(dirtyPaths, path)
	}
	return dirtyPaths, nil
}

func writeKeepSelection(runDir, sessionName, branch, baseSelector, baseBranch string) error {
	selection := map[string]string{
		"kept":   sessionName,
		"branch": branch,
	}
	if strings.TrimSpace(baseSelector) != "" {
		selection["base_selector"] = strings.TrimSpace(baseSelector)
	}
	if strings.TrimSpace(baseBranch) != "" {
		selection["base_branch"] = strings.TrimSpace(baseBranch)
	}
	data, err := json.MarshalIndent(selection, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal selection.json: %w", err)
	}
	selPath := filepath.Join(runDir, "selection.json")
	if err := os.WriteFile(selPath, data, 0644); err != nil {
		return fmt.Errorf("write selection.json: %w", err)
	}
	fmt.Printf("Selection recorded: %s\n", selPath)
	return nil
}
