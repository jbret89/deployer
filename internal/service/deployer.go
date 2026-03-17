package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"deployer/internal/config"
)

type Deployer struct {
	executor *CommandExecutor
	git      *GitService
	docker   *DockerService
	config   config.Config
}

func NewDeployer(cfg config.Config) *Deployer {
	executor := NewCommandExecutor()

	return &Deployer{
		executor: executor,
		git:      NewGitService(executor),
		docker:   NewDockerService(executor),
		config:   cfg,
	}
}

func (d *Deployer) Deploy(_ context.Context, repo string) (string, error) {
	repoPath := d.resolveRepoPath(repo)
	repoSSH := d.resolveRepoSSH(repo)

	var logs []string

	if err := os.MkdirAll(d.config.DeployBasePath, 0o755); err != nil {
		return "", fmt.Errorf("create deploy base path %q: %w", d.config.DeployBasePath, err)
	}

	info, err := os.Stat(repoPath)
	switch {
	case os.IsNotExist(err):
		logs = append(logs, fmt.Sprintf("cloning %s into %s", repoSSH, repoPath))
		if err := d.git.Clone(repoSSH, repoPath); err != nil {
			return strings.Join(logs, "\n"), err
		}
	case err != nil:
		return "", fmt.Errorf("stat repo path %q: %w", repoPath, err)
	case !info.IsDir():
		return "", fmt.Errorf("repo path %q exists and is not a directory", repoPath)
	default:
		logs = append(logs, fmt.Sprintf("fetching updates in %s", repoPath))
		if err := d.git.Fetch(repoPath); err != nil {
			return strings.Join(logs, "\n"), err
		}

		logs = append(logs, fmt.Sprintf("resetting repository to origin/%s", d.config.Branch))
		if err := d.git.ResetHard(repoPath, d.config.Branch); err != nil {
			return strings.Join(logs, "\n"), err
		}
	}

	logs = append(logs, fmt.Sprintf("running docker compose up for service %s", repo))
	if err := d.docker.ComposeUp(repoPath, repo); err != nil {
		return strings.Join(logs, "\n"), err
	}

	logs = append(logs, "deploy completed successfully")
	return strings.Join(logs, "\n"), nil
}

func (d *Deployer) Rollback(_ context.Context, repo string) (string, error) {
	repoPath := d.resolveRepoPath(repo)

	var logs []string

	if err := d.ensureRepoDirectory(repoPath); err != nil {
		return "", err
	}

	logs = append(logs, fmt.Sprintf("resolving previous commit in %s", repoPath))
	previousCommit, err := d.git.GetPreviousCommit(repoPath)
	if err != nil {
		return strings.Join(logs, "\n"), err
	}

	logs = append(logs, fmt.Sprintf("checking out previous commit %s", previousCommit))
	if err := d.git.CheckoutCommit(repoPath, previousCommit); err != nil {
		return strings.Join(logs, "\n"), err
	}

	logs = append(logs, fmt.Sprintf("running docker compose up for service %s", repo))
	if err := d.docker.ComposeUp(repoPath, repo); err != nil {
		return strings.Join(logs, "\n"), err
	}

	logs = append(logs, "rollback completed successfully")
	return strings.Join(logs, "\n"), nil
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
