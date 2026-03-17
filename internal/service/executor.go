package service

import "os/exec"

type commandRunner interface {
	runCommand(dir, cmd string, args ...string) (string, error)
}

type CommandExecutor struct{}

func NewCommandExecutor() *CommandExecutor {
	return &CommandExecutor{}
}

func (e *CommandExecutor) runCommand(dir, cmd string, args ...string) (string, error) {
	command := exec.Command(cmd, args...)
	command.Dir = dir

	output, err := command.CombinedOutput()
	return string(output), err
}
