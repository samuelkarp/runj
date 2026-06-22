package containerd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/containerd/containerd/api/types"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/containerd/log"
	"golang.org/x/sys/unix"
)

// NewManager returns a shim.Manager for the runj shim.  The manager forks the
// shim process and handles fallback cleanup; the task API is served by the
// plugin-registered service (see NewTaskService).
func NewManager(name string) shim.Manager {
	return manager{name: name}
}

type manager struct {
	name string
}

func (m manager) Name() string {
	return m.name
}

// Start forks a new shim process for the container and returns the bootstrap
// parameters, including the address at which the shim can be reached. This
// decoupled logic allows the shim logic itself to decide how many shims are in
// use: one per container, one per machine, one per group of containers, or
// some other decision. When this function returns, the current process exits.
// If there is no existing shim with an address to use, this function must fork
// a new shim process before returning.
func (m manager) Start(ctx context.Context, id string, opts shim.StartOpts) (_ shim.BootstrapParams, retErr error) {
	params := shim.BootstrapParams{
		Version:  3,
		Protocol: "ttrpc",
	}
	cmd, err := newReexec(ctx, id, opts.Address)
	if err != nil {
		return params, err
	}
	address, err := shim.SocketAddress(ctx, opts.Address, id, false)
	if err != nil {
		return params, err
	}
	socket, err := shim.NewSocket(address)
	if err != nil {
		if !shim.SocketEaddrinuse(err) {
			return params, err
		}
		if err := shim.RemoveSocket(address); err != nil {
			return params, fmt.Errorf("remove already used socket: %w", err)
		}
		if socket, err = shim.NewSocket(address); err != nil {
			return params, err
		}
	}
	f, err := socket.File()
	if err != nil {
		return params, err
	}
	cmd.ExtraFiles = append(cmd.ExtraFiles, f)

	if err := cmd.Start(); err != nil {
		return params, err
	}
	defer func() {
		if retErr != nil {
			_ = shim.RemoveSocket(address)
			cmd.Process.Kill()
		}
	}()
	// make sure to wait after start
	go cmd.Wait()
	if err := shim.WritePidFile("shim.pid", cmd.Process.Pid); err != nil {
		return params, err
	}

	params.Address = address
	return params, nil
}

// Stop is the fallback cleanup invoked via the `delete` binary call when
// containerd cannot reconnect to the shim.  It removes the jail and returns a
// synthetic exit status.  Stop should call runj delete but importantly _not_
// remove the shim's socket as that should happen when the shim is shut down.
func (m manager) Stop(ctx context.Context, id string) (shim.StopStatus, error) {
	if err := execKill(ctx, id, "KILL", true, 0); err != nil {
		log.G(ctx).WithError(err).Warn("failed to runj kill")
	}
	if err := execDelete(ctx, id); err != nil {
		log.G(ctx).WithError(err).Warn("failed to runj delete")
	}
	return shim.StopStatus{
		ExitedAt:   time.Now(),
		ExitStatus: 128 + int(unix.SIGKILL),
	}, nil
}

// Info returns runtime information for the shim.
func (m manager) Info(ctx context.Context, optionsR io.Reader) (*types.RuntimeInfo, error) {
	return &types.RuntimeInfo{Name: m.name}, nil
}

// newReexec creates a new exec.Cmd for running the shim API
func newReexec(ctx context.Context, id, containerdAddress string) (*exec.Cmd, error) {
	ns, err := namespaces.NamespaceRequired(ctx)
	if err != nil {
		return nil, err
	}
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	args := []string{
		"-namespace", ns,
		"-id", id,
		"-address", containerdAddress,
	}
	cmd := exec.Command(self, args...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "GOMAXPROCS=2")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Ensure a new process group is used so signals are not propagated by a calling shell
		Setpgid: true,
		Pgid:    0,
	}
	return cmd, nil
}
