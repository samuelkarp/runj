package main

import (
	"github.com/containerd/containerd/runtime/v2/shim"
	"go.sbk.wtf/runj/containerd"
)

func main() {
	shim.Run("runj", containerd.NewService)
}
