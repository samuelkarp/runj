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
	baseCache = filepath.Join(os.TempDir(), "runj-integ-test-cache", "base")
	fullRootfs = filepath.Join(os.TempDir(), "runj-integ-test-cache", "rootfs")
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
	if out, err := exec.Command("chflags", "-R", "noschg", fullRootfs).CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, string(out))
		return err
	}
	return os.RemoveAll(fullRootfs)
}

func prepareRootfs() error {
	if _, err := os.Stat(fullRootfs); err == nil {
		return fmt.Errorf("prepare: %q must not exist", fullRootfs)
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
