package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"deployer/internal/config"
)

type Deployer struct {
	executor *CommandExecutor
	git      gitOperations
	docker   dockerOperations
	config   config.Config
	logger   *slog.Logger
	locks    sync.Map
}

type gitOperations interface {
	Clone(repo, path string) error
	Fetch(path string) error
	ResetHard(path, branch string) error
	CheckoutCommit(path, commit string) error
	GetPreviousCommit(path string) (string, error)
}

type dockerOperations interface {
	ComposeUp(path, service string) error
}

func NewDeployer(cfg config.Config, logger *slog.Logger) *Deployer {
	executor := NewCommandExecutor(logger, cfg.CommandTimeout)
	return newDeployerWithDependencies(cfg, logger, executor, NewGitService(executor, logger), NewDockerService(executor, logger))
}

func newDeployerWithDependencies(cfg config.Config, logger *slog.Logger, executor *CommandExecutor, git gitOperations, docker dockerOperations) *Deployer {
	return &Deployer{
		executor: executor,
		git:      git,
		docker:   docker,
		config:   cfg,
		logger:   logger,
	}
}

func (d *Deployer) Deploy(_ context.Context, repo string) (string, error) {
	unlock := d.lockRepo(repo)
	defer unlock()

	repoPath := d.resolveRepoPath(repo)
	repoSSH := d.resolveRepoSSH(repo)
	start := time.Now()

	var logs []string
	d.logger.Info("deploy started", "repo", repo, "path", repoPath, "branch", d.config.Branch)

	if err := os.MkdirAll(d.config.DeployBasePath, 0o755); err != nil {
		return "", fmt.Errorf("create deploy base path %q: %w", d.config.DeployBasePath, err)
	}

	info, err := os.Stat(repoPath)
	switch {
	case os.IsNotExist(err):
		logs = append(logs, fmt.Sprintf("cloning %s into %s", repoSSH, repoPath))
		if err := d.git.Clone(repoSSH, repoPath); err != nil {
			return sanitizeOutput(strings.Join(logs, "\n")), err
		}
	case err != nil:
		return "", fmt.Errorf("stat repo path %q: %w", repoPath, err)
	case !info.IsDir():
		return "", fmt.Errorf("repo path %q exists and is not a directory", repoPath)
	default:
		logs = append(logs, fmt.Sprintf("fetching updates in %s", repoPath))
		if err := d.git.Fetch(repoPath); err != nil {
			return sanitizeOutput(strings.Join(logs, "\n")), err
		}

		logs = append(logs, fmt.Sprintf("resetting repository to origin/%s", d.config.Branch))
		if err := d.git.ResetHard(repoPath, d.config.Branch); err != nil {
			return sanitizeOutput(strings.Join(logs, "\n")), err
		}
	}

	logs = append(logs, fmt.Sprintf("running docker compose up for service %s", repo))
	if err := d.docker.ComposeUp(repoPath, repo); err != nil {
		return sanitizeOutput(strings.Join(logs, "\n")), err
	}

	logs = append(logs, "deploy completed successfully")
	sanitizedLogs := sanitizeOutput(strings.Join(logs, "\n"))
	d.logger.Info("deploy completed", "repo", repo, "path", repoPath, "duration", time.Since(start).String())
	return sanitizedLogs, nil
}

func (d *Deployer) Rollback(_ context.Context, repo string) (string, error) {
	unlock := d.lockRepo(repo)
	defer unlock()

	repoPath := d.resolveRepoPath(repo)
	start := time.Now()

	var logs []string
	d.logger.Info("rollback started", "repo", repo, "path", repoPath)

	if err := d.ensureRepoDirectory(repoPath); err != nil {
		return "", err
	}

	logs = append(logs, fmt.Sprintf("resolving previous commit in %s", repoPath))
	previousCommit, err := d.git.GetPreviousCommit(repoPath)
	if err != nil {
		return sanitizeOutput(strings.Join(logs, "\n")), err
	}

	logs = append(logs, fmt.Sprintf("checking out previous commit %s", previousCommit))
	if err := d.git.CheckoutCommit(repoPath, previousCommit); err != nil {
		return sanitizeOutput(strings.Join(logs, "\n")), err
	}

	logs = append(logs, fmt.Sprintf("running docker compose up for service %s", repo))
	if err := d.docker.ComposeUp(repoPath, repo); err != nil {
		return sanitizeOutput(strings.Join(logs, "\n")), err
	}

	logs = append(logs, "rollback completed successfully")
	sanitizedLogs := sanitizeOutput(strings.Join(logs, "\n"))
	d.logger.Info("rollback completed", "repo", repo, "path", repoPath, "duration", time.Since(start).String())
	return sanitizedLogs, nil
}

func (d *Deployer) resolveRepoPath(repo string) string {
	return filepath.Join(d.config.DeployBasePath, repo)
}

func (d *Deployer) resolveRepoSSH(repo string) string {
	base := d.config.GitBaseSSH
	if strings.HasSuffix(base, "/") || strings.HasSuffix(base, ":") {
		return base + repo + ".git"
	}

	return base + "/" + repo + ".git"
}

func (d *Deployer) ensureRepoDirectory(repoPath string) error {
	info, err := os.Stat(repoPath)
	switch {
	case os.IsNotExist(err):
		return fmt.Errorf("repo path %q does not exist", repoPath)
	case err != nil:
		return fmt.Errorf("stat repo path %q: %w", repoPath, err)
	case !info.IsDir():
		return fmt.Errorf("repo path %q exists and is not a directory", repoPath)
	default:
		return nil
	}
}

func (d *Deployer) lockRepo(repo string) func() {
	lock, _ := d.locks.LoadOrStore(repo, &sync.Mutex{})
	mutex := lock.(*sync.Mutex)
	mutex.Lock()

	return mutex.Unlock
}
