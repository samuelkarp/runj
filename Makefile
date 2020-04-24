all: runj

runj: bin/runj

bin/runj:
	go build -o bin/runj ./cmd/runj

clean:
	rm -rf bin
