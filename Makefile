SOURCES := $(shell find . -name '*.go')

all: runj

runj: bin/runj
bin/runj: $(SOURCES)
	go build -o bin/runj ./cmd/runj

clean:
	rm -rf bin
