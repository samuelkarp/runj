//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.sbk.wtf/runj/runtimespec"
)

func TestHostIPv4Network(t *testing.T) {
	spec := setupSimpleExitingJail(t)
	mux := http.NewServeMux()
	var called int64
	const response = "hi there!"
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&called, 1)
		fmt.Fprint(w, response)
	})
	server := &http.Server{
		Addr:    ":0",
		Handler: mux,
	}
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "failed to bind to port")
	t.Cleanup(func() { listener.Close() })
	port := listener.Addr().(*net.TCPAddr).Port
	t.Log("test server listening port:", port)

	go func() {
		err = server.Serve(listener)
		if err == http.ErrServerClosed {
			return
		}
		require.NoError(t, err, "failed to set up test server")
	}()
	t.Cleanup(func() { server.Shutdown(context.Background()) })

	spec.FreeBSD = &runtimespec.FreeBSD{
		Network: &runtimespec.FreeBSDNetwork{
			IPv4: &runtimespec.FreeBSDIPv4{
				Mode: "inherit"},
		},
	}
	spec.Process = &runtimespec.Process{
		Args: []string{"/integ-inside", "-test.run", "TestLocalhostHTTPHello"},
		Env:  []string{fmt.Sprintf("TEST_PORT=%d", port)},
	}

	stdout, stderr, err := runExitingJail(t, "integ-test-localhost-http", spec, 500*time.Millisecond)
	assert.NoError(t, err)
	t.Logf("received %d request(s)\n", called)
	assert.GreaterOrEqual(t, called, int64(1), "should receive at least one request")
	assertJailPass(t, stdout, stderr)
	lines := strings.Split(string(stdout), "\n")
	assert.Len(t, lines, 3, "should be exactly 3 lines of output")
	assert.Equal(t, response, lines[0], "response should match")
	if t.Failed() {
		t.Log("STDOUT:", string(stdout))
	}
}

func TestVNetBridge(t *testing.T) {
	// TODO: IPAM
	bridgeAddr := "172.31.255.1"
	jailAddr := "172.31.255.2"
	mask := "24"

	tests := []struct {
		name    string
		setupPF bool
		pingIP  string
	}{
		{"no-nat", false, bridgeAddr},
		{"nat", true, "8.8.8.8"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupPF {
				setupPFNAT(t, jailAddr)
			}
			_, epairB := setupEpairBridge(t, bridgeAddr, mask)
			spec := setupFullExitingJail(t)
			spec.Process = &runtimespec.Process{
				Args: []string{"/integ-inside", "-test.run", "TestVnetConfigAndPing", "-test.v"},
				Env: []string{
					"TEST_INTERFACE=" + epairB,
					"TEST_IP=" + jailAddr,
					"TEST_MASK=" + mask,
					"TEST_GATEWAY=" + bridgeAddr,
					"TEST_PING_IP=" + tc.pingIP,
				},
			}
			spec.FreeBSD = &runtimespec.FreeBSD{
				Network: &runtimespec.FreeBSDNetwork{VNet: &runtimespec.FreeBSDVNet{
					Mode:       "new",
					Interfaces: []string{epairB},
				}},
			}

			stdout, stderr, err := runExitingJail(t, "integ-test-vnet-bridge", spec, 30*time.Second)
			assert.NoError(t, err)
			assertJailPass(t, stdout, stderr)
			t.Log("STDOUT:", string(stdout))

		})
	}
}

func setupPFNAT(t *testing.T, jailAddr string) {
	// validate test can run (maybe we can save rules instead of this?)
	out, err := exec.Command("pfctl", "-s", "nat").CombinedOutput()
	require.NoError(t, err, "failed to check for nat rules:\n%s", string(out))
	require.Equal(t, "", string(out), "nat rules must be empty for this test to run")

	// enable IP forwarding
	err = exec.Command("sysctl", "net.inet.ip.forwarding=1").Run()
	require.NoError(t, err, "failed to enable IP forwarding")

	// construct nat rule
	out, err = exec.Command("route", "-4", "get", "default").Output()
	require.NoError(t, err, "failed to get default route")
	var defaultInterface string
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "interface:") {
			defaultInterface = strings.TrimPrefix(strings.TrimSpace(line), "interface: ")
		}
	}
	require.NotEmpty(t, defaultInterface, "failed to find default interface")
	dir := t.TempDir()
	const natTable = "test-vnet-bridge-nat"
	nat := fmt.Sprintf("nat on %s inet from <%s> to any -> (%s)\n", defaultInterface, natTable, defaultInterface)
	t.Log(nat)
	err = os.WriteFile(filepath.Join(dir, "nat.conf"), []byte(nat), 0o644)
	require.NoError(t, err, "failed to write nat.conf")
	out, err = exec.Command("pfctl", "-f", filepath.Join(dir, "nat.conf")).CombinedOutput()
	require.NoError(t, err, "failed to load nat rules into pf: %q", string(out))
	t.Cleanup(func() {
		out, err := exec.Command("pfctl", "-F", "nat").CombinedOutput()
		require.NoError(t, err, "failed to flush nat rules: %v", string(out))
	})

	err = exec.Command("pfctl", "-t", natTable, "-T", "add", jailAddr).Run()
	require.NoError(t, err, "failed to add jail address to pf table")
	t.Cleanup(func() {
		err = exec.Command("pfctl", "-t", natTable, "-T", "delete", jailAddr).Run()
		require.NoError(t, err, "failed to remove jail address from pf table")
	})
}

func setupEpairBridge(t *testing.T, bridgeAddr string, mask string) (string, string) {
	out, err := exec.Command("ifconfig", "epair", "create").Output()
	require.NoError(t, err, "failed to create epair: %q", string(out))
	epairA := strings.TrimSpace(string(out))
	epairB := epairA[:len(epairA)-1] + "b"
	t.Cleanup(func() {
		err = exec.Command("ifconfig", epairA, "destroy").Run()
		require.NoError(t, err, "failed to destroy %q", epairA)
	})
	out, err = exec.Command("ifconfig", "bridge", "create").Output()
	require.NoError(t, err, "failed to create bridge: %q", string(out))
	bridge := strings.TrimSpace(string(out))
	t.Cleanup(func() {
		err = exec.Command("ifconfig", bridge, "destroy").Run()
		require.NoError(t, err, "failed to destroy %q", epairA)
	})
	err = exec.Command("ifconfig", bridge, "inet", bridgeAddr+"/"+mask).Run()
	require.NoError(t, err, "failed to set bridge address %s/%s", bridgeAddr, mask)
	err = exec.Command("ifconfig", bridge, "addm", epairA).Run()
	require.NoError(t, err, "failed to add %q to bridge %q", epairA, bridge)
	err = exec.Command("ifconfig", epairA, "up").Run()
	require.NoError(t, err, "failed to bring %q up", epairA)
	return epairA, epairB
}
