# Build variables
BINARY_NAME=s0cmd
BUILD_DIR=build
VERSION := `git describe --abbrev=0 --tags || echo "v0.0.0"`
COMMIT=$(shell git rev-parse --short HEAD)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X github.com/xianml/s0cmd/cmd.Version=$(VERSION) -X github.com/xianml/s0cmd/cmd.GitCommit=$(COMMIT) -X github.com/xianml/s0cmd/cmd.BuildTime=$(BUILD_TIME)"

# Go related variables
GOFILES=$(shell find . -type f -name '*.go')
GOPATH=$(shell go env GOPATH)

# Default target
.PHONY: all
all: clean build

# Build binary
.PHONY: build
build:
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./main.go

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Install binary
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

# Cross compilation
.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./main.go

.PHONY: build-darwin
build-darwin:
	@echo "Building for MacOS..."
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./main.go

# Development
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

.PHONY: vet
vet:
	@echo "Vetting code..."
	go vet ./...