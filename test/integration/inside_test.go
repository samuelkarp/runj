// +build inside

package integration

import (
	"fmt"
	"testing"
	"time"
)

func TestHello(t *testing.T) {
	fmt.Println("Hello println!")
	time.Sleep(time.Second)
	t.Log("Hello t.Log!")
}
