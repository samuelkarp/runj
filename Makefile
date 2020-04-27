SOURCES != find . -name '*.go'

all: runj

runj: bin/runj
bin/runj: $(SOURCES)
	go build -o bin/runj ./cmd/runj

.PHONY: test
test:
	go test -v ./...

clean:
	rm -rf bin
