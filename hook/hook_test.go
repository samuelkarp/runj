package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.sbk.wtf/runj/state"
)

func TestRunSuccess(t *testing.T) {
	s := &state.Output{ID: "container1", Status: "created"}
	h := &runtimespec.Hook{
		Path: "/bin/sh",
		Args: []string{"sh", "-c", "exit 0"},
	}
	assert.NoError(t, Run(s, h))
}

func TestRunFailureReturnsError(t *testing.T) {
	s := &state.Output{ID: "container1", Status: "created"}
	h := &runtimespec.Hook{
		Path: "/bin/sh",
		Args: []string{"sh", "-c", "exit 3"},
	}
	assert.Error(t, Run(s, h))
}

func TestRunPassesStateOnStdin(t *testing.T) {
	s := &state.Output{ID: "container1", Status: "running", PID: 4422, Bundle: "/bundle"}
	out := filepath.Join(t.TempDir(), "stdin")
	h := &runtimespec.Hook{
		Path: "/bin/sh",
		Args: []string{"sh", "-c", "cat > " + out},
	}
	require.NoError(t, Run(s, h))

	got, err := os.ReadFile(out)
	require.NoError(t, err)
	want, err := json.Marshal(s)
	require.NoError(t, err)
	assert.JSONEq(t, string(want), string(got))
}

func TestRunHonorsTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in short mode")
	}
	s := &state.Output{ID: "container1", Status: "created"}
	timeout := 1
	// Exec sleep directly (rather than via "sh -c") so the timeout's SIGKILL
	// lands on the process itself; otherwise a forked grandchild would inherit
	// the stdio pipes and delay Run's return until it exited on its own.
	h := &runtimespec.Hook{
		Path:    "/bin/sleep",
		Args:    []string{"sleep", "30"},
		Timeout: &timeout,
	}
	start := time.Now()
	err := Run(s, h)
	assert.Error(t, err, "hook exceeding its timeout must return an error")
	assert.Less(t, time.Since(start), 10*time.Second, "Run should return shortly after the timeout fires")
}
