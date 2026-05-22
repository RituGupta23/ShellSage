BINARY     := sg
BUILD_DIR  := ./dist
CMD_DIR    := ./cmd

# Version injection — uses git tag if available, falls back to "dev".
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X github.com/shellsage/sg/internal/cli.Version=$(VERSION) \
	-X github.com/shellsage/sg/internal/cli.Commit=$(COMMIT) \
	-X github.com/shellsage/sg/internal/cli.BuildDate=$(BUILD_DATE)

.PHONY: all build install test clean lint tidy help

## all: build the binary (default target)
all: build

## build: compile the sg binary into ./dist/
build:
	@mkdir -p $(BUILD_DIR)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) $(CMD_DIR)
	@echo "Built $(BUILD_DIR)/$(BINARY)"

## install: build and install to GOPATH/bin (or ~/go/bin)
install:
	go install -trimpath -ldflags "$(LDFLAGS)" $(CMD_DIR)
	@echo "Installed $(BINARY) to $$(go env GOPATH)/bin"

## test: run all unit tests with race detector
test:
	go test -race -count=1 ./...

## test-cover: run tests and show coverage summary
test-cover:
	go test -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1

## lint: run golangci-lint (must be installed separately)
lint:
	golangci-lint run ./...

## tidy: tidy and verify go.mod / go.sum
tidy:
	go mod tidy
	go mod verify

## clean: remove build artifacts
clean:
	@rm -rf $(BUILD_DIR) coverage.out
	@echo "Cleaned"

## help: print this help message
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //'
