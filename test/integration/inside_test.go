// +build inside

package integration

import (
	"fmt"
	"os"
	"testing"
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
