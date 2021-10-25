// +build inside

package integration

import (
	"fmt"
	"io/fs"
	"io/ioutil"
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
