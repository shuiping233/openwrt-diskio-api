package metric

import (
	"os/exec"
	"strings"
)

type CommandRunnerInterface interface {
	Run(name string, args ...string) (string, error)
}

type CommandRunner struct{}

func (c CommandRunner) Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)

	result, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(result)), nil
}
