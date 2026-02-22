.PHONY: build run test lint fmt clean install snapshot release help

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"
PREFIX ?= $(HOME)/.local

build:
	go build $(LDFLAGS) -o bin/adhanctl ./cmd/adhanctl

run:
	go run ./cmd/adhanctl

test:
	go test -v ./...

lint:
	go vet ./...

fmt:
	go fmt ./...

clean:
	rm -rf bin/ dist/

install: build
	install -Dm755 bin/adhanctl $(PREFIX)/bin/adhanctl

snapshot:
	goreleaser build --snapshot --clean

release:
	goreleaser release --clean

help:
	@echo "adhanctl Makefile"
	@echo ""
	@echo "Commands:"
	@echo "  build    Build the binary"
	@echo "  run      Run the application"
	@echo "  test     Run tests"
	@echo "  lint     Run go vet"
	@echo "  fmt      Format code"
	@echo "  clean    Remove build artifacts"
	@echo "  install  Install binary (default: ~/.local/bin)"
	@echo "           Use PREFIX=/usr/local for system-wide install"
	@echo "  snapshot Build snapshot with goreleaser"
	@echo "  release  Create release with goreleaser"
