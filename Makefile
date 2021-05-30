SOURCES != find . -name '*.go'

all: binaries

binaries: runj runj-entrypoint containerd-shim-runj-v1

runj: bin/runj
bin/runj: $(SOURCES) go.mod go.sum
	go build -o bin/runj ./cmd/runj

runj-entrypoint: bin/runj-entrypoint
bin/runj-entrypoint: $(SOURCES) go.mod go.sum
	go build -o bin/runj-entrypoint ./cmd/runj-entrypoint

containerd-shim-runj-v1: bin/containerd-shim-runj-v1
bin/containerd-shim-runj-v1: $(SOURCES) go.mod go.sum
	go build -o bin/containerd-shim-runj-v1 ./cmd/containerd-shim-runj-v1

.PHONY: install
install: runj containerd-shim-runj-v1
	install -o 0 -g 0 bin/runj bin/runj-entrypoint bin/containerd-shim-runj-v1 /usr/local/bin

.PHONY: test
test:
	go test -v ./...

.PHONY: lint
lint:
	GOOS=freebsd golangci-lint run

.PHONY: clean
clean:
	rm -rf bin
