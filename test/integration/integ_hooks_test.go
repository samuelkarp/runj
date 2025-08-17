//go:build integration
// +build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
)

func TestHooks(t *testing.T) {
	spec := setupSimpleExitingJail(t)
	dir := t.TempDir()

	spec.Process = &runtimespec.Process{
		Args: []string{"/integ-inside", "-test.v", "-test.run", "TestHello"},
	}

	spec.Hooks = &runtimespec.Hooks{
		CreateRuntime: []runtimespec.Hook{runtimespec.Hook{
			Path: "/usr/bin/touch",
			Args: []string{"/usr/bin/touch", filepath.Join(dir, "create-runtime")},
		}},
		Poststop: []runtimespec.Hook{runtimespec.Hook{
			Path: "/usr/bin/touch",
			Args: []string{"/usr/bin/touch", filepath.Join(dir, "poststop")},
		}},
	}

	_, _, err := runExitingJail(t, "integ-test-hooks", spec, 500*time.Millisecond)
	assert.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "create-runtime"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, "poststop"))
	assert.NoError(t, err)
}

func TestHookTimeout(t *testing.T) {
	start := time.Now()
	spec := setupSimpleExitingJail(t)
	timeout := 4

	spec.Process = &runtimespec.Process{
		Args: []string{"/integ-inside", "-test.v", "-test.run", "TestHello"},
	}

	spec.Hooks = &runtimespec.Hooks{
		CreateRuntime: []runtimespec.Hook{runtimespec.Hook{
			Path:    "/bin/sleep",
			Args:    []string{"/bin/sleep", "5000"},
			Timeout: &timeout,
		}},
	}

	_, _, err := runExitingJail(t, "integ-test-hooks", spec, 500*time.Millisecond)
	assert.Error(t, err)
	assert.Less(t, time.Duration(timeout)*time.Second, time.Since(start)*time.Second)
}
