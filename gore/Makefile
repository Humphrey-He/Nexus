# ===============================================================================
# Gore Lint Makefile
# ===============================================================================

# Variables
BINARY_NAME := gore-lint
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown') -X main.date=$(BUILD_TIME)

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt
GOVET := $(GOCMD) vet

# Docker
DOCKER_IMAGE := goreio/gore-lint
DOCKER_TAG := $(VERSION)

# Directories
BUILD_DIR := ./build
COVERAGE_DIR := ./coverage

# OS detection
OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
ARCH := $(shell uname -m)

# ===============================================================================
# Targets
# ===============================================================================

.PHONY: all build test test-unit test-integration test-coverage lint lint-fix
.PHONY: clean fmt vet deps deps-update docker docker-build docker-push
.PHONY: install uninstall benchmark help

# Default target
all: build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/gore-lint

## test: Run all tests
test: test-unit test-integration

## test-unit: Run unit tests only
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -race -tags=unit -coverprofile=$(COVERAGE_DIR)/unit.out ./...

## test-integration: Run integration tests (requires MySQL)
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -race -tags=integration -coverprofile=$(COVERAGE_DIR)/integration.out ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"

## lint: Run linters
lint:
	@echo "Running golangci-lint..."
	golangci-lint run ./...

## lint-fix: Run linters with auto-fix
lint-fix:
	@echo "Running golangci-lint with fixes..."
	golangci-lint run --fix ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

## deps-update: Update dependencies
deps-update:
	@echo "Updating dependencies..."
	$(GOMOD) tidy
	$(GOMOD) verify

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) $(COVERAGE_DIR)
	@rm -f coverage.out
	@find . -name '*.test' -delete
	@find . -name '*_mock.go' -delete

## install: Install binary to GOBIN
install:
	@echo "Installing $(BINARY_NAME)..."
	CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(shell go env GOPATH)/bin/$(BINARY_NAME) ./cmd/gore-lint

## uninstall: Remove binary from GOBIN
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(shell go env GOPATH)/bin/$(BINARY_NAME)

## benchmark: Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem -tags=integration ./...

## docker: Build Docker image
docker:
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build --build-arg VERSION=$(VERSION) -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest

## docker-build: Build Docker image for multiple platforms
docker-build:
	@echo "Building Docker image for multiple platforms..."
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		--output type=image \
		.

## docker-push: Push Docker image to registry
docker-push:
	@echo "Pushing Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):latest

## help: Show this help
help:
	@echo "Gore Lint Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN { FS = ":.*?## " } /^[a-zA-Z_-]+:.*?## / { printf "  %-20s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# ===============================================================================
# Development helpers
# ===============================================================================

## dev: Run in development mode (with hot reload - requires air)
dev:
	@echo "Running in development mode..."
	air

## test-mysql: Run MySQL integration tests only
test-mysql:
	@echo "Running MySQL integration tests..."
	GORE_DB_DSN="root:root@tcp(localhost:3306)/gore_test" $(GOTEST) -v -race ./dialect/mysql/...

## lint-docker: Lint Dockerfile
lint-docker:
	@echo "Linting Dockerfile..."
	hadolint Dockerfile

## security: Run security checks
security:
	@echo "Running security checks..."
	gosec ./...
	govulncheck ./...
