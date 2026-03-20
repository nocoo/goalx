package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	ar "github.com/vonbai/autoresearch"
)

func TestCreateWorktreeDoesNotDeleteExistingBranch(t *testing.T) {
	repo := initGitRepo(t)
	writeAndCommit(t, repo, "base.txt", "base", "base commit")

	runGit(t, repo, "checkout", "-b", "goalx/demo/1")
	runGit(t, repo, "checkout", "-")

	worktree := filepath.Join(t.TempDir(), "wt")
	err := CreateWorktree(repo, worktree, "goalx/demo/1")
	if err == nil {
		t.Fatal("expected CreateWorktree to fail when branch already exists")
	}

	if err := exec.Command("git", "-C", repo, "rev-parse", "--verify", "goalx/demo/1").Run(); err != nil {
		t.Fatalf("branch ar/demo/1 should still exist: %v", err)
	}
}

func TestMergeWorktreeRejectsDirtyTree(t *testing.T) {
	repo := initGitRepo(t)
	writeAndCommit(t, repo, "base.txt", "base", "base commit")

	runGit(t, repo, "checkout", "-b", "feature")
	writeAndCommit(t, repo, "feature.txt", "feature", "feature commit")
	runGit(t, repo, "checkout", "-")

	if err := os.WriteFile(filepath.Join(repo, "dirty.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	if err := MergeWorktree(repo, "feature"); err == nil {
		t.Fatal("expected MergeWorktree to reject dirty worktree")
	}
}

func TestMergeWorktreeRejectsNonFastForward(t *testing.T) {
	repo := initGitRepo(t)
	writeAndCommit(t, repo, "base.txt", "base", "base commit")

	runGit(t, repo, "checkout", "-b", "feature")
	writeAndCommit(t, repo, "feature.txt", "feature", "feature commit")
	runGit(t, repo, "checkout", "-")
	writeAndCommit(t, repo, "main.txt", "main", "main commit")

	if err := MergeWorktree(repo, "feature"); err == nil {
		t.Fatal("expected MergeWorktree to reject non-fast-forward merge")
	}
}

func TestMergeWorktreeAllowsLocalAutoresearchFiles(t *testing.T) {
	repo := initGitRepo(t)
	writeAndCommit(t, repo, "base.txt", "base", "base commit")

	runGit(t, repo, "checkout", "-b", "feature")
	writeAndCommit(t, repo, "feature.txt", "feature", "feature commit")
	runGit(t, repo, "checkout", "-")

	if err := os.WriteFile(filepath.Join(repo, "goalx.yaml"), []byte("name: demo\n"), 0o644); err != nil {
		t.Fatalf("write ar.yaml: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".autoresearch"), 0o755); err != nil {
		t.Fatalf("mkdir .autoresearch: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".autoresearch", "config.yaml"), []byte("parallel: 1\n"), 0o644); err != nil {
		t.Fatalf("write .autoresearch/config.yaml: %v", err)
	}

	if err := MergeWorktree(repo, "feature"); err != nil {
		t.Fatalf("expected MergeWorktree to allow local autoresearch files, got: %v", err)
	}
}

func TestDropRemovesRunDirectoryAndBranch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	repo := initGitRepo(t)
	writeAndCommit(t, repo, "base.txt", "base", "base commit")

	runName := "drop-run"
	runDir := ar.RunDir(repo, runName)
	if err := os.MkdirAll(filepath.Join(runDir, "worktrees"), 0o755); err != nil {
		t.Fatalf("mkdir worktrees: %v", err)
	}
	snapshot := []byte("name: drop-run\nmode: research\nobjective: demo\ntarget:\n  files: [\"report.md\"]\nharness:\n  command: \"test -f base.txt\"\n")
	if err := os.WriteFile(filepath.Join(runDir, "goalx.yaml"), snapshot, 0o644); err != nil {
		t.Fatalf("write run snapshot: %v", err)
	}

	branch := "goalx/drop-run/1"
	worktreePath := filepath.Join(runDir, "worktrees", "drop-run-1")
	if err := CreateWorktree(repo, worktreePath, branch); err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	if err := Drop(repo, []string{"--run", runName}); err != nil {
		t.Fatalf("Drop: %v", err)
	}

	if _, err := os.Stat(runDir); !os.IsNotExist(err) {
		t.Fatalf("run dir still exists: %v", err)
	}
	if err := exec.Command("git", "-C", repo, "rev-parse", "--verify", branch).Run(); err == nil {
		t.Fatalf("branch %s should be deleted", branch)
	}
}

func initGitRepo(t *testing.T) string {
	t.Helper()

	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "config", "user.email", "test@example.com")
	return repo
}

func writeAndCommit(t *testing.T, repo, name, content, message string) {
	t.Helper()

	path := filepath.Join(repo, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	runGit(t, repo, "add", name)
	runGit(t, repo, "commit", "-m", message)
}

func runGit(t *testing.T, repo string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, string(out))
	}
}
