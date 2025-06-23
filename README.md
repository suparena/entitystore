# EntityStore

A sophisticated Go library for type-safe, annotation-driven data persistence with support for multiple storage backends.

[![Version](https://img.shields.io/badge/version-0.2.2-blue.svg)](CHANGELOG.md)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)](go.mod)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](LICENSE)

## Features

- **Type-Safe Operations**: Generic interfaces with compile-time type checking
- **Annotation-Driven**: OpenAPI vendor extensions drive code generation
- **Multiple Backends**: DynamoDB (primary), Redis, with more planned
- **Enhanced Streaming**: Efficient processing of large datasets with configurable options
- **Semantic Error Types**: Clear error handling with custom error types
- **Single Table Design**: Optimized for DynamoDB best practices
- **Code Generation**: Automatic type registration from OpenAPI specs

## Installation

```bash
go get github.com/suparena/entitystore@latest
```

## Prerequisites

See [Prerequisites Guide](docs/PREREQUISITES.md) for detailed requirements.

### Quick Setup with Just

```bash
# Install Just (command runner)
brew install just  # macOS
# or see https://github.com/casey/just#installation

# Setup development environment
just setup

# Setup DynamoDB table
just setup-dynamodb my-table

# Verify setup
just verify-dynamodb my-table
```

For detailed setup instructions, see the [Setup Guide](docs/SETUP_GUIDE.md).

## Quick Start

### 1. Define Your Entity

```go
type User struct {
    ID        string    `json:"id"`
    Email     string    `json:"email"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"createdAt"`
}
```

### 2. Register with OpenAPI Annotations

```yaml
User:
  type: object
  x-dynamodb-indexmap:
    PK: "USER#{ID}"        # Maps to DynamoDB attribute 'PK'
    SK: "USER#{ID}"        # Maps to DynamoDB attribute 'SK'
    GSI1PK: "EMAIL#{Email}" # Maps to DynamoDB attribute 'PK1' (GSI1 partition key)
    GSI1SK: "USER"         # Maps to DynamoDB attribute 'SK1' (GSI1 sort key)
```

**Note**: EntityStore automatically maps logical GSI names (GSI1PK/GSI1SK) to physical DynamoDB attribute names (PK1/SK1).

### 3. Use the DataStore

```go
// Create datastore
store, err := ddb.NewDynamodbDataStore[User](
    awsAccessKey, awsSecretKey, region, tableName,
)

// Store entity
user := User{ID: "123", Email: "user@example.com", Name: "John"}
err = store.Put(ctx, user)

// Retrieve entity
retrieved, err := store.GetOne(ctx, "123")
if errors.IsNotFound(err) {
    // Handle not found
}

// Query by GSI (email lookup)
results, err := store.QueryByGSI1PK(ctx, "user@example.com")

// Complex GSI query
results, err := store.QueryGSI().
    WithPartitionKey("user@example.com").
    WithSortKeyPrefix("STATUS#active").
    WithLimit(10).
    Execute(ctx)

// Stream large results
params := &storagemodels.QueryParams{
    TableName: tableName,
    KeyConditionExpression: "PK = :pk",
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":pk": &types.AttributeValueMemberS{Value: "USER#123"},
    },
}

for result := range store.Stream(ctx, params) {
    if result.Error != nil {
        log.Printf("Error: %v", result.Error)
        continue
    }
    process(result.Item)
}
```

## Recent Improvements

### Enhanced Streaming API
- Single-channel design for simpler usage
- Configurable buffering, retries, and progress tracking
- Built-in error recovery and retry logic
- Per-item metadata for debugging

### Better Error Handling
- Semantic error types (`NotFound`, `AlreadyExists`, `ValidationError`, etc.)
- Helper functions for error checking
- Consistent error wrapping with context

### Type-Safe Storage with Generics
- New `MultiTypeStorage` for type-safe datastore management
- No more type assertions when retrieving datastores
- Compile-time type checking
- [Migration Guide](docs/MIGRATION_GUIDE.md) available

### GSI Query Optimization
- Fluent query builder for GSI queries
- Convenience methods for common patterns
- Support for complex filtering and sorting
- [GSI Optimization Guide](docs/GSI_OPTIMIZATION_GUIDE.md) available

### Testing Support
- Comprehensive mock implementation in `datastore/mock`
- Configurable error injection
- Thread-safe test fixtures
- Helper methods for test setup

### Thread Safety
- All storage managers use proper mutex protection
- Safe for concurrent use

### Consolidated APIs
- Unified `QueryParams` for both regular queries and streaming
- Cleaner, more consistent interfaces

## Documentation

- [User Guide](docs/USER_GUIDE.md) - Complete usage guide with examples
- [System Design](docs/SYSTEM_DESIGN.md) - Architecture and design patterns
- [Quick Reference](docs/entitystore-quick-reference.md) - Common operations cheat sheet
- [Design Documentation](docs/entitystore-design.md) - Comprehensive design patterns

## Building

```bash
# Build the project
go build ./...

# Run tests
go test ./...

# Build code generator
./build.sh
```

## Contributing

We welcome contributions! Please see our contributing guidelines for details.

## License

Copyright © 2025 Suparena Software Inc. All rights reserved.