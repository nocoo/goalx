package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CreateWorktree creates a new git worktree with a new branch.
// Cleans up stale branch collisions from failed previous runs, but refuses to
// delete branches that are still checked out in another worktree.
func CreateWorktree(projectRoot, worktreePath, branch string) error {
	exec.Command("git", "-C", projectRoot, "worktree", "prune").Run()
	exists, err := branchExists(projectRoot, branch)
	if err != nil {
		return err
	}
	if exists {
		inUse, err := branchCheckedOutInAnyWorktree(projectRoot, branch)
		if err != nil {
			return err
		}
		if inUse {
			return fmt.Errorf("branch %s is already checked out in another worktree", branch)
		}
		if err := DeleteBranch(projectRoot, branch); err != nil {
			return fmt.Errorf("delete stale branch %s: %w", branch, err)
		}
	}
	cmd := exec.Command("git", "-C", projectRoot, "worktree", "add", worktreePath, "-b", branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	return nil
}

// RemoveWorktree removes a git worktree forcefully.
func RemoveWorktree(projectRoot, worktreePath string) error {
	return exec.Command("git", "-C", projectRoot, "worktree", "remove", worktreePath, "--force").Run()
}

// DeleteBranch removes a local branch after its worktree has been cleaned up.
func DeleteBranch(projectRoot, branch string) error {
	exec.Command("git", "-C", projectRoot, "worktree", "prune").Run()
	out, err := exec.Command("git", "-C", projectRoot, "branch", "-D", branch).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	return nil
}

// MergeWorktree merges a branch into the current branch of targetDir.
func MergeWorktree(targetDir, branch string) error {
	statusOut, err := exec.Command("git", "-C", targetDir, "status", "--porcelain").CombinedOutput()
	if err != nil {
		return fmt.Errorf("git status: %w: %s", err, statusOut)
	}
	for _, line := range strings.Split(string(statusOut), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		path := parsePorcelainPath(line)
		if isAllowedLocalConfigPath(path) {
			continue
		}
		return fmt.Errorf("source root has uncommitted changes; commit or stash changes before merge")
	}

	// Pre-check for conflicts using merge-tree
	head, _ := exec.Command("git", "-C", targetDir, "rev-parse", "HEAD").Output()
	branchRev, _ := exec.Command("git", "-C", targetDir, "rev-parse", branch).Output()
	if len(head) > 0 && len(branchRev) > 0 {
		mergeBase, mergeBaseErr := exec.Command("git", "-C", targetDir, "merge-base",
			strings.TrimSpace(string(head)),
			strings.TrimSpace(string(branchRev)),
		).CombinedOutput()
		if mergeBaseErr != nil {
			return fmt.Errorf("git merge-base: %w: %s", mergeBaseErr, mergeBase)
		}
		mtOut, mtErr := exec.Command("git", "-C", targetDir, "merge-tree",
			strings.TrimSpace(string(mergeBase)),
			strings.TrimSpace(string(head)),
			strings.TrimSpace(string(branchRev)),
		).CombinedOutput()
		if mtErr != nil {
			return fmt.Errorf("git merge-tree: %w: %s", mtErr, mtOut)
		}
		if hasMergeConflictMarkers(string(mtOut)) {
			return fmt.Errorf("merge conflict detected with %s — resolve manually or let master handle:\n%s", branch, string(mtOut)[:min(len(mtOut), 500)])
		}
	}

	out, err := exec.Command("git", "-C", targetDir, "merge", "--ff-only", branch).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "note: fast-forward not possible, creating merge commit\n")
		out, err = exec.Command("git", "-C", targetDir, "merge", "--no-ff", branch).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", err, out)
		}
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func hasMergeConflictMarkers(out string) bool {
	return strings.Contains(out, "<<<<<<<") &&
		strings.Contains(out, "=======") &&
		strings.Contains(out, ">>>>>>>")
}

func branchExists(projectRoot, branch string) (bool, error) {
	out, err := exec.Command("git", "-C", projectRoot, "show-ref", "--verify", "--quiet", "refs/heads/"+branch).CombinedOutput()
	if err == nil {
		return true, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("git show-ref %s: %w: %s", branch, err, out)
}

func branchCheckedOutInAnyWorktree(projectRoot, branch string) (bool, error) {
	out, err := exec.Command("git", "-C", projectRoot, "worktree", "list", "--porcelain").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("git worktree list: %w: %s", err, out)
	}

	target := "branch refs/heads/" + branch
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == target {
			return true, nil
		}
	}
	return false, nil
}

// TagArchive creates a git tag pointing at the given branch.
func TagArchive(projectRoot, branch, tag string) error {
	return exec.Command("git", "-C", projectRoot, "tag", tag, branch).Run()
}

func parsePorcelainPath(line string) string {
	if len(line) < 4 {
		return strings.TrimSpace(line)
	}
	path := strings.TrimSpace(line[3:])
	if idx := strings.LastIndex(path, " -> "); idx >= 0 {
		path = path[idx+4:]
	}
	return strings.Trim(path, "\"")
}

func isAllowedLocalConfigPath(path string) bool {
	return path == ".goalx" || strings.HasPrefix(path, ".goalx/") ||
		path == ".goalx" || strings.HasPrefix(path, ".goalx/") ||
		path == ".claude" || strings.HasPrefix(path, ".claude/") ||
		path == ".codex" || strings.HasPrefix(path, ".codex/")
}

// hasDirtyWorktree returns true if the project has uncommitted changes
// beyond config files that are expected to be local.
func hasDirtyWorktree(projectRoot string) (bool, error) {
	out, err := exec.Command("git", "-C", projectRoot, "status", "--porcelain").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("git status: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		path := parsePorcelainPath(line)
		if isAllowedLocalConfigPath(path) {
			continue
		}
		return true, nil
	}
	return false, nil
}
