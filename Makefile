SOURCES != find . -name '*.go'

all: binaries NOTICE

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

NOTICE: go.mod go.sum
	go-licenses report --template hack/notice.tpl ./... > NOTICE

.PHONY:
verify-notice:
	mv NOTICE NOTICE.bak
	$(MAKE) NOTICE
	diff NOTICE.bak NOTICE
	rm NOTICE.bak

.PHONY: clean
clean:
	@rm -rf bin
	@rm NOTICE.bak 2>/dev/null ||:
