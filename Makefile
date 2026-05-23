.PHONY: all build test fmt vet goimports clean run

BINARY_NAME=query-gateway

all: goimports fmt vet build test

build:
	go build -o bin/$(BINARY_NAME) ./cmd

test:
	go test -v -race ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

goimports:
	goimports -w .

clean:
	go clean
	rm -f bin/$(BINARY_NAME)

run: build
	./bin/$(BINARY_NAME)
