package config

import (
	"log/slog"
	"testing"
	"time"
)

func TestLoadUsesDefaultsWhenEnvVarsAreMissing(t *testing.T) {
	t.Setenv("HOST", "")
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("DEPLOY_BASE_PATH", "")
	t.Setenv("GIT_BASE_SSH", "")
	t.Setenv("BRANCH", "")
	t.Setenv("ADMIN_TOKEN", "")

	cfg := Load()

	if cfg.Host != "" {
		t.Fatalf("expected empty host, got %q", cfg.Host)
	}
	if cfg.Port != defaultPort {
		t.Fatalf("expected default port %q, got %q", defaultPort, cfg.Port)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Fatalf("expected info log level, got %v", cfg.LogLevel)
	}
	if cfg.DeployBasePath != defaultDeployBasePath {
		t.Fatalf("expected default deploy path %q, got %q", defaultDeployBasePath, cfg.DeployBasePath)
	}
	if cfg.GitBaseSSH != defaultGitBaseSSH {
		t.Fatalf("expected default git base %q, got %q", defaultGitBaseSSH, cfg.GitBaseSSH)
	}
	if cfg.Branch != defaultBranch {
		t.Fatalf("expected default branch %q, got %q", defaultBranch, cfg.Branch)
	}
	if cfg.AdminToken != "" {
		t.Fatalf("expected empty admin token, got %q", cfg.AdminToken)
	}
	if cfg.CommandTimeout != defaultCommandTimeout {
		t.Fatalf("expected default command timeout %s, got %s", defaultCommandTimeout, cfg.CommandTimeout)
	}
}

func TestLoadUsesEnvVarsWhenProvided(t *testing.T) {
	t.Setenv("HOST", "127.0.0.1")
	t.Setenv("PORT", "9090")
	t.Setenv("LOG_LEVEL", "DEBUG")
	t.Setenv("DEPLOY_BASE_PATH", "/tmp/repos")
	t.Setenv("GIT_BASE_SSH", "git@github.com:my-org/")
	t.Setenv("BRANCH", "develop")
	t.Setenv("ADMIN_TOKEN", "secret")
	t.Setenv("COMMAND_TIMEOUT", "30s")

	cfg := Load()

	if cfg.Host != "127.0.0.1" {
		t.Fatalf("expected host override, got %q", cfg.Host)
	}
	if cfg.Port != "9090" {
		t.Fatalf("expected port override, got %q", cfg.Port)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Fatalf("expected debug log level, got %v", cfg.LogLevel)
	}
	if cfg.DeployBasePath != "/tmp/repos" {
		t.Fatalf("expected deploy path override, got %q", cfg.DeployBasePath)
	}
	if cfg.GitBaseSSH != "git@github.com:my-org/" {
		t.Fatalf("expected git base override, got %q", cfg.GitBaseSSH)
	}
	if cfg.Branch != "develop" {
		t.Fatalf("expected branch override, got %q", cfg.Branch)
	}
	if cfg.AdminToken != "secret" {
		t.Fatalf("expected admin token override, got %q", cfg.AdminToken)
	}
	if cfg.CommandTimeout != 30*time.Second {
		t.Fatalf("expected command timeout override, got %s", cfg.CommandTimeout)
	}
}
