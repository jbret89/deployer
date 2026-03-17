package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

type commandRunner interface {
	runCommand(dir, cmd string, args ...string) (string, error)
}

type CommandExecutor struct {
	logger  *slog.Logger
	timeout time.Duration
}

func NewCommandExecutor(logger *slog.Logger, timeout time.Duration) *CommandExecutor {
	return &CommandExecutor{
		logger:  logger,
		timeout: timeout,
	}
}

func (e *CommandExecutor) runCommand(dir, cmd string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	command := exec.CommandContext(ctx, cmd, args...)
	command.Dir = dir

	start := time.Now()
	output, err := command.CombinedOutput()
	duration := time.Since(start)
	sanitizedOutput := sanitizeOutput(string(output))
	joinedArgs := strings.Join(args, " ")

	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			e.logger.Error("command timed out", "cmd", cmd, "args", joinedArgs, "dir", dir, "timeout", e.timeout.String(), "duration", duration.String())
			return sanitizedOutput, fmt.Errorf("command timed out after %s", e.timeout)
		}

		e.logger.Error("command failed", "cmd", cmd, "args", joinedArgs, "dir", dir, "duration", duration.String(), "error", err)
		return sanitizedOutput, err
	}

	e.logger.Info("command completed", "cmd", cmd, "args", joinedArgs, "dir", dir, "duration", duration.String())
	return sanitizedOutput, nil
}
