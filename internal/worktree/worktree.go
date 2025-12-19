package worktree

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func GetRepoRoot(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	return strings.TrimSpace(out.String()), nil
}

func GetWorktreeDir(repoRoot string) string {
	repoName := filepath.Base(repoRoot)
	return filepath.Join(filepath.Dir(repoRoot), repoName+"-worktrees")
}

func GetWorktreePath(repoRoot, branch string) string {
	return filepath.Join(GetWorktreeDir(repoRoot), sanitizeBranch(branch))
}

func sanitizeBranch(branch string) string {
	result := make([]byte, 0, len(branch))
	for i := 0; i < len(branch); i++ {
		c := branch[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result = append(result, c)
		} else {
			result = append(result, '-')
		}
	}
	return string(result)
}

func Exists(repoRoot, branch string) bool {
	path := GetWorktreePath(repoRoot, branch)
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func BranchExists(repoRoot, branch string) (bool, error) {
	cmd := exec.Command("git", "-C", repoRoot, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			cmd2 := exec.Command("git", "-C", repoRoot, "show-ref", "--verify", "--quiet", "refs/remotes/origin/"+branch)
			return cmd2.Run() == nil, nil
		}
		return false, err
	}
	return true, nil
}

func Create(repoRoot, branch string) (string, error) {
	worktreeDir := GetWorktreeDir(repoRoot)
	if err := os.MkdirAll(worktreeDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create worktree directory: %w", err)
	}

	worktreePath := GetWorktreePath(repoRoot, branch)

	exists, err := BranchExists(repoRoot, branch)
	if err != nil {
		return "", fmt.Errorf("failed to check branch: %w", err)
	}

	var cmd *exec.Cmd
	if exists {
		cmd = exec.Command("git", "-C", repoRoot, "worktree", "add", worktreePath, branch)
	} else {
		cmd = exec.Command("git", "-C", repoRoot, "worktree", "add", "-b", branch, worktreePath)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create worktree: %s", stderr.String())
	}

	return worktreePath, nil
}

func Remove(repoRoot, branch string) error {
	worktreePath := GetWorktreePath(repoRoot, branch)

	cmd := exec.Command("git", "-C", repoRoot, "worktree", "remove", worktreePath, "--force")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove worktree: %s", stderr.String())
	}

	return nil
}

func List(repoRoot string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoRoot, "worktree", "list", "--porcelain")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var paths []string
	for _, line := range strings.Split(out.String(), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			paths = append(paths, strings.TrimPrefix(line, "worktree "))
		}
	}
	return paths, nil
}

func IsMainRepo(path string) (bool, error) {
	root, err := GetRepoRoot(path)
	if err != nil {
		return false, err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}

	return absPath == root || filepath.Clean(absPath) == filepath.Clean(root), nil
}
