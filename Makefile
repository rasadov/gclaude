BINARY=gclaude
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

.PHONY: all build install clean test

all: build

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/gclaude

install:
	go install $(LDFLAGS) ./cmd/gclaude

clean:
	rm -f $(BINARY)

test:
	go test -v ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run

deps:
	go mod tidy

run:
	go run ./cmd/gclaude
