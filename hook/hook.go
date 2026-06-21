package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"go.sbk.wtf/runj/state"
)

// killGracePeriod bounds how long Run waits for a hook's stdio pipes to close
// after its timeout has fired and the process group has been killed.  It is a
// backstop for a process that escaped the group (e.g. via setsid) and is still
// holding the pipes open.
const killGracePeriod = 2 * time.Second

// Run runs a given hook
func Run(s *state.Output, h *runtimespec.Hook) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	var stdout, stderr bytes.Buffer
	var cancelFunc context.CancelFunc
	ctx := context.Background()

	if h.Timeout != nil {
		ctx, cancelFunc = context.WithTimeout(ctx, time.Duration(*h.Timeout)*time.Second)
		defer cancelFunc()
	}

	cmd := exec.CommandContext(ctx, h.Path, h.Args[1:]...)
	cmd.Env = h.Env
	cmd.Stdin = bytes.NewReader(b)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// Run the hook in its own process group.  Without this, a child forked by
	// the hook survives the SIGKILL sent to the hook process on timeout and
	// keeps the inherited stdio pipes open, blocking Wait well past the
	// timeout.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if h.Timeout != nil {
		// On timeout, signal the whole process group.  Setpgid makes the
		// group ID equal to the hook's PID, so -PID addresses the group.
		cmd.Cancel = func() error {
			err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			if errors.Is(err, syscall.ESRCH) {
				return os.ErrProcessDone
			}
			return err
		}
		cmd.WaitDelay = killGracePeriod
	}

	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running hook: %v, stdout: %s, stderr: %s\n", err, stdout.String(), stderr.String())
	}

	return err
}
