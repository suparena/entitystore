# Test Automation Guide

This guide describes the test automation setup for EntityStore.

## Overview

EntityStore uses a comprehensive test automation strategy:
- Unit tests for individual components
- Integration tests with DynamoDB Local
- Mock implementations for isolated testing
- CI/CD pipelines with GitHub Actions
- Code coverage tracking with Codecov

## Running Tests

### Quick Start

```bash
# Run all unit tests
make test

# Run tests with coverage
make test-coverage

# Run tests with race detection
make test-race

# Run integration tests (requires setup)
make test-integration
```

### Manual Commands

```bash
# Unit tests only
go test ./...

# With verbose output
go test -v ./...

# Specific package
go test -v ./datastore/ddb/...

# Specific test
go test -v -run TestStreamWithOptions ./datastore/ddb/
```

## Integration Testing

### Setup DynamoDB Local

```bash
# Using the provided script
./scripts/setup-dynamodb-local.sh

# Or manually with Docker
docker run -d --name dynamodb-local \
  -p 8000:8000 \
  amazon/dynamodb-local:latest
```

### Run Integration Tests

```bash
# Set environment variables
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-east-1
export DDB_TEST_TABLE_NAME=entitystore-test
export DYNAMODB_ENDPOINT=http://localhost:8000

# Run integration tests
go test -v -tags=integration ./...
```

## Test Coverage

### Generate Coverage Report

```bash
# Using the coverage script
./scripts/coverage.sh

# Or manually
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Coverage Goals
- Target: 80% overall coverage
- Critical paths: 90%+ coverage
- Mock packages excluded from coverage

## CI/CD Pipeline

### GitHub Actions Workflows

1. **CI Workflow** (`.github/workflows/ci.yml`)
   - Runs on push to main/develop and PRs
   - Linting with golangci-lint
   - Unit tests on multiple Go versions
   - Integration tests with DynamoDB Local
   - Security scanning with gosec
   - Code coverage upload to Codecov

2. **Release Workflow** (`.github/workflows/release.yml`)
   - Triggered by version tags (v*)
   - Builds binaries for multiple platforms
   - Creates GitHub releases

### Local CI Simulation

```bash
# Run the same checks as CI
make ci-test

# With integration tests
make ci-integration
```

## Mock Usage

### Basic Mock Example

```go
import "github.com/suparena/entitystore/datastore/mock"

func TestMyService(t *testing.T) {
    // Create mock store
    mockStore := mock.New[User]().
        WithGetKeyFunc(func(u User) string { return u.ID })
    
    // Inject into service
    service := NewUserService(mockStore)
    
    // Test service methods
    user := User{ID: "123", Name: "Test"}
    err := service.CreateUser(ctx, user)
    assert.NoError(t, err)
}
```

### Error Injection

```go
// Simulate errors
mockStore.WithPutError(errors.NewValidationError("email", "required"))
mockStore.WithDeleteError(errors.NewNotFoundError("User", "123"))
```

## Best Practices

### Test Organization

1. **Unit Tests**: Same package, `*_test.go` files
2. **Integration Tests**: Use build tags `//go:build integration`
3. **Benchmarks**: Separate `*_bench_test.go` files
4. **Test Data**: Use `testdata/` directories

### Test Naming

```go
// Good test names
func TestDataStore_GetOne_Success(t *testing.T)
func TestDataStore_GetOne_NotFound(t *testing.T)
func TestDataStore_Put_ValidationError(t *testing.T)
```

### Table-Driven Tests

```go
func TestOperations(t *testing.T) {
    tests := []struct {
        name    string
        input   Entity
        want    Result
        wantErr bool
    }{
        // Test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Debugging Tests

### Verbose Output
```bash
go test -v ./...
```

### Run Specific Test
```bash
go test -run TestName ./package
```

### Debug with Delve
```bash
dlv test ./package -- -test.run TestName
```

### Race Detection
```bash
go test -race ./...
```

## Continuous Improvement

### Adding New Tests

1. Write unit tests for new features
2. Add integration tests for storage operations
3. Update mocks if interfaces change
4. Ensure coverage remains above threshold

### Performance Testing

```bash
# Run benchmarks
go test -bench=. -benchmem ./...

# Compare benchmarks
go test -bench=. -benchmem ./... > new.txt
benchcmp old.txt new.txt
```

## Troubleshooting

### Common Issues

1. **DynamoDB Local not starting**
   - Check Docker is running
   - Ensure port 8000 is free
   - Check Docker logs: `docker logs dynamodb-local`

2. **Integration tests failing**
   - Verify environment variables are set
   - Check DynamoDB Local is accessible
   - Ensure test table exists

3. **Coverage below threshold**
   - Run coverage report to identify gaps
   - Focus on error paths and edge cases
   - Add table-driven tests for better coverage

### Getting Help

- Check test output for detailed errors
- Run tests with `-v` flag for more information
- Review CI logs in GitHub Actions
- Use debugger for complex issues