#!/bin/sh

go install github.com/containerd/protobuild@v0.2.0
go install github.com/containerd/protobuild/cmd/go-fix-acronym@v0.2.0
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
go install github.com/containerd/ttrpc/cmd/protoc-gen-go-ttrpc@944ef4a40df3446714a823207972b7d9858ffac5
