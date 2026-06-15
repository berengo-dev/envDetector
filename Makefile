.PHONY: build test clean

BINARY := env-doctor

build:
	go build -o $(BINARY) ./cmd/env-doctor

test:
	go test ./...

clean:
	rm -f $(BINARY)
