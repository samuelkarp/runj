package jail

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"go.sbk.wtf/runj/state"
)

const (
	execFifoFilename = "exec.fifo"
	execSkipFifo     = "-"
	consoleSocketEnv = "__RUNJ_CONSOLE_SOCKET"
	stdioFdCount     = 3
)

// SetupEntrypoint starts a runj-entrypoint process, which is used to start
// processes inside the jail.
//
// When used to start the jail's init process, runj-entrypoint will later be
// signalled through `runj start` to run the specified program in the jail. This
// indirection is necessary so that the STDIO for `runj create` or the supplied
// console socket is directed to that process.
//
// When used to start a secondary process inside the jail, the waiting step is
// skipped and runj-entrypoint will immediately proceed to create the process
// as soon as STDIO is configured.
//
// Note: this API is unstable; expect it to change.
func SetupEntrypoint(id string, init bool, argv []string, env []string, consoleSocketPath string) (*exec.Cmd, error) {
	path := execSkipFifo
	if init {
		var err error
		path, err = createExecFifo(id)
		if err != nil {
			return nil, err
		}
	}
	args := append([]string{id, path}, argv...)
	cmd := exec.Command("runj-entrypoint", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env

	// the caller of runj will handle receiving the console master
	if consoleSocketPath != "" {
		conn, err := net.Dial("unix", consoleSocketPath)
		if err != nil {
			return nil, err
		}
		uc, ok := conn.(*net.UnixConn)
		if !ok {
			return nil, errors.New("casting to UnixConn failed")
		}
		consoleSocket, err := uc.File()
		if err != nil {
			return nil, err
		}
		cmd.ExtraFiles = append(cmd.ExtraFiles, consoleSocket)
		cmd.Env = append(cmd.Env,
			consoleSocketEnv+"="+strconv.Itoa(stdioFdCount+len(cmd.ExtraFiles)-1),
		)
	}

	return cmd, cmd.Start()
}

// ExecEntrypoint execs a runj-entrypoint process in order to start processes
// inside the jail.
//
// Note: this API is unstable; expect it to change.
func ExecEntrypoint(id string, argv []string, env []string, consoleSocketPath string) error {
	// the caller of runj will handle receiving the console master
	if consoleSocketPath != "" {
		conn, err := net.Dial("unix", consoleSocketPath)
		if err != nil {
			return err
		}
		uc, ok := conn.(*net.UnixConn)
		if !ok {
			return errors.New("casting to UnixConn failed")
		}
		consoleSocket, err := uc.File()
		if err != nil {
			return err
		}
		fd, err := unix.Dup(int(consoleSocket.Fd()))
		if err != nil {
			return err
		}
		env = append(env, consoleSocketEnv+"="+strconv.Itoa(fd))
	}
	args := append([]string{"runj-entrypoint", id, execSkipFifo}, argv...)
	return unix.Exec("/usr/local/bin/runj-entrypoint", args, env)
}

// CleanupEntrypoint sends a SIGTERM to the PID recorded in the state file.
// This function returns with no error even if the process is not running or
// cannot be signaled.
func CleanupEntrypoint(id string) error {
	s, err := state.Load(id)
	if err != nil {
		return err
	}
	if s.PID == 0 {
		return nil
	}
	e, _ := os.FindProcess(s.PID)
	e.Signal(syscall.SIGTERM)
	return nil
}

// inspired by runc

// createExecFifo creates a fifo for communication between runj and
// runj-entrypoint.
// See runc/libcontainer/container_linux.go for a similar example
func createExecFifo(id string) (string, error) {
	path := fifoPath(id)
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("fifo: exec fifo %s already exists", path)
	}
	// umask??
	if err := unix.Mkfifo(path, 0622); err != nil {
		return "", err
	}
	return path, nil
}

func fifoPath(id string) string {
	return filepath.Join(state.Dir(id), execFifoFilename)
}

// AwaitFifoOpen waits for a runj-entrypoint process to open the fifo passed to
// it.  The fifo is used to indicate when runj-entrypoint should start the
// process inside the jail.
func AwaitFifoOpen(ctx context.Context, id string) error {
	type openResult struct {
		file *os.File
		err  error
	}
	fifoOpened := make(chan openResult)
	go func() {
		f, err := fifoOpen(fifoPath(id))
		fifoOpened <- openResult{f, err}
		close(fifoOpened)
	}()
	select {
	case result := <-fifoOpened:
		if result.err != nil {
			return result.err
		}
		return handleFifoResult(result.file)
	case <-ctx.Done():
		return errors.New("fifo: timed out")
	}
}

func fifoOpen(path string) (*os.File, error) {
	flags := os.O_RDONLY
	f, err := os.OpenFile(path, flags, 0)
	if err != nil {
		return nil, errors.Wrap(err, "fifo: open exec fifo for reading")
	}
	return f, nil
}

func handleFifoResult(f *os.File) error {
	defer f.Close()
	if err := readFromExecFifo(f); err != nil {
		return err
	}
	return os.Remove(f.Name())
}

func readFromExecFifo(execFifo io.Reader) error {
	data, err := ioutil.ReadAll(execFifo)
	if err != nil {
		return err
	}
	if len(data) <= 0 {
		return errors.New("cannot start an already running container")
	}
	return nil
}

// end
