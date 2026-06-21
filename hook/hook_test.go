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

// TestRunTimeoutKillsForkedChild is a regression test for the hook timeout not
// bounding wall-clock time.  The hook forks a long-lived grandchild ("sleep"
// behind a compound command so the shell does not exec into it) that inherits
// the stdio pipes.  Without process-group kill, the SIGKILL on timeout reaps
// only the shell, leaving the grandchild holding the pipes and blocking Run for
// the full sleep duration.
func TestRunTimeoutKillsForkedChild(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in short mode")
	}
	timeout := 1
	h := &runtimespec.Hook{
		Path:    "/bin/sh",
		Args:    []string{"sh", "-c", "sleep 30; true"},
		Timeout: &timeout,
	}
	start := time.Now()
	err := Run(&state.Output{ID: "container1", Status: "created"}, h)
	elapsed := time.Since(start)
	assert.Error(t, err, "hook exceeding its timeout must return an error")
	assert.Less(t, elapsed, 10*time.Second, "Run must return shortly after the timeout, not after the child exits")
}
