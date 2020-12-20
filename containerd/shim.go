package containerd

import (
	"context"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/runtime/v2/shim"
	"github.com/containerd/containerd/runtime/v2/task"
	"github.com/gogo/protobuf/types"
	"github.com/sirupsen/logrus"
)

func NewService(ctx context.Context, id string, publisher shim.Publisher, shutdown func()) (shim.Shim, error) {
	s := &service{
		id:      id,
		context: ctx,
		cancel:  shutdown,
	}

	if address, err := shim.ReadAddress("address"); err == nil {
		s.shimAddress = address
	}
	return s, nil
}

type service struct {
	id          string
	context     context.Context
	cancel      func()
	shimAddress string
}

func (s service) State(ctx context.Context, req *task.StateRequest) (*task.StateResponse, error) {
	log.G(ctx).WithField("req", req).Warn("STATE")
	return nil, errdefs.ErrNotImplemented
}

func (s service) Create(ctx context.Context, req *task.CreateTaskRequest) (*task.CreateTaskResponse, error) {
	log.G(ctx).WithField("req", req).Warn("CREATE")
	return nil, errdefs.ErrNotImplemented
}

func (s service) Start(ctx context.Context, req *task.StartRequest) (*task.StartResponse, error) {
	log.G(ctx).WithField("req", req).Warn("START")
	return nil, errdefs.ErrNotImplemented
}

func (s service) Delete(ctx context.Context, req *task.DeleteRequest) (*task.DeleteResponse, error) {
	log.G(ctx).WithField("req", req).Warn("DELETE")
	return nil, errdefs.ErrNotImplemented
}

func (s service) Pids(ctx context.Context, req *task.PidsRequest) (*task.PidsResponse, error) {
	log.G(ctx).WithField("req", req).Warn("PIDS")
	return nil, errdefs.ErrNotImplemented
}

func (s service) Pause(ctx context.Context, req *task.PauseRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("PAUSE")
	return nil, errdefs.ErrNotImplemented
}

func (s service) Resume(ctx context.Context, req *task.ResumeRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("RESUME")
	return nil, errdefs.ErrNotImplemented
}

func (s service) Checkpoint(ctx context.Context, req *task.CheckpointTaskRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("CHECKPOINT")
	return nil, errdefs.ErrNotImplemented
}

func (s service) Kill(ctx context.Context, req *task.KillRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("KILL")
	return nil, errdefs.ErrNotImplemented
}

func (s service) Exec(ctx context.Context, req *task.ExecProcessRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("EXEC")
	return nil, errdefs.ErrNotImplemented
}

func (s service) ResizePty(ctx context.Context, req *task.ResizePtyRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("RESIZEPTY")
	return nil, errdefs.ErrNotImplemented
}

func (s service) CloseIO(ctx context.Context, req *task.CloseIORequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("CLOSEIO")
	return nil, errdefs.ErrNotImplemented
}

func (s service) Update(ctx context.Context, req *task.UpdateTaskRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("UPDATE")
	return nil, errdefs.ErrNotImplemented
}

func (s service) Wait(ctx context.Context, req *task.WaitRequest) (*task.WaitResponse, error) {
	log.G(ctx).WithField("req", req).Warn("WAIT")
	return nil, errdefs.ErrNotImplemented
}

func (s service) Stats(ctx context.Context, req *task.StatsRequest) (*task.StatsResponse, error) {
	log.G(ctx).WithField("req", req).Warn("STATS")
	return nil, errdefs.ErrNotImplemented
}

func (s service) Connect(ctx context.Context, req *task.ConnectRequest) (*task.ConnectResponse, error) {
	log.G(ctx).WithField("req", req).Warn("CONNECT")
	return nil, errdefs.ErrNotImplemented
}

func (s service) Shutdown(ctx context.Context, req *task.ShutdownRequest) (*types.Empty, error) {
	log.G(ctx).WithField("req", req).Warn("SHUTDOWN")
	return nil, errdefs.ErrNotImplemented
}

func (s service) Cleanup(ctx context.Context) (*task.DeleteResponse, error) {
	log.G(ctx).Warn("CLEANUP")
	return nil, errdefs.ErrNotImplemented
}

func (s service) StartShim(ctx context.Context, id, containerdBinary, containerdAddress, containerdTTRPCAddress string) (string, error) {
	log.G(ctx).WithFields(logrus.Fields{
		"id":                     id,
		"containerdBinary":       containerdBinary,
		"containerdAddress":      containerdAddress,
		"containerdTTRPCAddress": containerdTTRPCAddress,
	}).Warn("STARTSHIM")
	return "", errdefs.ErrNotImplemented
}
