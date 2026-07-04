package state

import (
	"os"
	"path/filepath"
	"testing"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// redirectStateDir points the package's state directory at a temporary location
// for the duration of a test.
func redirectStateDir(t *testing.T) {
	t.Helper()
	orig := stateDir
	stateDir = t.TempDir()
	t.Cleanup(func() { stateDir = orig })
}

func TestOutput(t *testing.T) {
	s := &State{
		ID:     "container1",
		JID:    7,
		Status: StatusRunning,
		Bundle: "/bundle",
		PID:    4422,
	}
	out := s.Output()
	assert.Equal(t, runtimespec.Version, out.OCIVersion)
	assert.Equal(t, "container1", out.ID)
	assert.Equal(t, string(StatusRunning), out.Status)
	assert.Equal(t, "/bundle", out.Bundle)
	assert.Equal(t, 4422, out.PID)
	// JID is internal state and is not part of the OCI output.
	assert.Nil(t, out.Annotations)
}

func TestOutputOCIVersion(t *testing.T) {
	// The bundle's declared version is honored.
	withVersion := (&State{OCIVersion: "1.1.0-test"}).Output()
	assert.Equal(t, "1.1.0-test", withVersion.OCIVersion)
	// An unset version falls back to runj's supported version.
	withoutVersion := (&State{}).Output()
	assert.Equal(t, runtimespec.Version, withoutVersion.OCIVersion)
}

func TestCreateAndLoad(t *testing.T) {
	redirectStateDir(t)

	created, err := Create("container1", "/bundle")
	require.NoError(t, err)
	assert.Equal(t, "container1", created.ID)
	assert.Equal(t, "/bundle", created.Bundle)
	assert.Equal(t, StatusCreating, created.Status)

	loaded, err := Load("container1")
	require.NoError(t, err)
	assert.Equal(t, created, loaded)
}

func TestCreateRejectsDuplicate(t *testing.T) {
	redirectStateDir(t)

	_, err := Create("dupe", "/bundle")
	require.NoError(t, err)

	// initialize uses O_CREATE|O_EXCL, so a second Create for the same ID must
	// fail rather than clobber the existing state.
	_, err = Create("dupe", "/other")
	assert.ErrorIs(t, err, os.ErrExist)
}

func TestSaveRoundTrip(t *testing.T) {
	redirectStateDir(t)

	s, err := Create("container1", "/bundle")
	require.NoError(t, err)

	s.JID = 12
	s.PID = 999
	s.Status = StatusRunning
	require.NoError(t, s.Save())

	loaded, err := Load("container1")
	require.NoError(t, err)
	assert.Equal(t, 12, loaded.JID)
	assert.Equal(t, 999, loaded.PID)
	assert.Equal(t, StatusRunning, loaded.Status)
}

func TestLoadMissing(t *testing.T) {
	redirectStateDir(t)

	_, err := Load("does-not-exist")
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestRemove(t *testing.T) {
	redirectStateDir(t)

	_, err := Create("container1", "/bundle")
	require.NoError(t, err)
	require.DirExists(t, Dir("container1"))

	require.NoError(t, Remove("container1"))
	_, err = os.Stat(Dir("container1"))
	assert.ErrorIs(t, err, os.ErrNotExist)

	// Remove is idempotent: removing an already-absent container is not an error.
	assert.NoError(t, Remove("container1"))
}

func TestDir(t *testing.T) {
	redirectStateDir(t)
	assert.Equal(t, filepath.Join(stateDir, "abc"), Dir("abc"))
}
