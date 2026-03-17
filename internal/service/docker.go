package service

import (
	"fmt"
	"strings"
)

type DockerService struct {
	executor commandRunner
}

func NewDockerService(executor commandRunner) *DockerService {
	return &DockerService{
		executor: executor,
	}
}

func (d *DockerService) ComposeUp(path, service string) error {
	output, err := d.executor.runCommand(path, "docker", "compose", "up", "-d", "--build", service)
	if err != nil {
		return fmt.Errorf("docker compose up failed for service %q in %q: %w: %s", service, path, err, strings.TrimSpace(output))
	}

	return nil
}
