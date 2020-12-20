package containerd

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/containerd/containerd/api/events"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/runtime"
	"github.com/containerd/containerd/runtime/v2/shim"
	"github.com/containerd/containerd/runtime/v2/task"
	taskAPI "github.com/containerd/containerd/runtime/v2/task"
	"github.com/gogo/protobuf/types"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NewService creates a new runj service and returns it as a shim.Shim
func NewService(ctx context.Context, id string, publisher shim.Publisher, shutdown func()) (shim.Shim, error) {
	s := &service{
		id:      id,
		context: ctx,
		cancel:  shutdown,
		events:  make(chan interface{}, 128),
	}

	if address, err := shim.ReadAddress("address"); err == nil {
		s.shimAddress = address
	}
	go s.forward(ctx, publisher)
	return s, nil
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

// check to make sure the *service implements the GRPC API
var (
	_     taskAPI.TaskService = (*service)(nil)
	empty                     = &ptypes.Empty{}
)

type service struct {
	id          string
	context     context.Context
	cancel      func()
	events      chan interface{}
	shimAddress string
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

// Cleanup is called to clean any remaining resources for the container. Cleanup
// should call runj delete but importantly _not_ remove the shim's socket as
// that should happen in Shutdown.
func (s *service) Cleanup(ctx context.Context) (*task.DeleteResponse, error) {
	log.G(ctx).Warn("CLEANUP")
	return nil, errdefs.ErrNotImplemented
}

func (s *service) State(ctx context.Context, req *task.StateRequest) (*task.StateResponse, error) {
	log.G(ctx).WithField("req", req).Warn("STATE")
	return nil, errdefs.ErrNotImplemented
}

func (s *service) Create(ctx context.Context, req *task.CreateTaskRequest) (*task.CreateTaskResponse, error) {
	log.G(ctx).WithField("req", req).Warn("CREATE")
	return nil, errdefs.ErrNotImplemented
}

func (s *service) Start(ctx context.Context, req *task.StartRequest) (*task.StartResponse, error) {
	log.G(ctx).WithField("req", req).Warn("START")
	return nil, errdefs.ErrNotImplemented
}

func (s *service) Delete(ctx context.Context, req *task.DeleteRequest) (*task.DeleteResponse, error) {
	log.G(ctx).WithField("req", req).Warn("DELETE")
	return nil, errdefs.ErrNotImplemented
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

func (s *service) Kill(ctx context.Context, req *task.KillRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("KILL")
	return nil, errdefs.ErrNotImplemented
}

func (s *service) Exec(ctx context.Context, req *task.ExecProcessRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("EXEC")
	return nil, errdefs.ErrNotImplemented
}

func (s *service) ResizePty(ctx context.Context, req *task.ResizePtyRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("RESIZEPTY")
	return nil, errdefs.ErrNotImplemented
}

func (s *service) CloseIO(ctx context.Context, req *task.CloseIORequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("CLOSEIO")
	return nil, errdefs.ErrNotImplemented
}

func (s *service) Update(ctx context.Context, req *task.UpdateTaskRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("UPDATE")
	return nil, errdefs.ErrNotImplemented
}

func (s *service) Wait(ctx context.Context, req *task.WaitRequest) (*task.WaitResponse, error) {
	log.G(ctx).WithField("req", req).Warn("WAIT")
	return nil, errdefs.ErrNotImplemented
}

func (s *service) Stats(ctx context.Context, req *task.StatsRequest) (*task.StatsResponse, error) {
	log.G(ctx).WithField("req", req).Warn("STATS")
	return nil, errdefs.ErrNotImplemented
}

func (s *service) Connect(ctx context.Context, req *task.ConnectRequest) (*task.ConnectResponse, error) {
	log.G(ctx).WithField("req", req).Warn("CONNECT")
	return nil, errdefs.ErrNotImplemented
}
