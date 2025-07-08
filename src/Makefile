# Makefile for Zabbix Docker Swarm Plugin

BINARY_NAME=docker-swarm
GO_FILES=$(wildcard *.go)

.PHONY: all build clean build-x86_64 build-arm64 build-all deps fmt vet check help

all: build

build: build-x86_64

# Build for x86_64 Linux (most common server architecture)
build-x86_64: $(BINARY_NAME)-linux-x86_64

$(BINARY_NAME)-linux-x86_64: $(GO_FILES)
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-x86_64 .

# Build for ARM64 Linux (for ARM-based servers)
build-arm64: $(BINARY_NAME)-linux-arm64

$(BINARY_NAME)-linux-arm64: $(GO_FILES)
	GOOS=linux GOARCH=arm64 go build -o $(BINARY_NAME)-linux-arm64 .

# Build for both architectures
build-all: build-x86_64 build-arm64

clean:
	rm -f $(BINARY_NAME)-linux-x86_64 $(BINARY_NAME)-linux-arm64

deps:
	go mod tidy
	go mod download

fmt:
	go fmt ./...

vet:
	@echo "Note: vet only works on Linux due to Zabbix SDK platform limitations"
	GOOS=linux go vet ./...

check: fmt vet

help:
	@echo "Available targets:"
	@echo "  build             - Build for x86_64 Linux (default)"
	@echo "  build-x86_64      - Build for x86_64 Linux"
	@echo "  build-arm64       - Build for ARM64 Linux"
	@echo "  build-all         - Build for both architectures"
	@echo "  clean             - Remove all binaries"
	@echo "  deps              - Download and tidy dependencies"
	@echo "  fmt               - Format Go code"
	@echo "  vet               - Run go vet (Linux only)"
	@echo "  check             - Run fmt and vet"
	@echo "  help              - Show this help" 