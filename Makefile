SOURCES != find . -name '*.go'

all: binaries

binaries: runj runj-entrypoint containerd-shim-runj

runj: bin/runj
bin/runj: $(SOURCES)
	go build -o bin/runj ./cmd/runj

runj-entrypoint: bin/runj-entrypoint
bin/runj-entrypoint: $(SOURCES)
	go build -o bin/runj-entrypoint ./cmd/runj-entrypoint

containerd-shim-runj: bin/containerd-shim-runj
bin/containerd-shim-runj: $(SOURCES)
	go build -o bin/containerd-shim-runj ./cmd/containerd-shim-runj

.PHONY: install
install: runj containerd-shim-runj
	install -o 0 -g 0 bin/runj bin/runj-entrypoint bin/containerd-shim-runj /usr/local/bin

.PHONY: test
test:
	go test -v ./...

.PHONY: clean
clean:
	rm -rf bin
