package service

import (
	"fmt"
	"log/slog"
	"strings"
)

type DockerService struct {
	executor commandRunner
	logger   *slog.Logger
}

func NewDockerService(executor commandRunner, logger *slog.Logger) *DockerService {
	return &DockerService{
		executor: executor,
		logger:   logger,
	}
}

func (d *DockerService) ComposeUp(path, service string) error {
	d.logger.Info("docker compose up", "path", path, "service", service)
	output, err := d.executor.runCommand(path, "docker", "compose", "up", "-d", "--build", service)
	if err != nil {
		return fmt.Errorf("docker compose up failed for service %q in %q: %w: %s", service, path, err, sanitizeOutput(strings.TrimSpace(output)))
	}

	return nil
}
