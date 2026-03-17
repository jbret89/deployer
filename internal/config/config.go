package config

import (
	"log/slog"
	"os"
	"time"
)

const (
	defaultPort           = "8080"
	defaultDeployBasePath = "./repos"
	defaultGitBaseSSH     = "git@github.com:"
	defaultBranch         = "main"
	defaultCommandTimeout = 5 * time.Minute
)

type Config struct {
	Host           string
	Port           string
	LogLevel       slog.Level
	DeployBasePath string
	GitBaseSSH     string
	Branch         string
	AdminToken     string
	CommandTimeout time.Duration
}

func Load() Config {
	return Config{
		Host:           getEnv("HOST", ""),
		Port:           getEnv("PORT", defaultPort),
		LogLevel:       parseLogLevel(getEnv("LOG_LEVEL", "INFO")),
		DeployBasePath: getEnv("DEPLOY_BASE_PATH", defaultDeployBasePath),
		GitBaseSSH:     getEnv("GIT_BASE_SSH", defaultGitBaseSSH),
		Branch:         getEnv("BRANCH", defaultBranch),
		AdminToken:     getEnv("ADMIN_TOKEN", ""),
		CommandTimeout: parseDuration(getEnv("COMMAND_TIMEOUT", defaultCommandTimeout.String()), defaultCommandTimeout),
	}
}

func (c Config) Address() string {
	if c.Host == "" {
		return ":" + c.Port
	}

	return c.Host + ":" + c.Port
}

func parseLogLevel(raw string) slog.Level {
	var level slog.Level
	if err := level.UnmarshalText([]byte(raw)); err != nil {
		return slog.LevelInfo
	}

	return level
}

func parseDuration(raw string, fallback time.Duration) time.Duration {
	duration, err := time.ParseDuration(raw)
	if err != nil || duration <= 0 {
		return fallback
	}

	return duration
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}

	return fallback
}
