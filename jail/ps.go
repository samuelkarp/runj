package jail

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
)

func IsRunning(ctx context.Context, jail string) (bool, error) {
	cmd := exec.CommandContext(ctx, "ps", "--libxo", "json", "-x", "-J", jail)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		// `ps` exits with 1 when there are no processes, which is a valid state
		if ee, ok := err.(*exec.ExitError); ok {
			if ee.ProcessState.ExitCode() == 1 {
				return false, nil
			}
		}
		return false, err
	}
	result := &psOutput{}
	err = json.Unmarshal(out, result)
	if err != nil {
		return false, err
	}
	if result == nil || result.ProcessInformation == nil {
		return false, errors.New("nil result")
	}
	return len(result.ProcessInformation.Processes) > 0, nil
}

type psOutput struct {
	ProcessInformation *psInfo `json:"process-information"`
}

type psInfo struct {
	Processes []psProcess `json:"process"`
}

type psProcess struct {
	PID          string `json:"pid"`
	TerminalName string `json:"terminal-name"`
	State        string `json:"state"`
	CPUTime      string `json:"cpu-time"`
	Command      string `json:"command"`
}
