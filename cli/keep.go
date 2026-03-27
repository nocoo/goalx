package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Keep merges or preserves a specific session from a run.
func Keep(projectRoot string, args []string) error {
	if printUsageIfHelp(args, "usage: goalx keep [--run NAME] [session-name]") {
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

	branch := fmt.Sprintf("goalx/%s/%d", rc.Config.Name, idx)
	wtPath := WorktreePath(rc.RunDir, rc.Config.Name, idx)
	if info, err := os.Stat(wtPath); err == nil && info.IsDir() {
		if err := MergeWorktree(runWT, branch); err != nil {
			return fmt.Errorf("merge %s: %w", branch, err)
		}
		fmt.Printf("Merged %s into run worktree.\n", branch)
	} else {
		fmt.Printf("Session %s has no worktree (changes already in run worktree).\n", sessionName)
	}

	// Write selection.json
	selection := map[string]string{
		"kept":   sessionName,
		"branch": branch,
	}
	data, _ := json.MarshalIndent(selection, "", "  ")
	selPath := filepath.Join(rc.RunDir, "selection.json")
	if err := os.WriteFile(selPath, data, 0644); err != nil {
		return fmt.Errorf("write selection.json: %w", err)
	}
	fmt.Printf("Selection recorded: %s\n", selPath)
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
