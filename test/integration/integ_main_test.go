//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/cheggaaa/pb/v3"
	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.sbk.wtf/runj/demo"
	"go.sbk.wtf/runj/internal/util"
)

const (
	cacheFormat = "%s.%s.base.txz"
)

var (
	baseCache  string
	fullRootfs string
)

func TestMain(m *testing.M) {
	// Exercise the freshly built binaries rather than whatever runj is
	// installed on PATH.  runj locates runj-entrypoint through PATH too, so
	// prepending the build directory covers both.
	binDir, err := filepath.Abs("bin")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve bin dir: %v\n", err)
		os.Exit(1)
	}
	os.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	baseCache = filepath.Join(os.TempDir(), "runj-integ-test-cache", "base")
	fullRootfs = filepath.Join(os.TempDir(), "runj-integ-test-cache", "rootfs")
	// Sweep before prepareRootfs: a still-running leftover jail can hold the
	// rootfs busy and block its removal, so orphaned jails are removed first.
	sweepIntegLeftovers()
	if err := prepareRootfs(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to prepare rootfs: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	if err := cleanup(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to remove rootfs: %v\n", err)
	}
	os.Exit(code)
}

func cleanup() error {
	fmt.Println("Cleaning rootfs...")
	return removeRootfs()
}

const (
	// integStateDir is where runj stores per-container state.  It mirrors the
	// default in state/dir.go; the sweep reads it to find leaked state.
	integStateDir = "/var/lib/runj/jails"

	// integTestIfGroup is the ifconfig(8) group the vnet tests put the epair
	// and bridge interfaces they create into, so the sweep can tell them apart
	// from unrelated interfaces.
	integTestIfGroup = "integ-test"

	// integTestNATTable is the pf table the vnet NAT test populates.
	integTestNATTable = "test-vnet-bridge-nat"

	// integTestJailPrefix is the container-ID prefix used for all integration
	// tests.  New integration tests should use this same prefix.
	integTestJailPrefix = "integ-test-"
)

// isIntegTestJail reports whether name follows an integration-test container-ID
// convention.
func isIntegTestJail(name string) bool {
	return strings.HasPrefix(name, integTestJailPrefix)
}

// sweepIntegLeftovers removes the kernel jails, runj state directories, network
// interfaces, and pf nat state left by an interrupted integ-test.  A killed
// test never reaches a test's cleanup, orphaning the root-owned kernel jail and
// its state directory under /var/lib/runj/jails, plus the epair/bridge
// interfaces and pf nat state the vnet tests set up.  Those wedge the next run:
// `jail "…" already exists`, `state.json: file exists`, or setupPFNAT's
// precondition that no nat rule already exists.
//
// Each step is best-effort and scoped to an integration-test marker (a jail-ID
// prefix, the integ-test ifconfig group, or the vnet nat table), so the sweep
// never touches unrelated state.  It logs what it removes and never fails the
// run; off root or off FreeBSD the underlying commands simply error and are
// logged.
func sweepIntegLeftovers() {
	// Jails first: a still-running jail holds its state directory (and possibly
	// the rootfs) busy and owns a vnet interface, so it must go before the
	// state, interface, and rootfs steps.
	sweepIntegJails()
	sweepIntegStateDirs()
	// Interfaces after jails, so no live jail still holds a vnet interface.
	sweepIntegInterfaces()
	sweepIntegPFNAT()
}

// sweepIntegJails removes live integration-test kernel jails.
func sweepIntegJails() {
	out, err := exec.Command("jls", "name").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sweep: jls failed, skipping jail sweep: %v: %s\n", err, out)
		return
	}
	for _, name := range strings.Fields(string(out)) {
		if !isIntegTestJail(name) {
			continue
		}
		fmt.Printf("sweep: removing leftover jail %q\n", name)
		if o, err := exec.Command("jail", "-r", name).CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "sweep: jail -r %q failed: %v: %s\n", name, err, o)
		}
	}
}

// sweepIntegStateDirs removes leftover runj state directories.  A killed run
// leaves the state dir even when the kernel jail is already gone.
func sweepIntegStateDirs() {
	entries, err := os.ReadDir(integStateDir)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "sweep: reading %s failed, skipping state sweep: %v\n", integStateDir, err)
		}
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if !isIntegTestJail(name) {
			continue
		}
		dir := filepath.Join(integStateDir, name)
		fmt.Printf("sweep: removing leftover state dir %q\n", dir)
		if o, err := exec.Command("chflags", "-R", "noschg", dir).CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "sweep: chflags %q failed: %v: %s\n", dir, err, o)
		}
		if err := os.RemoveAll(dir); err != nil {
			fmt.Fprintf(os.Stderr, "sweep: removing %q failed: %v\n", dir, err)
		}
	}
}

// sweepIntegInterfaces destroys the epair and bridge interfaces the vnet tests
// create.  setupEpairBridge tags them into the integTestIfGroup ifconfig group,
// so `ifconfig -g` enumerates exactly the test's interfaces and nothing else;
// the caller removes jails first, so no live jail still holds one.
func sweepIntegInterfaces() {
	out, err := exec.Command("ifconfig", "-g", integTestIfGroup).Output()
	if err != nil {
		// No interface belongs to the group, or ifconfig is unavailable;
		// nothing to sweep.
		return
	}
	for _, iface := range strings.Fields(string(out)) {
		fmt.Printf("sweep: destroying leftover interface %q\n", iface)
		if o, err := exec.Command("ifconfig", iface, "destroy").CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "sweep: ifconfig %q destroy failed: %v: %s\n", iface, err, o)
		}
	}
}

// sweepIntegPFNAT removes the pf nat state the vnet NAT test leaves behind.
// setupPFNAT loads a nat ruleset and populates integTestNATTable, and refuses
// to run while any nat rule already exists, so an interrupted NAT test would
// fail the next run.
func sweepIntegPFNAT() {
	if err := exec.Command("pfctl", "-t", integTestNATTable, "-T", "show").Run(); err != nil {
		// Table absent, or pf disabled/unavailable: nothing of ours to clean.
		return
	}
	fmt.Printf("sweep: flushing leftover pf nat and table %q\n", integTestNATTable)
	if o, err := exec.Command("pfctl", "-F", "nat").CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "sweep: pfctl -F nat failed: %v: %s\n", err, o)
	}
	if o, err := exec.Command("pfctl", "-t", integTestNATTable, "-T", "kill").CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "sweep: pfctl -T kill %q failed: %v: %s\n", integTestNATTable, err, o)
	}
}

// mountpointsUnder parses `mount -p` output and returns the mountpoints at or
// below dir, deepest first so a nested mount is unmounted before its parent.
func mountpointsUnder(mountOutput, dir string) []string {
	prefix := dir + string(os.PathSeparator)
	var points []string
	for _, line := range strings.Split(mountOutput, "\n") {
		// mount -p prints fstab-style lines: "device mountpoint fstype ...".
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if mp := fields[1]; mp == dir || strings.HasPrefix(mp, prefix) {
			points = append(points, mp)
		}
	}
	sort.Slice(points, func(i, j int) bool { return len(points[i]) > len(points[j]) })
	return points
}

// unmountUnder unmounts every filesystem mounted at or below dir, deepest
// first.  An interrupted integ-test leaves runj's per-jail mounts in place
// (setupFullExitingJail mounts devfs into the rootfs), because runj only
// unmounts them during `runj delete`.  A leftover mount blocks reclaiming the
// rootfs: chflags -R descends into it and RemoveAll cannot remove the busy
// mountpoint.
func unmountUnder(dir string) {
	out, err := exec.Command("mount", "-p").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "reclaim: mount -p failed, skipping unmount: %v: %s\n", err, out)
		return
	}
	for _, mountpoint := range mountpointsUnder(string(out), dir) {
		fmt.Printf("reclaim: unmounting leftover mount %q\n", mountpoint)
		if o, err := exec.Command("umount", mountpoint).CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "reclaim: umount %q failed, retrying with -f: %v: %s\n", mountpoint, err, o)
			if o, err := exec.Command("umount", "-f", mountpoint).CombinedOutput(); err != nil {
				fmt.Fprintf(os.Stderr, "reclaim: umount -f %q failed: %v: %s\n", mountpoint, err, o)
			}
		}
	}
}

// removeRootfs removes the extracted rootfs.  It is a no-op when the directory
// is absent and best-effort otherwise: it unmounts anything left mounted under
// the rootfs, then clears the schg flag recursively (the extracted base system
// carries immutable files) before removing the tree.
func removeRootfs() error {
	if _, err := os.Stat(fullRootfs); os.IsNotExist(err) {
		return nil
	}
	unmountUnder(fullRootfs)
	if out, err := exec.Command("chflags", "-R", "noschg", fullRootfs).CombinedOutput(); err != nil {
		fmt.Fprint(os.Stderr, string(out))
		return err
	}
	return os.RemoveAll(fullRootfs)
}

func prepareRootfs() error {
	// An interrupted integ-test never reaches cleanup(), orphaning the
	// extracted rootfs.  Reclaim it here rather than requiring a manual root
	// cleanup.
	if err := removeRootfs(); err != nil {
		return err
	}
	if err := os.MkdirAll(baseCache, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(fullRootfs, 0o755); err != nil {
		return err
	}

	machine, err := demo.FreeBSDMachine(context.Background())
	if err != nil {
		return err
	}
	fmt.Println("Found machine: ", machine)

	arch, err := demo.FreeBSDArch(context.Background())
	if err != nil {
		return err
	}
	fmt.Println("Found arch: ", arch)

	version, err := demo.FreeBSDVersion(context.Background())
	if err != nil {
		return err
	}
	fmt.Println("Found version: ", version)

	// check if the file is already downloaded
	cacheFile := filepath.Join(baseCache, fmt.Sprintf(cacheFormat, version, arch))
	if _, err := os.Stat(cacheFile); err != nil {
		f, err := os.OpenFile(cacheFile, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		err = downloadImage(machine, arch, version, f)
		f.Close()
		if err != nil {
			return err
		}
	}
	// extract
	fmt.Println("Extracting rootfs...")
	out, err := exec.Command("tar", "--directory", fullRootfs, "-xJf", cacheFile).CombinedOutput()
	fmt.Println(string(out))
	return err
}

// TestRemoveRootfsReclaimsStale proves the reclaim path prepareRootfs relies
// on: removeRootfs deletes a leftover rootfs from an interrupted run, and a
// second call on an absent directory is a no-op.  It swaps the fullRootfs
// global for a temporary directory so it does not touch the real rootfs
// TestMain prepared.
func TestRemoveRootfsReclaimsStale(t *testing.T) {
	realRootfs := fullRootfs
	t.Cleanup(func() { fullRootfs = realRootfs })
	fullRootfs = filepath.Join(t.TempDir(), "rootfs")

	require.NoError(t, os.MkdirAll(fullRootfs, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(fullRootfs, "file"), []byte("stale"), 0o644))

	require.NoError(t, removeRootfs(), "reclaim stale rootfs")
	_, err := os.Stat(fullRootfs)
	assert.True(t, os.IsNotExist(err), "rootfs should be gone after reclaim")

	assert.NoError(t, removeRootfs(), "reclaim of absent rootfs should be a no-op")
}

func TestIsIntegTestJail(t *testing.T) {
	for _, name := range []string{
		"integ-test-hello",
		"integ-test-hooks",
		"integ-test-ociversion",
		"integ-test-localhost-http",
		"integ-test-vnet-bridge",
		"integ-test-create-delete-0",
		"integ-test-create-delete-3",
	} {
		assert.True(t, isIntegTestJail(name), "%q should match", name)
	}
	for _, name := range []string{
		"",
		"foo",
		"smoke-hello",
		"smoke",
		"my-integ-test-hello",  // contains but does not begin with the prefix
		"test-create-delete-1", // old pattern
		"integtest-hello",      // missing the hyphen
	} {
		assert.False(t, isIntegTestJail(name), "%q should not match", name)
	}
}

// TestMountpointsUnder proves removeRootfs selects exactly the mounts at or
// below the rootfs (so the devfs setupFullExitingJail leaves behind is
// unmounted), returns them deepest first, and ignores mounts elsewhere and a
// sibling whose path merely shares the rootfs as a string prefix.
func TestMountpointsUnder(t *testing.T) {
	const dir = "/tmp/runj-integ-test-cache/rootfs"
	mountOutput := strings.Join([]string{
		"/dev/gpt/rootfs / ufs rw 1 1",
		"devfs " + dir + "/dev devfs rw 0 0",
		"tmpfs " + dir + " tmpfs rw 0 0",
		"nullfs " + dir + "/volume/nested nullfs rw 0 0",
		"devfs /tmp/runj-integ-test-cache/rootfs-other/dev devfs rw 0 0", // prefix string, not under dir
		"", // trailing blank line
	}, "\n")

	assert.Equal(t, []string{
		dir + "/volume/nested",
		dir + "/dev",
		dir,
	}, mountpointsUnder(mountOutput, dir))
}

func downloadImage(machine, arch, version string, f *os.File) error {
	fmt.Printf("Downloading image for %s/%s %s into %s\n", machine, arch, version, f.Name())
	rootfs, rootLen, err := demo.DownloadRootfs(machine, arch, version)
	if err != nil {
		return err
	}
	defer rootfs.Close()
	bar := pb.Full.Start64(rootLen)
	barReader := bar.NewProxyReader(rootfs)
	_, err = io.Copy(f, barReader)
	bar.Finish()
	return err
}

func setupSimpleExitingJail(t *testing.T) runtimespec.Spec {
	root := t.TempDir()

	s, err := os.Stat("bin/integ-inside")
	require.NoError(t, err, "stat bin/integ-inside")
	err = util.CopyFile("bin/integ-inside", filepath.Join(root, "integ-inside"), s.Mode())
	require.NoError(t, err, "copy inside binary")

	t.Cleanup(func() {
		err := os.RemoveAll(root)
		assert.NoError(t, err, "failed to remove tempdir")
	})
	return runtimespec.Spec{
		Root: &runtimespec.Root{Path: root},
	}
}

func setupFullExitingJail(t *testing.T) runtimespec.Spec {
	s, err := os.Stat("bin/integ-inside")
	require.NoError(t, err, "stat bin/integ-inside")
	integInside := filepath.Join(fullRootfs, "integ-inside")
	if _, err := os.Stat(integInside); err == nil {
		err = os.Remove(integInside)
		assert.NoError(t, err, "remove old inside binary")
	}
	err = util.CopyFile("bin/integ-inside", integInside, s.Mode())
	require.NoError(t, err, "copy inside binary")

	return runtimespec.Spec{
		Root: &runtimespec.Root{Path: fullRootfs},
		Mounts: []runtimespec.Mount{{
			Destination: "/dev",
			Source:      "devfs",
			Type:        "devfs",
			Options:     []string{"ruleset=4"},
		}},
	}
}

func assertJailPass(t *testing.T, stdout, stderr []byte) {
	t.Helper()
	assert.Equal(t, []byte{}, stderr, "stderr should be empty")
	lines := strings.Split(string(stdout), "\n")
	require.GreaterOrEqual(t, len(lines), 2, "stdout should have at least two lines")
	assert.Equal(t, "PASS", lines[len(lines)-2], "second to last line of output should be PASS")
}

// createJail writes spec to a fresh bundle and runs `runj create` for id,
// returning the command's combined output and error.  It suits tests that assert
// on the outcome of create itself, such as validation rejections.  The bundle
// and the jail are removed when the test ends, whether or not create succeeded.
func createJail(t *testing.T, id string, spec runtimespec.Spec) ([]byte, error) {
	t.Helper()
	dir, err := os.MkdirTemp("", "runj-integ-test-"+strings.ReplaceAll(t.Name(), "/", "-"))
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "root"), 0755), "create root dir")

	configJSON, err := json.Marshal(spec)
	require.NoError(t, err, "marshal config")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), configJSON, 0644), "write config")

	exec.Command("runj", "delete", id).Run() // best-effort: clear any leftover
	t.Cleanup(func() { exec.Command("runj", "delete", id).Run() })

	return exec.Command("runj", "create", id, dir).CombinedOutput()
}

// runExitingJail is a helper that takes a spec as input, sets up a bundle
// starts a jail, collects its output, and waits for the jail's entrypoint to
// exit.  It can be used in tests where the entrypoint embeds the test
// assertions.
// TODO: Build a better non-racy or less-racy end condition.
// The wait parameter is currently used as a simple sleep between `runj start`
// and `runj delete`.  A normal wait is not used as the jail's main process is
// not a direct child of this test; it's instead a child of the `runj create`
// process.
func runExitingJail(t *testing.T, id string, spec runtimespec.Spec, wait time.Duration) ([]byte, []byte, error) {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "-")
	bundleDir, err := os.MkdirTemp("", "runj-integ-test-"+name+"-"+id)
	if err != nil {
		return nil, nil, err
	}
	t.Cleanup(func() {
		if err == nil && !t.Failed() {
			os.RemoveAll(bundleDir)
		} else {
			t.Log("preserving tempdir due to error or failed", bundleDir, err, t.Failed())
			stderr, err := os.ReadFile(filepath.Join(bundleDir, "stderr"))
			if err == nil {
				t.Logf("stderr: %s", string(stderr))
			}
		}
	})
	rootDir := filepath.Join(bundleDir, "root")
	err = os.MkdirAll(rootDir, 0755)
	if err != nil {
		return nil, nil, fmt.Errorf("create bundle dir: %w", err)
	}
	t.Log("bundle", bundleDir)

	configJSON, err := json.Marshal(spec)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal config: %w", err)
	}
	err = os.WriteFile(filepath.Join(bundleDir, "config.json"), configJSON, 0644)
	if err != nil {
		return nil, nil, fmt.Errorf("write config: %w", err)
	}

	cmd := exec.Command("runj", "create", id, bundleDir)
	cmd.Stdin = nil
	stdout, err := os.OpenFile(filepath.Join(bundleDir, "stdout"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, fmt.Errorf("create stdout file: %w", err)
	}
	cmd.Stdout = stdout
	stderr, err := os.OpenFile(filepath.Join(bundleDir, "stderr"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, fmt.Errorf("create stderr file: %w", err)
	}
	cmd.Stderr = stderr

	err = cmd.Run()
	if err != nil {
		return nil, nil, fmt.Errorf("runj create: %w", err)
	}
	err = stdout.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("close stdout file: %w", err)
	}
	err = stderr.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("close stderr file: %w", err)
	}

	defer func() {
		// remove jail
		cmd = exec.Command("runj", "delete", id)
		cmd.Stdin = nil
		outBytes, cleanupErr := cmd.CombinedOutput()
		if cleanupErr != nil && err == nil {
			err = fmt.Errorf("runj delete: %w", cleanupErr)
		}
		if len(outBytes) > 0 {
			t.Log("runj delete output:", string(outBytes))
		}
	}()

	// runj start
	cmd = exec.Command("runj", "start", id)
	err = cmd.Run()
	if err != nil {
		return nil, nil, fmt.Errorf("runj start: %w", err)
	}
	time.Sleep(wait)

	stdoutBytes, err := os.ReadFile(filepath.Join(bundleDir, "stdout"))
	if err != nil {
		return nil, nil, fmt.Errorf("read stdout file: %w", err)
	}
	stderrBytes, err := os.ReadFile(filepath.Join(bundleDir, "stderr"))
	if err != nil {
		return nil, nil, fmt.Errorf("read stderr file: %w", err)
	}
	return stdoutBytes, stderrBytes, nil
}
