package containerd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os/exec"

	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/sys/reaper"
	runc "github.com/containerd/go-runc"
)

// execCreate runs the "create" subcommand for runj
func execCreate(ctx context.Context, id, bundle string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, "runj", "create", id, bundle)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	ec, err := reaper.Default.Start(cmd)
	_, err = WaitNoFlush(cmd, ec)
	if err != nil {
		log.G(ctx).WithError(err).WithField("id", id).Error("runj create failed")
	}
	return err
}

// WaitNoFlush waits for a process to exit but does not flush IO with cmd.Wait
func WaitNoFlush(c *exec.Cmd, ec chan runc.Exit) (int, error) {
	for e := range ec {
		if e.Pid == c.Process.Pid {
			reaper.Default.Unsubscribe(ec)
			return e.Status, nil
		}
	}
	// return no such process if the ec channel is closed and no more exit
	// events will be sent
	return -1, reaper.ErrNoSuchProcess
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