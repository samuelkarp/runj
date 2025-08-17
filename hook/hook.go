package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"go.sbk.wtf/runj/state"
)

// Run runs a given hook
func Run(s *state.Output, h *runtimespec.Hook) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	var stdout, stderr bytes.Buffer
	var cancelFunc context.CancelFunc
	ctx := context.Background()

	if h.Timeout != nil {
		ctx, cancelFunc = context.WithTimeout(ctx, time.Duration(*h.Timeout)*time.Second)
		defer cancelFunc()
	}

	cmd := exec.CommandContext(ctx, h.Path, h.Args[1:]...)
	cmd.Env = h.Env
	cmd.Stdin = bytes.NewReader(b)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running hook: %v, stdout: %s, stderr: %s\n", err, stdout.String(), stderr.String())
	}

	return err
}
