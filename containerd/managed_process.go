package containerd

import (
	"errors"
	"io"
	"sync"

	"github.com/containerd/console"
	runc "github.com/containerd/go-runc"
)

// managedProcess contains the state for a process that is managed by the runj
// shim.
type managedProcess struct {
	mu  sync.Mutex
	pid int
	// exit records exit details for the managed process
	exit runc.Exit
	// waitblock is a channel signaling the end of process execution
	waitblock chan struct{}
	// stdioFifo is a slice of io.Closer to close when the process exits
	stdioFifo []io.Closer
	// con is the console for the process
	con console.Console
}

// SetPID stores the PID of the process
func (m *managedProcess) SetPID(p int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pid != 0 && m.pid != p {
		return errors.New("cannot re-set pid to different value")
	}
	m.pid = p
	return nil
}

// GetPID retrieves the PID of the process
func (m *managedProcess) GetPID() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.pid
}

// SetExited records exit details of the process
func (m *managedProcess) SetExited(e runc.Exit) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.exit = e
}

// GetExited retrieves the exit details of the process
func (m *managedProcess) GetExited() runc.Exit {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.exit
}

// SetStdioFifo stores the io.Closers to be closed when the process exits
func (m *managedProcess) SetStdioFifo(stdio []io.Closer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.stdioFifo) > 0 {
		return errors.New("cannot re-set stdioFifo to different value")
	}
	m.stdioFifo = stdio
	return nil
}

// GetStdioFifo retrieves the io.Closers that should be closed when the process
// exits
func (m *managedProcess) GetStdioFifo() []io.Closer {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]io.Closer{}, m.stdioFifo...)
}

// SetConsole stores the console for the process
func (m *managedProcess) SetConsole(con console.Console) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.con = con
}

// GetConsole retrieves the console for the process
func (m *managedProcess) GetConsole() console.Console {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.con
}
