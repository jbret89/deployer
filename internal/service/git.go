package service

import (
	"fmt"
	"strings"
)

type GitService struct {
	executor commandRunner
}

func NewGitService(executor commandRunner) *GitService {
	return &GitService{
		executor: executor,
	}
}

func (g *GitService) Clone(repo, path string) error {
	output, err := g.executor.runCommand("", "git", "clone", repo, path)
	if err != nil {
		return fmt.Errorf("git clone %q into %q failed: %w: %s", repo, path, err, strings.TrimSpace(output))
	}

	return nil
}

func (g *GitService) Fetch(path string) error {
	output, err := g.executor.runCommand(path, "git", "fetch", "--all", "--prune")
	if err != nil {
		return fmt.Errorf("git fetch in %q failed: %w: %s", path, err, strings.TrimSpace(output))
	}

	return nil
}

func (g *GitService) ResetHard(path, branch string) error {
	output, err := g.executor.runCommand(path, "git", "reset", "--hard", "origin/"+branch)
	if err != nil {
		return fmt.Errorf("git reset --hard origin/%s in %q failed: %w: %s", branch, path, err, strings.TrimSpace(output))
	}

	return nil
}

func (g *GitService) CheckoutCommit(path, commit string) error {
	output, err := g.executor.runCommand(path, "git", "checkout", commit)
	if err != nil {
		return fmt.Errorf("git checkout %s in %q failed: %w: %s", commit, path, err, strings.TrimSpace(output))
	}

	return nil
}

func (g *GitService) GetPreviousCommit(path string) (string, error) {
	output, err := g.executor.runCommand(path, "git", "rev-parse", "HEAD~1")
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD~1 in %q failed: %w: %s", path, err, strings.TrimSpace(output))
	}

	return strings.TrimSpace(output), nil
}
