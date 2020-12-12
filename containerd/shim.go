package containerd

import (
	"context"

	"github.com/containerd/containerd/runtime/v2/shim"
	"github.com/containerd/containerd/runtime/v2/task"
	"github.com/gogo/protobuf/types"
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
	panic("implement me")
}

func (s service) Create(ctx context.Context, req *task.CreateTaskRequest) (*task.CreateTaskResponse, error) {
	panic("implement me")
}

func (s service) Start(ctx context.Context, req *task.StartRequest) (*task.StartResponse, error) {
	panic("implement me")
}

func (s service) Delete(ctx context.Context, req *task.DeleteRequest) (*task.DeleteResponse, error) {
	panic("implement me")
}

func (s service) Pids(ctx context.Context, req *task.PidsRequest) (*task.PidsResponse, error) {
	panic("implement me")
}

func (s service) Pause(ctx context.Context, req *task.PauseRequest) (*types.Empty, error) {
	panic("implement me")
}

func (s service) Resume(ctx context.Context, req *task.ResumeRequest) (*types.Empty, error) {
	panic("implement me")
}

func (s service) Checkpoint(ctx context.Context, req *task.CheckpointTaskRequest) (*types.Empty, error) {
	panic("implement me")
}

func (s service) Kill(ctx context.Context, req *task.KillRequest) (*types.Empty, error) {
	panic("implement me")
}

func (s service) Exec(ctx context.Context, req *task.ExecProcessRequest) (*types.Empty, error) {
	panic("implement me")
}

func (s service) ResizePty(ctx context.Context, req *task.ResizePtyRequest) (*types.Empty, error) {
	panic("implement me")
}

func (s service) CloseIO(ctx context.Context, req *task.CloseIORequest) (*types.Empty, error) {
	panic("implement me")
}

func (s service) Update(ctx context.Context, req *task.UpdateTaskRequest) (*types.Empty, error) {
	panic("implement me")
}

func (s service) Wait(ctx context.Context, req *task.WaitRequest) (*task.WaitResponse, error) {
	panic("implement me")
}

func (s service) Stats(ctx context.Context, req *task.StatsRequest) (*task.StatsResponse, error) {
	panic("implement me")
}

func (s service) Connect(ctx context.Context, req *task.ConnectRequest) (*task.ConnectResponse, error) {
	panic("implement me")
}

func (s service) Shutdown(ctx context.Context, req *task.ShutdownRequest) (*types.Empty, error) {
	panic("implement me")
}

func (s service) Cleanup(ctx context.Context) (*task.DeleteResponse, error) {
	panic("implement me")
}

func (s service) StartShim(ctx context.Context, id, containerdBinary, containerdAddress, containerdTTRPCAddress string) (string, error) {
	panic("implement me")
}
