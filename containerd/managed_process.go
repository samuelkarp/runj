package containerd

import (
	"errors"
	"sync"

	"go.sbk.wtf/runj/state"

	"github.com/containerd/console"
	runc "github.com/containerd/go-runc"
	specs "github.com/opencontainers/runtime-spec/specs-go"
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
	// pio is the process I/O.  Each non-nil stream should be closed when the
	// process exits
	pio stdio
	// con is the console for the process
	con console.Console

	// auxiliary processes have additional data not maintained for the primary
	// process

	// spec is the process specification for the auxiliary process
	spec *specs.Process
	// specfile is the file where the spec is serialized
	specfile string
	// state is the state of the auxiliary process
	state state.Status
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

// SetStdio stores the process IO streams
func (m *managedProcess) SetStdio(pio stdio) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pio = pio
	return nil
}

// GetStdio retrieves the process IO streams
func (m *managedProcess) GetStdio() stdio {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pio
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

func (m *managedProcess) SetSpec(spec *specs.Process) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.spec = spec
}

func (m *managedProcess) GetSpec() *specs.Process {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.spec
}

func (m *managedProcess) SetSpecfile(specfile string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.specfile = specfile
}

func (m *managedProcess) GetSpecfile() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.specfile
}

func (m *managedProcess) SetState(state state.Status) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state = state
}

func (m *managedProcess) GetState() state.Status {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.state
}
