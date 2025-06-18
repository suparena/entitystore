# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

EntityStore is a Go library implementing a sophisticated storage abstraction pattern that enables type-safe, annotation-driven data persistence. It follows a design-time → build-time → runtime workflow where OpenAPI specifications with vendor extensions drive code generation for automatic type registration and index mapping. The library supports DynamoDB (primary), Redis, and future backends through a pluggable architecture, enabling single-table design patterns and complex query capabilities.

## Commands

### Build
```bash
# Build the indexmap preprocessor tool
./build.sh

# Build the entire project
go build ./...
```

### Testing
```bash
# Run all unit tests (no AWS required)
go test ./...

# Run unit tests with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run integration tests (requires AWS credentials)
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=your_key_id
export AWS_SECRET_ACCESS_KEY=your_secret_key
export DDB_TEST_TABLE_NAME=your_test_table
go test -tags=integration ./...

# Run specific test
go test -run TestDynamodbDataStore ./datastore/ddb/

# Run tests with race detection
go test -race ./...

### Development
```bash
# Get dependencies
go mod download

# Tidy dependencies
go mod tidy

# Two-phase code generation:
# 1. Generate API models from OpenAPI (using swagger-codegen or similar)
# 2. Generate EntityStore registrations
go run cmd/indexmap/main.go -input path/to/openapi.yaml -output generated/entitystore_gen.go

# Alternative: Use built binary
./indexmap-pps -input api.yaml -output generated/entitystore_gen.go
```

## Architecture

### Workflow: Design-Time → Build-Time → Runtime

1. **Design-Time**: Add `x-dynamodb-indexmap` annotations to OpenAPI specs
2. **Build-Time**: Code generation creates type registrations
3. **Runtime**: Storage manager provides type-safe data access

### Core Abstractions

1. **Storage Interface** (`entitystore.go`): Non-generic top-level interface that manages multiple DataStore instances. The `StorageManager` allows registering different data stores by key (e.g., "core.UserProfile").

2. **DataStore Layer** (`datastore/datastore.go`): Generic interface `DataStore[T]` providing CRUD and batch operations:
   - `GetOne`: Retrieve by key
   - `Put`: Store entity
   - `UpdateWithCondition`: Conditional updates with DynamoDB expressions
   - `Query`: Query with filters, GSI support, pagination
   - `Stream`: Streaming results for large datasets
   - `Delete`: Remove entity
   - `BatchGet`, `BatchPut`, `BatchDelete`: Bulk operations (25 item limit)

3. **DynamoDB Implementation** (`datastore/ddb/`): Primary implementation supporting:
   - Macro expansion for dynamic keys (e.g., `"USER#{ID}"`, `{Email}`)
   - Complex query patterns with GSI support
   - Streaming for memory-efficient large result processing
   - Conditional updates for optimistic locking
   - Automatic EntityType attribute for polymorphic storage

### Registry System

- **Type Registry** (`registry/types.go`): Maps type names to unmarshal functions
  - Simple names for OpenAPI models: `"UserProfile"`
  - Package-qualified for core types: `"core.UserProfile"`
- **Index Map Registry** (`registry/indexmap.go`): Associates Go types with index patterns
  - Supports PK, SK, GSI1PK, GSI1SK, GSI2PK, GSI2SK patterns
  - Macro syntax: `{Field}`, `PREFIX#{Field}`, `"LITERAL"`

### Code Generation

The `processor/postprocess.go` reads OpenAPI specs with `x-dynamodb-indexmap` vendor extensions:
```yaml
x-dynamodb-indexmap:
  PK: "USER#{UserId}"
  SK: "PROFILE"
  GSI1PK: "EMAIL#{Email}"
```
Generates automatic type registration and index mapping code.

## Key Design Patterns

1. **Repository Pattern**: Services use type-safe repositories for data access
2. **Single Table Design**: Multiple entity types in one DynamoDB table using composite keys
3. **Annotation-Driven**: OpenAPI vendor extensions drive code generation
4. **Generic Type Safety**: Compile-time type checking with Go generics
5. **Pluggable Storage**: Abstract interface supports multiple backends (DynamoDB, Redis, future: PostgreSQL, MongoDB)
6. **Streaming Pattern**: Memory-efficient processing of large result sets

## Important Usage Patterns

### OpenAPI Annotation
```yaml
UserProfile:
  x-dynamodb-indexmap:
    PK: "USER#{UserId}"     # Composite key with prefix
    SK: "PROFILE"           # Static sort key
    GSI1PK: "EMAIL#{Email}" # GSI for email lookups
```

### Service Layer Usage
```go
// Get typed datastore
ds, _ := storage.GetDataStore("core.UserProfile")
userStore := ds.(datastore.DataStore[core.UserProfile])

// CRUD operations
profile := &core.UserProfile{UserId: "123", Email: "user@example.com"}
err := userStore.Put(ctx, *profile)
retrieved, _ := userStore.GetOne(ctx, "123")
```

## Recent Enhancements

### Phase 1: Core Improvements
- **Enhanced Streaming**: Single-channel API with retry logic, progress tracking, and configurable options
- **Semantic Error Types**: Custom error package with `NotFound`, `ValidationError`, etc.
- **Thread Safety**: Fixed mutex protection in storage managers
- **API Consolidation**: Unified `QueryParams` for queries and streaming

### Phase 2: Type Safety & Testing
- **Type-Safe Storage**: New `MultiTypeStorage` with compile-time type checking
- **Mock Implementation**: Comprehensive mock in `datastore/mock` for testing
- **Migration Support**: Backward compatible with migration guide

## Documentation

Comprehensive documentation available in `/docs/`:
- `entitystore-design.md`: Full architecture and design patterns (745 lines)
- `entitystore-guide.md`: Implementation guide and workflow (353 lines)
- `entitystore-quick-reference.md`: Quick reference and examples (419 lines)
- `SYSTEM_DESIGN.md`: Technical system design with streaming patterns
- `USER_GUIDE.md`: Practical usage guide with error handling examples
- `MIGRATION_GUIDE.md`: Guide for migrating to type-safe storage
- `README.md`: Project overview and quick start

## Package Structure

- `/errors`: Semantic error types and helper functions
- `/datastore/mock`: Mock implementation for testing
- `/generic_storage.go`: Type-safe storage implementation
- `/datastore/ddb/stream.go`: Enhanced streaming implementation