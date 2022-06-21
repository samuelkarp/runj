#!/bin/sh
set -ex

GOPATH=$(go env GOPATH)
mkdir -p "${GOPATH}/src/go.sbk.wtf"
ln -s $(pwd) "${GOPATH}/src/go.sbk.wtf/runj"

install -o 0 -g 0 \
  "${GOPATH}"/bin/protobuild \
  "${GOPATH}"/bin/go-fix-acronym \
  "${GOPATH}"/bin/protoc-gen-go \
  "${GOPATH}"/bin/protoc-gen-go-grpc \
  "${GOPATH}"/bin/protoc-gen-go-ttrpc \
  /usr/local/bin
