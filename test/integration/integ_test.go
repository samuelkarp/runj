//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDelete(t *testing.T) {
	dir, err := os.MkdirTemp("", "runj-integ-test-"+t.Name())
	require.NoError(t, err)
	defer func() {
		if !t.Failed() {
			os.RemoveAll(dir)
		} else {
			t.Log("preserving tempdir due to failure", dir)
		}
	}()

	tests := []runtimespec.Spec{
		// minimal
		{
			Process: &runtimespec.Process{},
		},
		// arguments
		{
			Process: &runtimespec.Process{
				Args: []string{"one", "two", "three"},
			},
		},
		// environment variables
		{
			Process: &runtimespec.Process{
				Env: []string{"one=two", "three=four", "five"},
			},
		},
		// hostname
		{
			Hostname: "foo.bar.example.com",
			Process:  &runtimespec.Process{},
		},
		// ipv4
		{
			Process: &runtimespec.Process{},
			FreeBSD: &runtimespec.FreeBSD{
				Jail: &runtimespec.FreeBSDJail{
					Ip4:     runtimespec.FreeBSDShareNew,
					Ip4Addr: []string{"127.0.0.2"},
				},
			},
		},
		// vnet
		{
			Process: &runtimespec.Process{},
			FreeBSD: &runtimespec.FreeBSD{
				Jail: &runtimespec.FreeBSDJail{
					Vnet: runtimespec.FreeBSDShareNew,
				},
			},
		},
	}

	for i, tc := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			bundleDir := filepath.Join(dir, strconv.Itoa(i))
			defer func() {
				if !t.Failed() {
					os.RemoveAll(bundleDir)
				} else {
					t.Log("preserving tempdir due to failure", bundleDir)
				}
			}()
			rootDir := filepath.Join(bundleDir, "root")
			err := os.MkdirAll(rootDir, 0755)
			require.NoError(t, err, "create bundle dir")
			t.Log("bundle", bundleDir)

			configJSON, err := json.Marshal(tc)
			require.NoError(t, err, "marshal config")
			err = os.WriteFile(filepath.Join(bundleDir, "config.json"), configJSON, 0644)
			require.NoError(t, err, "write config")

			id := "test-create-delete-" + strconv.Itoa(i)
			var cmd *exec.Cmd
			switch i % 3 {
			case 0:
				cmd = exec.Command("runj", "create", id, bundleDir, "--pid-file", "jail.pid")
				t.Log("using argument form")
			case 1:
				cmd = exec.Command("runj", "create", id, "--bundle", bundleDir, "--pid-file", "jail.pid")
				t.Log("using --bundle form")
			case 2:
				cmd = exec.Command("runj", "create", id, "-b", bundleDir, "--pid-file", "jail.pid")
				t.Log("using -b form")
			default:
				t.Fatalf("Unhandled test variant; %d%%3 = %d", i, i%3)
			}
			cmd.Stdin = nil
			out, err := os.OpenFile(filepath.Join(bundleDir, "out"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
			require.NoError(t, err, "out file")
			cmd.Stdout = out
			cmd.Stderr = out
			err = cmd.Run()
			assert.NoError(t, err, "runj create")
			err = out.Close()
			assert.NoError(t, err, "out file close")
			outBytes, err := os.ReadFile(filepath.Join(bundleDir, "out"))
			assert.NoError(t, err, "out file read")
			t.Log("runj create output:", string(outBytes))

			pidfile, err := os.ReadFile("jail.pid")
			assert.NoError(t, err)
			t.Logf("pid: %q", string(pidfile))

			cmd = exec.Command("runj", "delete", id)
			cmd.Stdin = nil
			outBytes, err = cmd.CombinedOutput()
			assert.NoError(t, err, "runj delete")
			t.Log("runj delete output:", string(outBytes))
		})
	}
}

func TestJailHello(t *testing.T) {
	spec := setupSimpleExitingJail(t)

	spec.Process = &runtimespec.Process{
		Args: []string{"/integ-inside", "-test.v", "-test.run", "TestHello"},
	}

	stdout, stderr, err := runExitingJail(t, "integ-test-hello", spec, 500*time.Millisecond)
	assert.NoError(t, err)
	t.Log("STDOUT:", string(stdout))
	t.Log("STDERR:", string(stderr))
}

func TestJailEnv(t *testing.T) {
	env := []string{"Hello=World", "FOO=bar"}

	spec := setupSimpleExitingJail(t)

	spec.Process = &runtimespec.Process{
		Args: []string{"/integ-inside", "-test.run", "TestEnv"},
		Env:  env,
	}

	stdout, stderr, err := runExitingJail(t, "integ-test-env", spec, 500*time.Millisecond)
	assert.NoError(t, err)
	assertJailPass(t, stdout, stderr)
	lines := strings.Split(string(stdout), "\n")
	assert.ElementsMatch(t, env, lines[:len(lines)-2], "environment variables should match")
	if t.Failed() {
		t.Log("STDOUT:", string(stdout))
	}
}

func TestJailNullMount(t *testing.T) {
	spec := setupSimpleExitingJail(t)

	volume := t.TempDir()
	err := os.WriteFile(filepath.Join(volume, "hello.txt"), []byte("input file"), 0644)
	require.NoError(t, err, "input file")

	spec.Process = &runtimespec.Process{
		Args: []string{"/integ-inside", "-test.run", "TestNullMount"},
	}
	spec.Mounts = []runtimespec.Mount{{
		Destination: "/volume",
		Type:        "nullfs",
		Source:      volume,
	}}
	stdout, stderr, err := runExitingJail(t, "integ-test-null", spec, 500*time.Millisecond)
	assert.NoError(t, err)
	assertJailPass(t, stdout, stderr)
	output, err := os.ReadFile(filepath.Join(volume, "world.txt"))
	assert.NoError(t, err, "failed to read world.txt")
	assert.Equal(t, "output file", string(output))
	if t.Failed() {
		t.Log("STDOUT:", string(stdout))
	}
}

func TestJailHostname(t *testing.T) {
	hostname := fmt.Sprintf("%s.example", t.Name())

	spec := setupSimpleExitingJail(t)

	spec.Hostname = hostname
	spec.Process = &runtimespec.Process{
		Args: []string{"/integ-inside", "-test.run", "TestHostname"},
	}

	stdout, stderr, err := runExitingJail(t, "integ-test-hostname", spec, 500*time.Millisecond)
	assert.NoError(t, err)
	assertJailPass(t, stdout, stderr)
	lines := strings.Split(string(stdout), "\n")
	assert.Len(t, lines, 3, "should be exactly 3 lines of output")
	assert.Equal(t, hostname, lines[0], "hostname should match")
	if t.Failed() {
		t.Log("STDOUT:", string(stdout))
	}
}
