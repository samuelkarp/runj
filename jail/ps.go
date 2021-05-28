package jail

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strconv"
)

// IsRunning attempts to determine whether a given jail is running.  This is
// accomplished by looking to see whether the jail's primary pid (passed as an
// argument) is still active and by whether there are any processes present in
// the jail.  This function is best-effort, racy, and subject to change.  It
// currently depends on the host's "ps" command.
func IsRunning(ctx context.Context, jail string, pid int) (bool, error) {
	if pid > 0 {
		if ok, err := psCmd(exec.CommandContext(ctx, "ps", "--libxo", "json", "-x", strconv.Itoa(pid))); err != nil {
			return false, err
		} else if ok {
			// if the primary pid is present, we're done
			return true, nil
		}
	}

	var (
		ok  bool
		err error
	)
	if ok, err = psCmd(exec.CommandContext(ctx, "ps", "--libxo", "json", "-x", "-J", jail)); err != nil {
		return false, err
	}
	return ok, nil
}

// psCmd executes a "ps" command provided as an *exec.Cmd and output with libxo
// json and parses the result to determine whether any processes are running.
func psCmd(cmd *exec.Cmd) (bool, error) {
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
