package containerd

import (
	"context"
	"encoding/json"
	"os/exec"

	"github.com/containerd/containerd/log"
)

// execCreate runs the "create" subcommand for runj
func execCreate(ctx context.Context, id, bundle string) error {
	cmd := exec.CommandContext(ctx, "runj", "create", id, bundle)
	err := cmd.Run()
	if err != nil {
		log.G(ctx).WithError(err).WithField("id", id).Error("runj create failed")
	}
	return err
}

type ociState struct {
	OCIVersion  string            `json:"ociVersion"`
	ID          string            `json:"id"`
	Status      string            `json:"status"`
	PID         int               `json:"pid,omitempty"`
	Bundle      string            `json:"bundle"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// execState runs the "state" subcommand for runj
func execState(ctx context.Context, id string) (*ociState, error) {
	cmd := exec.CommandContext(ctx, "runj", "state", id)
	b, err := cmd.Output()
	if err != nil {
		log.G(ctx).WithError(err).WithField("output", string(b)).WithField("id", id).Error("runj state failed")
		return nil, err
	}
	s := &ociState{}
	err = json.Unmarshal(b, s)
	return s, err
}

// execDelete runs the "delete" subcommand for runj
func execDelete(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "runj", "delete", id)
	b, err := cmd.CombinedOutput()
	if err != nil {
		log.G(ctx).WithError(err).WithField("output", string(b)).WithField("id", id).Error("runj delete failed")
		return err
	}
	return nil
}
