package service

import (
	"fmt"
	"log/slog"
	"strings"
)

type GitService struct {
	executor commandRunner
	logger   *slog.Logger
}

func NewGitService(executor commandRunner, logger *slog.Logger) *GitService {
	return &GitService{
		executor: executor,
		logger:   logger,
	}
}

func (g *GitService) Clone(repo, path string) error {
	g.logger.Info("git clone", "repo", repo, "path", path)
	output, err := g.executor.runCommand("", "git", "clone", repo, path)
	if err != nil {
		return fmt.Errorf("git clone %q into %q failed: %w: %s", repo, path, err, sanitizeOutput(strings.TrimSpace(output)))
	}

	return nil
}

func (g *GitService) Fetch(path string) error {
	g.logger.Info("git fetch", "path", path)
	output, err := g.executor.runCommand(path, "git", "fetch", "--all", "--prune")
	if err != nil {
		return fmt.Errorf("git fetch in %q failed: %w: %s", path, err, sanitizeOutput(strings.TrimSpace(output)))
	}

	return nil
}

func (g *GitService) ResetHard(path, branch string) error {
	g.logger.Info("git reset --hard", "path", path, "branch", branch)
	output, err := g.executor.runCommand(path, "git", "reset", "--hard", "origin/"+branch)
	if err != nil {
		return fmt.Errorf("git reset --hard origin/%s in %q failed: %w: %s", branch, path, err, sanitizeOutput(strings.TrimSpace(output)))
	}

	return nil
}

func (g *GitService) CheckoutCommit(path, commit string) error {
	g.logger.Info("git checkout", "path", path, "commit", commit)
	output, err := g.executor.runCommand(path, "git", "checkout", commit)
	if err != nil {
		return fmt.Errorf("git checkout %s in %q failed: %w: %s", commit, path, err, sanitizeOutput(strings.TrimSpace(output)))
	}

	return nil
}

func (g *GitService) GetPreviousCommit(path string) (string, error) {
	g.logger.Info("git rev-parse", "path", path, "ref", "HEAD~1")
	output, err := g.executor.runCommand(path, "git", "rev-parse", "HEAD~1")
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD~1 in %q failed: %w: %s", path, err, sanitizeOutput(strings.TrimSpace(output)))
	}

	return sanitizeOutput(strings.TrimSpace(output)), nil
}
