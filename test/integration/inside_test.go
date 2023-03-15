//go:build inside
// +build inside

package integration

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestLocalhostHTTPHello(t *testing.T) {
	port := os.Getenv("TEST_PORT")
	requestURL := fmt.Sprintf("http://127.0.0.1:%s/hello", port)
	resp, err := http.Get(requestURL)
	assert.NoError(t, err, "failed to get from %q", requestURL)
	defer resp.Body.Close()
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
