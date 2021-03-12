package containerd

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"

	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/sys/reaper"
)

// execCreate runs the "create" subcommand for runj
func execCreate(ctx context.Context, id, bundle string) error {
	cmd := exec.CommandContext(ctx, "runj", "create", id, bundle)
	ec, err := reaper.Default.Start(cmd)
	_, err = reaper.Default.Wait(cmd, ec)
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
	b, err := combinedOutput(cmd)
	if err != nil {
		log.G(ctx).
			WithError(err).
			WithField("output", string(b)).
			WithField("id", id).Error("runj state failed")
		return nil, err
	}
	s := &ociState{}
	err = json.Unmarshal(b, s)
	return s, err
}

// execDelete runs the "delete" subcommand for runj
func execDelete(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "runj", "delete", id)
	b, err := combinedOutput(cmd)
	if err != nil {
		log.G(ctx).WithError(err).WithField("output", string(b)).WithField("id", id).Error("runj delete failed")
		return err
	}
	return nil
}

// execKill runs the "kill" subcommand for runj
func execKill(ctx context.Context, id string, signal string, all bool) error {
	args := []string{"kill", id, signal}
	if all {
		args = append(args, "--all")
	}
	cmd := exec.CommandContext(ctx, "runj", args...)
	b, err := combinedOutput(cmd)
	if err != nil {
		log.G(ctx).WithError(err).WithField("output", string(b)).WithField("id", id).Error("runj kill failed")
		return err
	}
	return nil
}

// execStart runs the "start" subcommand for runj
func execStart(ctx context.Context, id string) error {
	cmd := exec.CommandContext(ctx, "runj", "start", id)
	b, err := combinedOutput(cmd)
	if err != nil {
		log.G(ctx).WithError(err).WithField("output", string(b)).WithField("id", id).Error("runj start failed")
		return err
	}
	return nil
}

func combinedOutput(cmd *exec.Cmd) ([]byte, error) {
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	ec, err := reaper.Default.Start(cmd)
	_, err = reaper.Default.Wait(cmd, ec)
	b := stdout.Bytes()
	return b, err
}
