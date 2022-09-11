//go:build inside
// +build inside

package integration

import (
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
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
	input, err := ioutil.ReadFile("/volume/hello.txt")
	assert.NoError(t, err, "cannot read hello.txt")
	assert.Equal(t, "input file", string(input), "unexpected file contents")
	err = ioutil.WriteFile("/volume/world.txt", []byte("output file"), 0644)
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
