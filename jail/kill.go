package jail

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"golang.org/x/sys/unix"
)

// Kill sends a signal to a process in a jail
func Kill(ctx context.Context, jail string, pid int, signal unix.Signal) error {
	cmd := exec.CommandContext(ctx, "jexec", jail, "kill", fmt.Sprintf("-%d", signal), strconv.Itoa(pid))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// KillAll sends a signal to all processes in a jail
func KillAll(ctx context.Context, jail string, signal unix.Signal) error {
	return Kill(ctx, jail, -1, signal)
}
