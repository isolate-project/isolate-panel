.PHONY: help run build test lint clean

VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
PKG        := github.com/isolate-project/isolate-panel/internal/version
LDFLAGS    := -X $(PKG).Version=$(VERSION) \
              -X $(PKG).BuildDate=$(BUILD_DATE) \
              -X $(PKG).GitCommit=$(GIT_COMMIT)

help:
	@echo "Available commands:"
	@echo "  make run     - Run the server"
	@echo "  make build   - Build the binary"
	@echo "  make test    - Run tests"
	@echo "  make lint    - Run linter"
	@echo "  make clean   - Clean build artifacts"

run:
	go run cmd/server/main.go

build:
	go build -ldflags "$(LDFLAGS)" -o bin/server cmd/server/main.go

test:
	go test -v ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/ tmp/
