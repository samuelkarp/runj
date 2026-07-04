//go:build integration
// +build integration

package integration

import (
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

// jailStatfsCount runs the in-jail getfsstat probe and returns the number of
// filesystems visible to the jail.
func jailStatfsCount(t *testing.T, id string, enforceStatfs int) int {
	t.Helper()
	spec := setupSimpleExitingJail(t)
	spec.Process = &runtimespec.Process{
		Args: []string{"/integ-inside", "-test.run", "TestEnforceStatfsCount"},
	}
	spec.FreeBSD = &runtimespec.FreeBSD{
		Jail: &runtimespec.FreeBSDJail{EnforceStatfs: &enforceStatfs},
	}

	stdout, stderr, err := runExitingJail(t, id, spec, 500*time.Millisecond)
	assert.NoError(t, err)
	assertJailPass(t, stdout, stderr)
	lines := strings.Split(string(stdout), "\n")
	require.GreaterOrEqual(t, len(lines), 2, "stdout should have a count and PASS")
	n, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	require.NoError(t, err, "parse getfsstat count from %q", string(stdout))
	return n
}

// mountExtraHostFS mounts a nullfs outside any jail root so the host mount table
// holds more than one filesystem, and returns the host's getfsstat count.  The
// mount is removed when the test ends.
func mountExtraHostFS(t *testing.T) int {
	t.Helper()
	src := t.TempDir()
	mnt := t.TempDir()
	out, err := exec.Command("mount", "-t", "nullfs", src, mnt).CombinedOutput()
	require.NoError(t, err, "mount nullfs: %s", out)
	t.Cleanup(func() {
		out, err := exec.Command("umount", mnt).CombinedOutput()
		assert.NoError(t, err, "umount nullfs: %s", out)
	})
	host, err := unix.Getfsstat(nil, unix.MNT_NOWAIT)
	require.NoError(t, err, "host getfsstat")
	require.Greater(t, host, 1, "nullfs mount should give the host multiple filesystems")
	return host
}

// TestJailEnforceStatfsUnrestricted confirms enforce_statfs=0 exposes the host's
// entire mount table to the jail.  A simple jail adds no mounts of its own and
// shares the host's global mount list, so the jail's getfsstat count must equal
// the host's.  The nullfs mounted outside the jail root is one that =1 would hide
// and =2 would exclude, so equality with the full host count distinguishes =0
// from =1 and =2.
func TestJailEnforceStatfsUnrestricted(t *testing.T) {
	host := mountExtraHostFS(t)
	n := jailStatfsCount(t, "integ-test-statfs-0", 0)
	assert.Equal(t, host, n, "enforce_statfs=0 should expose the host's entire mount table")
}

// TestJailEnforceStatfsRestricted confirms enforce_statfs=2 restricts the jail to
// seeing only the filesystem its root resides on, even when the host mount table
// holds more.
func TestJailEnforceStatfsRestricted(t *testing.T) {
	mountExtraHostFS(t)
	n := jailStatfsCount(t, "integ-test-statfs-2", 2)
	assert.Equal(t, 1, n, "enforce_statfs=2 should restrict the jail to its own root")
}
