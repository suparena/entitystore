# EntityStore Makefile with comprehensive build and release support
.PHONY: all build test test-unit test-integration test-race test-coverage clean lint fmt vet release version help

# Variables
GOPATH := $(shell go env GOPATH)
GOBIN := $(GOPATH)/bin
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# Version information
VERSION := $(shell cat VERSION)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION := $(shell go version | awk '{print $$3}')

# Build flags
LDFLAGS := -X 'github.com/suparena/entitystore.Version=$(VERSION)'
LDFLAGS += -X 'github.com/suparena/entitystore.GitCommit=$(GIT_COMMIT)'
LDFLAGS += -X 'github.com/suparena/entitystore.BuildDate=$(BUILD_DATE)'
LDFLAGS += -X 'github.com/suparena/entitystore.GoVersion=$(GO_VERSION)'

# Output directories
DIST_DIR := dist
BIN_DIR := bin

# Binary name
BINARY_NAME := indexmap-pps

# Test variables
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html

# Default target
all: clean lint test build

# Version management
version:
	@echo "EntityStore Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(GO_VERSION)"

version-bump-patch:
	@echo "Bumping patch version..."
	@./scripts/bump-version.sh patch

version-bump-minor:
	@echo "Bumping minor version..."
	@./scripts/bump-version.sh minor

version-bump-major:
	@echo "Bumping major version..."
	@./scripts/bump-version.sh major

# Build targets
build: build-cli
	@echo "Building all packages..."
	@go build -v -ldflags "$(LDFLAGS)" ./...

build-cli:
	@echo "Building $(BINARY_NAME) CLI..."
	@mkdir -p $(BIN_DIR)
	@go build -v -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/indexmap

build-race:
	@echo "Building with race detector..."
	@go build -race -v -ldflags "$(LDFLAGS)" ./...

build-all: build-linux build-darwin build-windows

build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/indexmap
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/indexmap

build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/indexmap
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/indexmap

build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/indexmap

# Test targets
test: test-unit

test-unit:
	@echo "Running unit tests..."
	@go test -v ./...

test-integration:
	@echo "Running integration tests..."
	@echo "Note: Requires AWS credentials and DDB_TEST_TABLE_NAME environment variable"
	@go test -v -tags=integration ./...

test-race:
	@echo "Running tests with race detector..."
	@go test -race -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@echo "Generating coverage report..."
	@go tool cover -func=$(COVERAGE_FILE)
	@go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

test-short:
	@echo "Running short tests..."
	@go test -short -v ./...

# Benchmark tests
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Code quality targets
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		go vet ./...; \
	fi

fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@if command -v goimports >/dev/null; then \
		goimports -w .; \
	fi

vet:
	@echo "Running go vet..."
	@go vet ./...

# Dependency management
deps:
	@echo "Downloading dependencies..."
	@go mod download

deps-update:
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

deps-verify:
	@echo "Verifying dependencies..."
	@go mod verify

mod-tidy:
	@echo "Tidying go.mod..."
	@go mod tidy

# Clean targets
clean:
	@echo "Cleaning build artifacts..."
	@go clean -testcache
	@rm -rf $(DIST_DIR) $(BIN_DIR)
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)

# Docker targets for integration testing
docker-dynamodb-start:
	@echo "Starting DynamoDB Local..."
	@docker run -d --name dynamodb-local \
		-p 8000:8000 \
		amazon/dynamodb-local \
		-jar DynamoDBLocal.jar -inMemory -sharedDb

docker-dynamodb-stop:
	@echo "Stopping DynamoDB Local..."
	@docker stop dynamodb-local || true
	@docker rm dynamodb-local || true

# Release targets
release-dry-run:
	@echo "Performing dry run of release process..."
	@echo "Current version: $(VERSION)"
	@echo "Would tag as: v$(VERSION)"
	@echo "Would build for: linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64"

release-local: clean build-all
	@echo "Creating local release..."
	@mkdir -p $(DIST_DIR)/release-$(VERSION)
	@cp $(DIST_DIR)/* $(DIST_DIR)/release-$(VERSION)/
	@cp README.md LICENSE CHANGELOG.md $(DIST_DIR)/release-$(VERSION)/ 2>/dev/null || true
	@cd $(DIST_DIR) && tar czf entitystore-$(VERSION).tar.gz release-$(VERSION)
	@cd $(DIST_DIR) && zip -r entitystore-$(VERSION).zip release-$(VERSION)
	@echo "Local release created in $(DIST_DIR)/"

release: release-check clean test build-all release-tag
	@echo "Release $(VERSION) complete!"
	@echo "Don't forget to push tags: git push origin v$(VERSION)"

release-check:
	@echo "Checking release prerequisites..."
	@if [ -z "$(VERSION)" ]; then echo "VERSION is not set"; exit 1; fi
	@if git tag -l "v$(VERSION)" | grep -q "v$(VERSION)"; then \
		echo "Tag v$(VERSION) already exists!"; exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Working directory not clean! Commit or stash changes first."; exit 1; \
	fi
	@echo "Release check passed!"

release-tag:
	@echo "Creating git tag v$(VERSION)..."
	@git tag -a "v$(VERSION)" -m "Release version $(VERSION)"
	@echo "Tag created. Push with: git push origin v$(VERSION)"

# Checksums
checksums:
	@echo "Generating checksums..."
	@cd $(DIST_DIR) && sha256sum $(BINARY_NAME)-* > checksums.txt

# CI/CD helpers
ci-test: lint test-race test-coverage

ci-integration: docker-dynamodb-start test-integration docker-dynamodb-stop

# Documentation
docs:
	@echo "Generating documentation..."
	@go doc -all ./... > API_DOCS.txt
	@echo "Checking documentation coverage..."
	@go list ./... | xargs -I {} sh -c 'echo "Package: {}" && go doc -short {}'

godoc-serve:
	@echo "Starting godoc server on http://localhost:6060"
	@godoc -http=:6060

# Installation
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install -ldflags "$(LDFLAGS)" ./cmd/indexmap

uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(GOBIN)/$(BINARY_NAME)

# Development helpers
dev-setup:
	@echo "Setting up development environment..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/go-delve/delve/cmd/dlv@latest
	@go install golang.org/x/tools/cmd/godoc@latest
	@echo "Development tools installed!"

# Generate code
generate:
	@echo "Running code generation..."
	@go generate ./...

# Security scan
security:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null; then \
		gosec -fmt=json -out=security-report.json ./...; \
		echo "Security report generated: security-report.json"; \
	else \
		echo "gosec not installed. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi

# Show available targets
help:
	@echo "EntityStore Makefile - Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  all              - Clean, lint, test, and build"
	@echo "  build            - Build all packages"
	@echo "  build-cli        - Build the CLI tool"
	@echo "  build-all        - Build for all platforms"
	@echo "  build-race       - Build with race detector"
	@echo ""
	@echo "Test targets:"
	@echo "  test             - Run unit tests"
	@echo "  test-integration - Run integration tests (requires AWS)"
	@echo "  test-race        - Run tests with race detector"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  bench            - Run benchmarks"
	@echo ""
	@echo "Code quality:"
	@echo "  lint             - Run linters"
	@echo "  fmt              - Format code"
	@echo "  vet              - Run go vet"
	@echo "  security         - Run security scan"
	@echo ""
	@echo "Release targets:"
	@echo "  version          - Show version information"
	@echo "  release          - Create a new release"
	@echo "  release-dry-run  - Preview release process"
	@echo "  release-local    - Create local release artifacts"
	@echo ""
	@echo "Other targets:"
	@echo "  deps             - Download dependencies"
	@echo "  clean            - Clean build artifacts"
	@echo "  help             - Show this help message"

# Default build script compatibility
legacy-build:
	@./build.sh