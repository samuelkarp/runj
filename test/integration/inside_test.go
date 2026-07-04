//go:build inside
// +build inside

package integration

import (
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestHello(t *testing.T) {
	fmt.Println("Hello println!")
	t.Log("Hello t.Log!")
}

func TestEnv(t *testing.T) {
	for _, env := range os.Environ() {
		fmt.Println(env)
	}
}

func TestNullMount(t *testing.T) {
	stat, err := os.Stat("/volume/hello.txt")
	assert.NoError(t, err, "cannot stat hello.txt")
	assert.Equal(t, fs.FileMode(0), stat.Mode()&fs.ModeType, "unexpected file mode")
	input, err := os.ReadFile("/volume/hello.txt")
	assert.NoError(t, err, "cannot read hello.txt")
	assert.Equal(t, "input file", string(input), "unexpected file contents")
	err = os.WriteFile("/volume/world.txt", []byte("output file"), 0644)
	assert.NoError(t, err, "cannot write world.txt")
}

func TestHostname(t *testing.T) {
	hostname, err := os.Hostname()
	assert.NoError(t, err, "failed to retrieve hostname")
	fmt.Println(hostname)
}

// TestIP6Visible asserts that the IPv6 address the jail was configured with
// (TEST_IP6ADDR) is visible on one of the jail's interfaces.  If the jail's
// ip6.addr parameter never reached the kernel, the jail's network stack would
// not expose the address and this fails.
func TestIP6Visible(t *testing.T) {
	want := net.ParseIP(os.Getenv("TEST_IP6ADDR"))
	require.NotNil(t, want, "TEST_IP6ADDR must be a valid IP address")

	addrs, err := net.InterfaceAddrs()
	require.NoError(t, err, "failed to list interface addresses")

	seen := make([]string, 0, len(addrs))
	found := false
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		seen = append(seen, ipnet.IP.String())
		// To4 returns nil for a genuine IPv6 address.
		if ipnet.IP.To4() == nil && ipnet.IP.Equal(want) {
			found = true
		}
	}
	assert.True(t, found, "expected IPv6 address %s visible in jail; saw %v", want, seen)
}

// TestEnforceStatfsCount prints the number of filesystems visible to the jail
// via getfsstat(2).  enforce_statfs restricts this count.
func TestEnforceStatfsCount(t *testing.T) {
	n, err := unix.Getfsstat(nil, unix.MNT_NOWAIT)
	assert.NoError(t, err, "getfsstat")
	fmt.Println(n)
}

func TestDomainname(t *testing.T) {
	domainname, err := unix.Sysctl("kern.domainname")
	assert.NoError(t, err, "failed to retrieve domainname")
	fmt.Println(domainname)
}

func TestLocalhostHTTPHello(t *testing.T) {
	port := os.Getenv("TEST_PORT")
	requestURL := fmt.Sprintf("http://127.0.0.1:%s/hello", port)
	resp, err := http.Get(requestURL)
	assert.NoError(t, err, "failed to get from %q", requestURL)
	if err == nil {
		defer resp.Body.Close()
	}
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "failed to read body")
	fmt.Println(string(body))
}

func TestVnetConfigAndPing(t *testing.T) {
	var (
		iface   = os.Getenv("TEST_INTERFACE")
		ip      = os.Getenv("TEST_IP")
		mask    = os.Getenv("TEST_MASK")
		gateway = os.Getenv("TEST_GATEWAY")
		pingIP  = os.Getenv("TEST_PING_IP")
	)
	out, err := exec.Command("/sbin/ifconfig", iface, "inet", ip+"/"+mask).CombinedOutput()
	t.Logf("ifconfig %s inet %s/%s: %s", iface, ip, mask, string(out))
	assert.NoError(t, err)

	out, err = exec.Command("/sbin/route", "-4", "add", "default", gateway).CombinedOutput()
	t.Logf("route -4 add default %s: %s", ip, string(out))
	assert.NoError(t, err)

	out, err = exec.Command("/sbin/ping", "-c2", pingIP).CombinedOutput()
	t.Logf("ping -c2 %s: %s", pingIP, string(out))
	assert.NoError(t, err)
}
