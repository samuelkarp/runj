package main

import (
	"context"

	"github.com/containerd/containerd/v2/pkg/shim"
	"go.sbk.wtf/runj/containerd"
)

func main() {
	shim.Run(context.Background(), containerd.NewManager("wtf.sbk.runj.v1"))
}
