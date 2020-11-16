SOURCES != find . -name '*.go'

all: binaries

binaries: runj runj-entrypoint

runj: bin/runj
bin/runj: $(SOURCES)
	go build -o bin/runj ./cmd/runj

runj-entrypoint: bin/runj-entrypoint
bin/runj-entrypoint: $(SOURCES)
	go build -o bin/runj-entrypoint ./cmd/runj-entrypoint

.PHONY: test
test:
	go test -v ./...

.PHONY: clean
clean:
	rm -rf bin
