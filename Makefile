SOURCES != find . -name '*.go'
PROTOS != find . -name '*.proto'

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

bin/integration: $(SOURCES) go.mod go.sum
	go test -c ./test/integration -o ./bin/integration -tags integration

bin/integ-inside: $(SOURCES) go.mod go.sum
	CGO_ENABLED=0 go test -c ./test/integration -o ./bin/integ-inside -tags inside

.PHONY: integ-test
integ-test: bin/integration bin/integ-inside
	sudo bin/integration -test.v

protos: .protos-stamp
.protos-stamp: $(PROTOS)
	@find . -path ./vendor -prune -false -o -name '*.pb.go' | xargs -r rm
	go list ./... | grep -v vendor | xargs protobuild
	go-fix-acronym -w -a '(Id|Io|Uuid|Os|Ip)$$' $(shell find . -path ./vendor -name '*.pb.go')
	touch .protos-stamp

.PHONY: clean-protos
	rm .protos-stamp

.PHONY: clean
clean:
	rm -rf bin
