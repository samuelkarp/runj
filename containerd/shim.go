package containerd

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"go.sbk.wtf/runj/state"

	"github.com/containerd/console"
	"github.com/containerd/containerd/api/events"
	tasktypes "github.com/containerd/containerd/api/types/task"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/pkg/process"
	"github.com/containerd/containerd/runtime"
	"github.com/containerd/containerd/runtime/v2/shim"
	"github.com/containerd/containerd/runtime/v2/task"
	taskAPI "github.com/containerd/containerd/runtime/v2/task"
	"github.com/containerd/containerd/sys/reaper"
	runc "github.com/containerd/go-runc"
	"github.com/containerd/typeurl"
	"github.com/gogo/protobuf/types"
	ptypes "github.com/gogo/protobuf/types"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// NewService creates a new runj service and returns it as a shim.Shim
func NewService(ctx context.Context, id string, publisher shim.Publisher, shutdown func()) (shim.Shim, error) {
	s := &service{
		id:      id,
		context: ctx,
		cancel:  shutdown,
		events:  make(chan interface{}, 128),
		// subscribe to the reaper to receive process exit information
		exits: reaper.Default.Subscribe(),
		primary: managedProcess{
			waitblock: make(chan struct{}, 0),
		},
		auxiliary: make(map[string]*managedProcess),
	}

	if address, err := shim.ReadAddress("address"); err == nil {
		s.shimAddress = address
	}

	// register the shim as a reaper so that it receives exit events for all (orphaned) descendent processes and can
	// wait on their results
	SetupReaperSignals(ctx, log.G(ctx).WithField("id", id))
	go s.processExits()

	go s.forward(ctx, publisher)
	return s, nil
}

// processExits handles exits for child processes inside the jail
func (s *service) processExits() {
	for e := range s.exits {
		log.G(s.context).WithField("pid", e.Pid).Warn("PROCESSING EXIT!")
		s.checkProcesses(e)
	}
}

// checkProcesses records exit data for processes inside the jail.  The only
// process currently handled is the init/main process.
func (s *service) checkProcesses(e runc.Exit) {
	proc, id := s.findProcess(e.Pid)
	if proc == nil {
		return
	}
	if id == "" {
		log.G(s.context).WithField("pid", e.Pid).Warn("INIT EXITED!")
	} else {
		log.G(s.context).WithField("pid", e.Pid).Warn("AUX EXITED!")
	}

	if id == "" {
		// When the primary process (which has no id) exits, all children should be killed
		err := execKill(s.context, s.id, "KILL", true, 0)
		if err != nil {
			logrus.WithError(err).WithField("id", s.id).Error("failed to kill init's children")
		}
	}
	proc.SetState(state.StatusStopped)
	proc.SetExited(e)
	s.sendL(&events.TaskExit{
		ContainerID: s.id,
		ID:          id,
		Pid:         uint32(e.Pid),
		ExitStatus:  uint32(e.Status),
		ExitedAt:    e.Timestamp,
	})
	proc.GetStdio().Close()
	// indicate that results are now ready for any pending Wait calls
	close(proc.waitblock)
}

func (s *service) findProcess(pid int) (*managedProcess, string) {
	primaryPid := s.primary.GetPID()
	if pid == primaryPid {
		return &s.primary, ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, aux := range s.auxiliary {
		if aux.GetPID() == pid {
			return aux, id
		}
	}
	return nil, ""
}

// forward forwards events to the shim.Publisher
func (s *service) forward(ctx context.Context, publisher shim.Publisher) {
	ns, _ := namespaces.Namespace(ctx)
	ctx = namespaces.WithNamespace(context.Background(), ns)
	for e := range s.events {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := publisher.Publish(ctx, mapTopic(e), e)
		cancel()
		if err != nil {
			logrus.WithError(err).Error("post event")
		}
	}
	publisher.Close()
}

// mapTopic converts an event from an interface type to the specific
// event topic id
func mapTopic(e interface{}) string {
	switch e.(type) {
	case *events.TaskCreate:
		return runtime.TaskCreateEventTopic
	case *events.TaskStart:
		return runtime.TaskStartEventTopic
	case *events.TaskOOM:
		return runtime.TaskOOMEventTopic
	case *events.TaskExit:
		return runtime.TaskExitEventTopic
	case *events.TaskDelete:
		return runtime.TaskDeleteEventTopic
	case *events.TaskExecAdded:
		return runtime.TaskExecAddedEventTopic
	case *events.TaskExecStarted:
		return runtime.TaskExecStartedEventTopic
	case *events.TaskPaused:
		return runtime.TaskPausedEventTopic
	case *events.TaskResumed:
		return runtime.TaskResumedEventTopic
	case *events.TaskCheckpointed:
		return runtime.TaskCheckpointedEventTopic
	default:
		logrus.Warnf("no topic for type %#v", e)
	}
	return runtime.TaskUnknownTopic
}

var (
	// check to make sure the *service implements the GRPC API
	_ taskAPI.TaskService = (*service)(nil)

	// empty is an empty return value
	empty = &ptypes.Empty{}
)

type service struct {
	id          string
	context     context.Context
	cancel      func()
	events      chan interface{}
	eventSendMu sync.Mutex
	shimAddress string
	exits       chan runc.Exit

	mu         sync.Mutex
	bundlePath string
	// primary is the primary process for the jail.  The lifetime of the jail
	// is tied to this process.
	primary managedProcess
	// auxiliary is a map of additional processes that run in the jail.
	auxiliary map[string]*managedProcess
}

// StartShim is called whenever a new container is created.  The role of the
// function is to return a domain socket address where the shim can be reached
// for further API calls.  This allows the shim logic to decide how many shims
// are in-use: one per container, one per machine, one per group of containers,
// or some other decision.  When this function returns, the current process
// exits.  If there is no existing shim with an address to use, this function
// must fork a new shim process.
func (s *service) StartShim(ctx context.Context, id, containerdBinary, containerdAddress, containerdTTRPCAddress string) (string, error) {
	cmd, err := newReexec(ctx, id, containerdAddress)
	if err != nil {
		return "", err
	}

	address, err := shim.SocketAddress(ctx, containerdAddress, id)
	if err != nil {
		return "", err
	}
	socket, err := shim.NewSocket(address)
	if err != nil {
		if !shim.SocketEaddrinuse(err) {
			return "", err
		}
		if err := shim.RemoveSocket(address); err != nil {
			return "", errors.Wrap(err, "remove already used socket")
		}
		if socket, err = shim.NewSocket(address); err != nil {
			return "", err
		}
	}
	f, err := socket.File()
	if err != nil {
		return "", err
	}
	cmd.ExtraFiles = append(cmd.ExtraFiles, f)

	if err := cmd.Start(); err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			_ = shim.RemoveSocket(address)
			cmd.Process.Kill()
		}
	}()
	// make sure to wait after start
	go cmd.Wait()
	if err := shim.WritePidFile("shim.pid", cmd.Process.Pid); err != nil {
		return "", err
	}
	if err := shim.WriteAddress("address", address); err != nil {
		return "", err
	}

	return address, nil
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

// Shutdown is called to allow the shim to exit.  Shutdown deletes resources
// like the socket address and must cause the shim.Publisher to be closed so the
// process exits.
func (s *service) Shutdown(ctx context.Context, req *task.ShutdownRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("SHUTDOWN")
	s.cancel()
	// shim.Publisher is closed after all events in s.events are processed
	close(s.events)
	if address, err := shim.ReadAddress("address"); err == nil {
		_ = shim.RemoveSocket(address)
	}
	return empty, nil
}

// Cleanup is called to clean any remaining resources for the container and is
// called through the `delete` subcommand rather than over ttrpc if containerd
// is unable to reconnect to the shim. Cleanup should call runj delete but
// importantly _not_ remove the shim's socket as that should happen in Shutdown.
// Cleanup is a binary call that cleans up any resources used by the shim when
// the service crashes; it is a fallback of Delete.
func (s *service) Cleanup(ctx context.Context) (*task.DeleteResponse, error) {
	opts, ok := ctx.Value(shim.OptsKey{}).(shim.Opts)
	if !ok {
		return nil, errors.New("failed to read opts")
	}
	return s.delete(ctx, opts.BundlePath)
}

// Delete a process or container.  When deleting a container, Delete should call
// runj delete but importantly _not_ remove the shim's socket as that should
// happen in Shutdown.
func (s *service) Delete(ctx context.Context, req *task.DeleteRequest) (*task.DeleteResponse, error) {
	log.G(ctx).WithField("req", req).Warn("DELETE")
	if req.ID != s.id {
		log.G(ctx).WithField("reqID", req.ID).WithField("id", s.id).Error("mismatched IDs")
		return nil, errdefs.ErrInvalidArgument
	}
	if req.ExecID != "" {
		return s.deleteAux(ctx, req.ExecID)
	}
	path := s.getBundlePath()
	if path == "" {
		log.G(ctx).Error("bundle path missing")
		return nil, errdefs.ErrFailedPrecondition
	}

	return s.delete(ctx, path)
}

// delete performs work that is common between Cleanup and Delete.
func (s *service) delete(ctx context.Context, bundlePath string) (*task.DeleteResponse, error) {
	if err := execKill(ctx, s.id, "KILL", true, 0); err != nil {
		log.G(ctx).WithError(err).Error("failed to run runj kill --all")
		return nil, err
	}
	if err := execDelete(ctx, s.id); err != nil {
		log.G(ctx).WithError(err).Error("failed to run runj delete")
		return nil, err
	}
	if err := mount.UnmountAll(filepath.Join(bundlePath, "rootfs"), 0); err != nil {
		log.G(ctx).WithError(err).Warn("failed to cleanup rootfs mount")
	}
	return &taskAPI.DeleteResponse{
		ExitedAt:   time.Now(),
		ExitStatus: 128 + uint32(unix.SIGKILL),
	}, nil
}

func (s *service) deleteAux(ctx context.Context, id string) (*task.DeleteResponse, error) {
	log.G(ctx).WithField("execID", id).Warn("Delete Exec!")
	aux := s.getAuxiliary(id)
	if aux == nil {
		log.G(ctx).WithField("execID", id).Debug("delete: exec process not found")
		return nil, errdefs.ErrNotFound
	}
	auxState := aux.GetState()
	switch auxState {
	case state.StatusRunning:
		// process must not be running
		return nil, errdefs.ErrInvalidArgument
	case state.StatusCreating, state.StatusCreated:
		close(aux.waitblock)
	}

	if specfile := aux.GetSpecfile(); specfile != "" {
		os.Remove(specfile)
		aux.SetSpecfile("")
	}
	s.deleteAuxiliary(id)
	return &taskAPI.DeleteResponse{
		ExitedAt:   time.Now(),
		ExitStatus: 128 + uint32(unix.SIGKILL),
	}, nil
}

// Create sets up the OCI bundle and invokes runj create
func (s *service) Create(ctx context.Context, req *task.CreateTaskRequest) (*task.CreateTaskResponse, error) {
	log.G(ctx).WithField("req", req).Warn("CREATE")
	if req.ID != s.id {
		log.G(ctx).WithField("reqID", req.ID).WithField("id", s.id).Error("mismatched IDs")
		return nil, errdefs.ErrInvalidArgument
	}
	s.setBundlePath(req.Bundle)

	var mounts []process.Mount
	for _, m := range req.Rootfs {
		mounts = append(mounts, process.Mount{
			Type:    m.Type,
			Source:  m.Source,
			Target:  m.Target,
			Options: m.Options,
		})
	}

	rootfs := ""
	if len(mounts) > 0 {
		log.G(ctx).WithField("mounts", mounts).Warn("mkdir rootfs")
		rootfs = filepath.Join(req.Bundle, "rootfs")
		if err := os.Mkdir(rootfs, 0711); err != nil && !os.IsExist(err) {
			return nil, err
		}
	}
	var err error
	defer func() {
		if err != nil {
			log.G(ctx).WithField("rootfs", rootfs).WithError(err).Error("failed to create,unmounting rootfs")
			if err2 := mount.UnmountAll(rootfs, 0); err2 != nil {
				log.G(ctx).WithError(err2).Warn("failed to cleanup rootfs mount")
			}
		}
	}()
	for _, rm := range mounts {
		m := &mount.Mount{
			Type:    rm.Type,
			Source:  rm.Source,
			Options: rm.Options,
		}
		log.G(ctx).WithField("mount", m).WithField("rootfs", rootfs).Warn("mount")
		if err := m.Mount(rootfs); err != nil {
			return nil, errors.Wrapf(err, "failed to mount rootfs component %v", m)
		}
	}

	var pio stdio
	pio, err = setupIO(ctx, req.Stdin, req.Stdout, req.Stderr)
	defer func() {
		if err == nil {
			return
		}
		pio.Close()
	}()
	if err != nil {
		return nil, err
	}

	con, err := execCreate(ctx, req.ID, req.Bundle, pio.stdin, pio.stdout, pio.stderr, req.Terminal)
	if err != nil {
		log.G(ctx).WithError(err).Error("failed to create jail")
		return nil, err
	}
	s.primary.SetStdio(pio)
	s.primary.SetConsole(con)

	ociState, err := execState(ctx, req.ID)
	if err != nil {
		log.G(ctx).WithError(err).Error("failed to get jail state")
		return nil, err
	}

	log.G(ctx).WithField("pid", ociState.PID).WithField("state", ociState).Warn("entrypoint waiting!")
	s.primary.SetPID(ociState.PID)

	s.sendL(&events.TaskCreate{
		ContainerID: req.ID,
		Bundle:      req.Bundle,
		Rootfs:      req.Rootfs,
		Pid:         uint32(ociState.PID),
	})

	return &taskAPI.CreateTaskResponse{
		Pid: uint32(ociState.PID),
	}, nil
}

func (s *service) setBundlePath(bundlePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.bundlePath != "" && s.bundlePath != bundlePath {
		return errors.New("cannot re-set bundlePath to different value")
	}
	s.bundlePath = bundlePath
	return nil
}

func (s *service) getBundlePath() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.bundlePath
}

func (s *service) getAuxiliary(id string) *managedProcess {
	s.mu.Lock()
	defer s.mu.Unlock()

	if aux, ok := s.auxiliary[id]; ok {
		return aux
	}
	return nil
}

func (s *service) newAuxiliary(id string) *managedProcess {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.auxiliary[id]; ok {
		return nil
	}

	s.auxiliary[id] = &managedProcess{
		waitblock: make(chan struct{}, 0),
	}
	return s.auxiliary[id]
}

func (s *service) setAuxiliary(id string, aux *managedProcess) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.auxiliary[id] = aux
}

func (s *service) deleteAuxiliary(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.auxiliary, id)
}

// sendUnsafe sends an event without acquiring the event lock
func (s *service) sendUnsafe(evt interface{}) {
	s.events <- evt
}

// sendL acquires the event lock and then sends an event
func (s *service) sendL(evt interface{}) {
	s.eventSendMu.Lock()
	defer s.eventSendMu.Unlock()

	s.events <- evt
}

// State returns the state of the container.  The returned state is a composite
// of the state information from the underlying container and information that
// was recorded by this shim (exit information from the primary process).
func (s *service) State(ctx context.Context, req *task.StateRequest) (*task.StateResponse, error) {
	log.G(ctx).WithField("req", req).Warn("STATE")
	if req.ExecID != "" {
		return s.stateAux(ctx, req.ExecID)
	}
	if req.ID != s.id {
		log.G(ctx).WithField("reqID", req.ID).WithField("id", s.id).Error("mismatched IDs")
		return nil, errdefs.ErrInvalidArgument
	}
	bundlePath := s.getBundlePath()
	ociState, err := execState(ctx, s.id)
	if err != nil {
		return nil, err
	}
	resp := &task.StateResponse{
		ID:     s.id,
		Bundle: bundlePath,
		Pid:    uint32(ociState.PID),
		Status: runjStatusToContainerdStatus(ociState.Status),
	}
	log.G(ctx).WithField("state", ociState).WithField("resp", resp).Warn("STATE")
	if resp.Status == tasktypes.StatusStopped {
		exit := s.primary.GetExited()
		resp.ExitedAt = exit.Timestamp
		resp.ExitStatus = uint32(exit.Status)
	}
	return resp, nil
}

func (s *service) stateAux(ctx context.Context, id string) (*task.StateResponse, error) {
	log.G(ctx).WithField("execID", id).Error("Exec state!")
	aux := s.getAuxiliary(id)
	if aux == nil {
		return nil, errdefs.ErrNotFound
	}
	exit := aux.GetExited()
	return &task.StateResponse{
		ID:         s.id,
		ExecID:     id,
		Pid:        uint32(aux.GetPID()),
		Status:     runjStatusToContainerdStatus(string(aux.GetState())),
		ExitedAt:   exit.Timestamp,
		ExitStatus: uint32(exit.Status),
	}, nil
}

func runjStatusToContainerdStatus(in string) tasktypes.Status {
	switch state.Status(in) {
	case state.StatusCreating:
		return tasktypes.StatusUnknown
	case state.StatusCreated:
		return tasktypes.StatusCreated
	case state.StatusRunning:
		return tasktypes.StatusRunning
	case state.StatusStopped:
		return tasktypes.StatusStopped
	}
	return tasktypes.StatusUnknown
}

// Start is responsible for starting processes inside a container.  Start can be
// used to either start the container's primary process (previously specified
// with Create) or a secondary process (previously specified with Exec).  When
// used for the primary process, Start invokes "runj start".  When used for a
// secondary process, Start invokes "runj exec".
func (s *service) Start(ctx context.Context, req *task.StartRequest) (*task.StartResponse, error) {
	log.G(ctx).WithField("req", req).Warn("START")
	if req.ID != s.id {
		log.G(ctx).WithField("reqID", req.ID).WithField("id", s.id).Error("mismatched IDs")
		return nil, errdefs.ErrInvalidArgument
	}
	if req.ExecID == "" {
		return s.startPrimary(ctx, s.id)
	}
	return s.startAux(ctx, s.id, req.ExecID)
}

func (s *service) startPrimary(ctx context.Context, id string) (*task.StartResponse, error) {
	ociState, err := execState(ctx, id)
	if err != nil {
		return nil, err
	}
	log.G(ctx).WithField("state", ociState).Warn("START")
	// hold the sendUnsafe lock so that the start events are sent before any exit events in the error case
	s.eventSendMu.Lock()
	defer s.eventSendMu.Unlock()
	err = execStart(ctx, s.id)
	if err != nil {
		return nil, err
	}
	log.G(ctx).WithField("state", ociState).Warn("START runj")

	s.sendUnsafe(&events.TaskStart{
		ContainerID: s.id,
		Pid:         uint32(ociState.PID),
	})
	return &task.StartResponse{
		Pid: uint32(ociState.PID),
	}, nil
}

func (s *service) startAux(ctx context.Context, id, execID string) (*task.StartResponse, error) {
	proc := s.getAuxiliary(execID)
	if proc == nil {
		return nil, errdefs.ErrNotFound
	}
	if proc.GetState() != state.StatusCreated {
		return nil, errdefs.ErrInvalidArgument
	}
	log.G(ctx).WithField("execID", execID).Warn("START EXEC")
	// hold the sendUnsafe lock so that the start events are sent before any exit events in the error case
	s.eventSendMu.Lock()
	defer s.eventSendMu.Unlock()
	pio := proc.GetStdio()

	pid, err := execExec(ctx, id, proc.GetSpecfile(), pio.stdin, pio.stdout, pio.stderr)
	log.G(ctx).WithField("execID", execID).WithError(err).Warn("START EXEC runj")
	if err != nil {
		proc.SetState(state.StatusStopped)
		close(proc.waitblock)
		return nil, err
	}
	proc.SetPID(pid)
	proc.SetState(state.StatusRunning)

	s.sendUnsafe(&events.TaskStart{
		ContainerID: id,
		Pid:         uint32(pid),
	})
	return &task.StartResponse{
		Pid: uint32(pid),
	}, nil
}

func (s service) Pids(ctx context.Context, req *task.PidsRequest) (*task.PidsResponse, error) {
	log.G(ctx).WithField("req", req).Warn("PIDS")
	return nil, errdefs.ErrNotImplemented
}

func (s *service) Pause(ctx context.Context, req *task.PauseRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("PAUSE")
	return nil, errdefs.ErrNotImplemented
}

func (s *service) Resume(ctx context.Context, req *task.ResumeRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("RESUME")
	return nil, errdefs.ErrNotImplemented
}

func (s *service) Checkpoint(ctx context.Context, req *task.CheckpointTaskRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("CHECKPOINT")
	return nil, errdefs.ErrNotImplemented
}

// Kill sends signals to processes inside a container
func (s *service) Kill(ctx context.Context, req *task.KillRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("KILL")
	pid := 0
	if req.ExecID != "" {
		log.G(ctx).WithField("execID", req.ExecID).Error("Exec kill aux!")
		aux := s.getAuxiliary(req.ExecID)
		if aux == nil {
			return nil, errdefs.ErrNotFound
		}
		if aux.GetState() != state.StatusRunning {
			return nil, errdefs.ErrInvalidArgument
		}
		pid = aux.GetPID()
		if pid == 0 {
			return nil, errdefs.ErrInvalidArgument
		}
	}
	if req.ID != s.id {
		log.G(ctx).WithField("reqID", req.ID).WithField("id", s.id).Error("mismatched IDs")
		return nil, errdefs.ErrInvalidArgument
	}
	err := execKill(ctx, s.id, strconv.FormatUint(uint64(req.Signal), 10), req.All, pid)
	return nil, err
}

// Exec sets up a new secondary process that should be run in the container, but
// does not start the process.  After calling Exec to set up the process
// (including its args, environment, and IO), call Start to start it.
func (s *service) Exec(ctx context.Context, req *task.ExecProcessRequest) (*types.Empty, error) {
	l := log.G(ctx).WithField("id", req.ID).WithField("execID", req.ExecID)
	l.WithField("req", req).Warn("EXEC")
	specAny, err := typeurl.UnmarshalAny(req.Spec)
	if err != nil {
		l.WithError(err).Error("failed to unmarshal spec")
		return nil, errdefs.ErrInvalidArgument
	}
	spec, ok := specAny.(*specs.Process)
	if !ok {
		l.Error("mismatched type for spec")
		return nil, errdefs.ErrInvalidArgument
	}
	l.WithField("spec", spec).Warn("EXEC")
	aux := s.newAuxiliary(req.ExecID)
	if aux == nil {
		return nil, errdefs.ErrAlreadyExists
	}
	aux.SetSpec(spec)
	aux.SetState(state.StatusCreated)

	var pio stdio
	pio, err = setupIO(ctx, req.Stdin, req.Stdout, req.Stderr)
	defer func() {
		if err == nil {
			return
		}
		pio.Close()
	}()
	if err != nil {
		return nil, err
	}
	aux.SetStdio(pio)

	f, err := ioutil.TempFile("", "runj-process")
	if err != nil {
		return nil, err
	}
	err = json.NewEncoder(f).Encode(spec)
	f.Close()
	if err != nil {
		return nil, err
	}
	aux.SetSpecfile(f.Name())

	s.sendL(&events.TaskExecAdded{
		ContainerID: s.id,
		ExecID:      req.ExecID,
	})
	return empty, nil
}

func (s *service) ResizePty(ctx context.Context, req *task.ResizePtyRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("RESIZEPTY")
	if req.ExecID != "" {
		log.G(ctx).WithField("execID", req.ExecID).Error("Exec not implemented!")
		return nil, errdefs.ErrNotImplemented
	}
	if req.ID != s.id {
		log.G(ctx).WithField("reqID", req.ID).WithField("id", s.id).Error("mismatched IDs")
		return nil, errdefs.ErrInvalidArgument
	}
	con := s.primary.GetConsole()
	if con == nil {
		return nil, errdefs.ErrUnavailable
	}
	if err := con.Resize(console.WinSize{
		Width:  uint16(req.Width),
		Height: uint16(req.Height),
	}); err != nil {
		return nil, errdefs.ToGRPC(err)
	}
	return empty, nil
}

func (s *service) CloseIO(ctx context.Context, req *task.CloseIORequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("CLOSEIO")
	return nil, errdefs.ErrNotImplemented
}

func (s *service) Update(ctx context.Context, req *task.UpdateTaskRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("UPDATE")
	return nil, errdefs.ErrNotImplemented
}

// Wait blocks while the identified process is running and returns its exit code and exit timestamp when complete.
// The data for Wait (including the channel it uses as an indicator of when results are ready) is provided by the
// SIGCHLD handler, reaper, and subscribed goroutine.
func (s *service) Wait(ctx context.Context, req *task.WaitRequest) (*task.WaitResponse, error) {
	log.G(ctx).WithField("req", req).Warn("WAIT")
	l := log.G(ctx).WithField("reqID", req.ID).WithField("id", s.id)
	if req.ID != s.id {
		l.Error("mismatched IDs")
		return nil, errdefs.ErrInvalidArgument
	}
	proc := &s.primary
	if req.ExecID != "" {
		if proc = s.getAuxiliary(req.ExecID); proc == nil {
			l.Error("Cannot find aux process")
			return nil, errdefs.ErrNotFound
		}
	}
	// Only the init/main process of the jail is currently supported.  This logic will need to change for exec support.
	<-proc.waitblock
	e := proc.GetExited()
	l.WithField("pid", e.Pid).WithField("status", e.Status).Warn("Process exited")
	return &task.WaitResponse{
		ExitStatus: uint32(e.Status),
		ExitedAt:   e.Timestamp,
	}, nil
}

func (s *service) Stats(ctx context.Context, req *task.StatsRequest) (*task.StatsResponse, error) {
	log.G(ctx).WithField("req", req).Warn("STATS")
	return nil, errdefs.ErrNotImplemented
}

func (s *service) Connect(ctx context.Context, req *task.ConnectRequest) (*task.ConnectResponse, error) {
	log.G(ctx).WithField("req", req).Warn("CONNECT")
	return nil, errdefs.ErrNotImplemented
}
