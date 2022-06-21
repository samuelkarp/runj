# Building runj

runj is a set of Go programs and requires the Go toolchain to build.  You can
install it with `pkg install go`.  The programs can be built with `make`.  Note
that the default `make` target does not rebuild all files, read further for why.

## Protocol Buffers

The runj containerd shim (`containerd-shim-runj-v1`) defines extensions to shim
options via [protocol buffers](https://developers.google.com/protocol-buffers/).
These consist of schemas (defined in files suffixed with `.proto`) and generated
Go source code (in files suffixed with `.pb.go`).  Both the schemas and
generated Go source code files are checked into this repository and do not need
to be regenerated in the normal case.

However, if you are modifying the schemas, special considerations apply.

### Build tools

Generating new Go source files from the protocol buffer schemas for runj
requires the following tools:

* `protoc` (the protobuf compiler)
* [`protobuild`](https://github.com/containerd/protobuild) (a wrapper around
  `protoc`)
* [`go-fix-acronym`](https://github.com/containerd/protobuild/tree/main/cmd/go-fix-acronym)
  (a tool adjusting field names)
* [`protoc-gen-go`](https://github.com/protocolbuffers/protobuf-go) (Go support
  for protocol buffers)
* [`protoc-gen-go-grpc`](https://github.com/grpc/grpc-go/tree/master/cmd/protoc-gen-go-grpc)
  (Go support for gRPC services in protocol buffers)
* [`protoc-gen-go-ttrpc`](https://github.com/containerd/ttrpc/tree/main/cmd/protoc-gen-go-ttrpc)
  (Go support for TTRPC services in protocol buffers)

`protoc` can be installed with `pkg install protobuf`.  The other dependencies
can be installed with `../script/install-dev-tools.sh`.

### Directory layout

`protoc` and `protobuild` are particular about the directory layout the source
code is in while they are invoked.  For this reason, the `protos` make target is
excluded from the default target and must be invoked explicitly.  To build, you
must:

1. Check this repository out into your `GOPATH` (e.g.,
   `~/go/src/go.sbk.wtf/runj`, use `go env GOPATH` to find it)
2. Ensure that your current working directory matches the `GOPATH` directory.
   If your `/home` is a symlink to `/usr/home`, you may encounter errors here.
   You can force `make(1)` to have its working directory set as the symlink path
   with `make -C $(pwd)`.
3. Build with `make protos` (or `make -C $(pwd) protos`)
