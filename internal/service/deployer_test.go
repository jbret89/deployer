package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"deployer/internal/config"
)

type fakeGit struct {
	cloneCalls          []cloneCall
	fetchCalls          []string
	resetCalls          []resetCall
	checkoutCalls       []checkoutCall
	previousCommitCalls []string
	cloneErr            error
	fetchErr            error
	resetErr            error
	checkoutErr         error
	previousCommitErr   error
	previousCommit      string
}

type cloneCall struct {
	repo string
	path string
}

type resetCall struct {
	path   string
	branch string
}

type checkoutCall struct {
	path   string
	commit string
}

func (f *fakeGit) Clone(repo, path string) error {
	f.cloneCalls = append(f.cloneCalls, cloneCall{repo: repo, path: path})
	return f.cloneErr
}

func (f *fakeGit) Fetch(path string) error {
	f.fetchCalls = append(f.fetchCalls, path)
	return f.fetchErr
}

func (f *fakeGit) ResetHard(path, branch string) error {
	f.resetCalls = append(f.resetCalls, resetCall{path: path, branch: branch})
	return f.resetErr
}

func (f *fakeGit) CheckoutCommit(path, commit string) error {
	f.checkoutCalls = append(f.checkoutCalls, checkoutCall{path: path, commit: commit})
	return f.checkoutErr
}

func (f *fakeGit) GetPreviousCommit(path string) (string, error) {
	f.previousCommitCalls = append(f.previousCommitCalls, path)
	return f.previousCommit, f.previousCommitErr
}

type fakeDocker struct {
	composeCalls []composeCall
	composeErr   error
}

type composeCall struct {
	path    string
	service string
}

func (f *fakeDocker) ComposeUp(path, service string) error {
	f.composeCalls = append(f.composeCalls, composeCall{path: path, service: service})
	return f.composeErr
}

func TestDeployClonesMissingRepoAndRunsCompose(t *testing.T) {
	basePath := t.TempDir()
	git := &fakeGit{}
	docker := &fakeDocker{}
	deployer := newDeployerWithDependencies(testConfig(basePath), testLogger(), nil, git, docker)

	logs, err := deployer.Deploy(context.Background(), "app")
	if err != nil {
		t.Fatalf("deploy returned error: %v", err)
	}

	expectedPath := filepath.Join(basePath, "app")
	if len(git.cloneCalls) != 1 {
		t.Fatalf("expected one clone call, got %d", len(git.cloneCalls))
	}
	if git.cloneCalls[0].repo != "git@github.com:my-org/app.git" {
		t.Fatalf("unexpected clone repo %q", git.cloneCalls[0].repo)
	}
	if git.cloneCalls[0].path != expectedPath {
		t.Fatalf("unexpected clone path %q", git.cloneCalls[0].path)
	}
	if len(docker.composeCalls) != 1 {
		t.Fatalf("expected one compose call, got %d", len(docker.composeCalls))
	}
	if docker.composeCalls[0].path != expectedPath || docker.composeCalls[0].service != "app" {
		t.Fatalf("unexpected compose call %#v", docker.composeCalls[0])
	}
	if !strings.Contains(logs, "deploy completed successfully") {
		t.Fatalf("expected success log, got %q", logs)
	}
}

func TestDeployFetchesAndResetsExistingRepo(t *testing.T) {
	basePath := t.TempDir()
	repoPath := filepath.Join(basePath, "app")
	if err := mkdir(repoPath); err != nil {
		t.Fatalf("mkdir repo path: %v", err)
	}

	git := &fakeGit{}
	docker := &fakeDocker{}
	deployer := newDeployerWithDependencies(testConfig(basePath), testLogger(), nil, git, docker)

	_, err := deployer.Deploy(context.Background(), "app")
	if err != nil {
		t.Fatalf("deploy returned error: %v", err)
	}

	if len(git.fetchCalls) != 1 || git.fetchCalls[0] != repoPath {
		t.Fatalf("unexpected fetch calls %#v", git.fetchCalls)
	}
	if len(git.resetCalls) != 1 {
		t.Fatalf("expected one reset call, got %d", len(git.resetCalls))
	}
	if git.resetCalls[0].branch != "main" {
		t.Fatalf("expected reset branch main, got %q", git.resetCalls[0].branch)
	}
}

func TestDeployReturnsPartialLogsWhenComposeFails(t *testing.T) {
	basePath := t.TempDir()
	git := &fakeGit{}
	docker := &fakeDocker{composeErr: errors.New("compose failed")}
	deployer := newDeployerWithDependencies(testConfig(basePath), testLogger(), nil, git, docker)

	logs, err := deployer.Deploy(context.Background(), "app")
	if err == nil {
		t.Fatal("expected deploy error")
	}
	if !strings.Contains(logs, "running docker compose up for service app") {
		t.Fatalf("expected compose log in %q", logs)
	}
}

func TestRollbackChecksOutPreviousCommitAndRunsCompose(t *testing.T) {
	basePath := t.TempDir()
	repoPath := filepath.Join(basePath, "app")
	if err := mkdir(repoPath); err != nil {
		t.Fatalf("mkdir repo path: %v", err)
	}

	git := &fakeGit{previousCommit: "abc123"}
	docker := &fakeDocker{}
	deployer := newDeployerWithDependencies(testConfig(basePath), testLogger(), nil, git, docker)

	logs, err := deployer.Rollback(context.Background(), "app")
	if err != nil {
		t.Fatalf("rollback returned error: %v", err)
	}

	if len(git.previousCommitCalls) != 1 || git.previousCommitCalls[0] != repoPath {
		t.Fatalf("unexpected previous commit calls %#v", git.previousCommitCalls)
	}
	if len(git.checkoutCalls) != 1 {
		t.Fatalf("expected one checkout call, got %d", len(git.checkoutCalls))
	}
	if git.checkoutCalls[0].commit != "abc123" {
		t.Fatalf("expected commit abc123, got %q", git.checkoutCalls[0].commit)
	}
	if len(docker.composeCalls) != 1 {
		t.Fatalf("expected one compose call, got %d", len(docker.composeCalls))
	}
	if !strings.Contains(logs, "rollback completed successfully") {
		t.Fatalf("expected success log, got %q", logs)
	}
}

func TestRollbackFailsWhenRepoDoesNotExist(t *testing.T) {
	deployer := newDeployerWithDependencies(testConfig(t.TempDir()), testLogger(), nil, &fakeGit{}, &fakeDocker{})

	_, err := deployer.Rollback(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected rollback error")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestTryLockRepoRejectsConcurrentAccess(t *testing.T) {
	deployer := newDeployerWithDependencies(testConfig(t.TempDir()), testLogger(), nil, &fakeGit{}, &fakeDocker{})

	unlock, ok := deployer.tryLockRepo("app")
	if !ok {
		t.Fatal("expected first lock acquisition to succeed")
	}
	defer unlock()

	secondUnlock, ok := deployer.tryLockRepo("app")
	if ok {
		secondUnlock()
		t.Fatal("expected second lock acquisition to fail")
	}
}

func testConfig(basePath string) config.Config {
	return config.Config{
		DeployBasePath: basePath,
		GitBaseSSH:     "git@github.com:my-org/",
		Branch:         "main",
		CommandTimeout: 5 * time.Second,
	}
}

func mkdir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
