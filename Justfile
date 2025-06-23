# EntityStore Justfile
# Run 'just' to see available commands

# Default command - show help
default:
    @just --list

# Development environment setup
setup:
    @echo "Setting up development environment..."
    go mod download
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    @echo "Setup complete!"

# Build the project
build:
    go build ./...

# Build the indexmap preprocessor
build-indexmap:
    go build -ldflags "-X 'github.com/suparena/entitystore.Version=$(cat VERSION)'" \
        -o indexmap-pps ./cmd/indexmap

# Run all tests
test:
    go test ./...

# Run tests with coverage
test-coverage:
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report generated: coverage.html"

# Run integration tests (requires AWS credentials)
test-integration:
    AWS_REGION=${AWS_REGION:-us-east-1} go test -tags=integration ./...

# Run specific test pattern
test-run pattern:
    go test -run {{pattern}} -v ./...

# Run GSI-related tests
test-gsi:
    go test -run "GSI|Gsi" -v ./datastore/ddb/

# Run time-based query tests
test-time:
    go test -run "Time" -v ./datastore/ddb/

# Lint the code
lint:
    golangci-lint run

# Format code
fmt:
    go fmt ./...
    gofmt -s -w .

# Clean build artifacts
clean:
    rm -rf dist/ coverage.out coverage.html indexmap-pps

# Setup DynamoDB table
setup-dynamodb table_name='entitystore-table':
    @chmod +x scripts/setup-dynamodb-table.sh
    ./scripts/setup-dynamodb-table.sh {{table_name}}

# Verify DynamoDB setup
verify-dynamodb table_name='entitystore-table':
    @chmod +x scripts/verify-dynamodb-setup.sh
    ./scripts/verify-dynamodb-setup.sh {{table_name}}

# Run local DynamoDB for testing
dynamodb-local:
    docker run -d -p 8000:8000 \
        --name dynamodb-local \
        amazon/dynamodb-local \
        -jar DynamoDBLocal.jar -sharedDb -inMemory

# Stop local DynamoDB
dynamodb-local-stop:
    docker stop dynamodb-local && docker rm dynamodb-local

# Build for all platforms
build-all: clean
    @echo "Building for all platforms..."
    @mkdir -p dist
    GOOS=linux GOARCH=amd64 go build -ldflags "-X 'github.com/suparena/entitystore.Version=$(cat VERSION)'" \
        -o dist/indexmap-pps-linux-amd64 ./cmd/indexmap
    GOOS=linux GOARCH=arm64 go build -ldflags "-X 'github.com/suparena/entitystore.Version=$(cat VERSION)'" \
        -o dist/indexmap-pps-linux-arm64 ./cmd/indexmap
    GOOS=darwin GOARCH=amd64 go build -ldflags "-X 'github.com/suparena/entitystore.Version=$(cat VERSION)'" \
        -o dist/indexmap-pps-darwin-amd64 ./cmd/indexmap
    GOOS=darwin GOARCH=arm64 go build -ldflags "-X 'github.com/suparena/entitystore.Version=$(cat VERSION)'" \
        -o dist/indexmap-pps-darwin-arm64 ./cmd/indexmap
    GOOS=windows GOARCH=amd64 go build -ldflags "-X 'github.com/suparena/entitystore.Version=$(cat VERSION)'" \
        -o dist/indexmap-pps-windows-amd64.exe ./cmd/indexmap

# Bump version (patch, minor, or major)
bump-version type='patch':
    @chmod +x scripts/bump-version.sh
    ./scripts/bump-version.sh {{type}}

# Create a release
release version:
    @echo "Creating release {{version}}..."
    @echo {{version}} > VERSION
    @sed -i.bak 's/Version = ".*"/Version = "{{version}}"/' version.go && rm version.go.bak
    @git add VERSION version.go
    @git commit -m "chore: bump version to {{version}}"
    @git tag -a v{{version}} -m "Release v{{version}}"
    @git push origin main
    @git push origin v{{version}}
    @just build-all
    @gh release create v{{version}} dist/* --title "v{{version}}" --generate-notes

# Run benchmarks
bench:
    go test -bench=. -benchmem ./...

# Check for module updates
check-updates:
    go list -u -m all

# Update dependencies
update-deps:
    go get -u ./...
    go mod tidy

# Run a full CI check
ci: fmt lint test
    @echo "CI checks passed!"

# Start development environment
dev:
    @echo "Starting development environment..."
    @just setup
    @just verify-dynamodb
    @echo "Ready for development!"

# Generate documentation
docs:
    @echo "Generating documentation..."
    godoc -http=:6060 &
    @echo "Documentation server started at http://localhost:6060"

# Run security scan
security:
    @echo "Running security scan..."
    @go list -json -m all | nancy sleuth || true
    @gosec ./... || true

# Show current version
version:
    @cat VERSION

# Run quick check (format, build, test)
check: fmt build test
    @echo "Quick check passed!"

# Install Just (for setup instructions)
install-just:
    @echo "To install Just:"
    @echo "  macOS:  brew install just"
    @echo "  Linux:  curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin"
    @echo "  Other:  https://github.com/casey/just#installation"