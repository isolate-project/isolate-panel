.PHONY: help run build test lint clean

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
	go build -o bin/server cmd/server/main.go

test:
	go test -v ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/ tmp/
