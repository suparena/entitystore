# EntityStore System Design

## Overview

EntityStore is a Go library that provides a type-safe, generic storage abstraction layer optimized for AWS DynamoDB. It supports multiple entity types within a single table through a flexible registry system and macro-based key mapping.

## Architecture

### Core Components

```
┌─────────────────────┐
│   Application Code  │
└──────────┬──────────┘
           │
┌──────────▼──────────┐
│   Storage Manager   │ ◄── Non-generic interface
└──────────┬──────────┘
           │
┌──────────▼──────────┐
│  DataStore[T] API   │ ◄── Generic interface
└──────────┬──────────┘
           │
┌──────────▼──────────┐
│ DynamoDB DataStore  │ ◄── Concrete implementation
└──────────┬──────────┘
           │
┌──────────▼──────────┐
│   AWS DynamoDB      │
└─────────────────────┘
```

### Component Responsibilities

#### 1. Storage Manager (`entitystore.go`)
- **Purpose**: Manages multiple DataStore instances
- **Design**: Non-generic to allow storing different typed datastores
- **Key Methods**:
  - `RegisterDataStore(key string, store DataStore[any])`
  - `GetDataStore(key string) DataStore[any]`

#### 2. DataStore Interface (`datastore/datastore.go`)
- **Purpose**: Generic CRUD interface for type-safe operations
- **Design**: Uses Go generics for compile-time type safety
- **Operations**:
  ```go
  type DataStore[T any] interface {
      GetOne(ctx context.Context, key string) (*T, error)
      Put(ctx context.Context, entity T) error
      UpdateWithCondition(ctx context.Context, entity T, condition string) error
      Query(ctx context.Context, params QueryParameters) ([]T, error)
      Stream(ctx context.Context, params QueryParameters) (<-chan StreamResult[T], error)
      Delete(ctx context.Context, key string) error
  }
  ```

#### 3. DynamoDB Implementation (`datastore/ddb/`)
- **Purpose**: Concrete DynamoDB implementation of DataStore
- **Features**:
  - Macro expansion for dynamic key generation
  - Support for complex query patterns
  - Streaming for large datasets
  - Conditional updates

#### 4. Registry System
- **Type Registry** (`registry/types.go`):
  - Maps entity type names to unmarshal functions
  - Enables polymorphic storage in single table
  - Thread-safe registration

- **Index Map Registry** (`registry/indexmap.go`):
  - Maps Go types to DynamoDB key configurations
  - Supports primary keys, sort keys, and GSI keys
  - Macro-based field extraction

#### 5. Code Generator (`processor/postprocess.go`)
- **Purpose**: Generate registration code from OpenAPI specs
- **Input**: OpenAPI YAML with vendor extensions
- **Output**: Go code for automatic type registration

## Key Design Patterns

### 1. Single Table Design Support

EntityStore is optimized for DynamoDB single-table design:

```
┌─────────────────────────────────────────────┐
│              DynamoDB Table                 │
├─────────┬──────────┬───────────┬───────────┤
│   PK    │    SK    │EntityType │   Data    │
├─────────┼──────────┼───────────┼───────────┤
│USER#123 │ USER#123 │   User    │ {...}     │
│USER#123 │ ORDER#1  │   Order   │ {...}     │
│PROD#ABC │ PROD#ABC │  Product  │ {...}     │
└─────────┴──────────┴───────────┴───────────┘
```

### 2. Macro-Based Key Mapping

Keys are defined using macros that extract values from entities:

```go
indexMap := map[string]interface{}{
    "PK": "USER#{ID}",           // Extracts ID field
    "SK": "PROFILE#{ProfileID}", // Extracts ProfileID field
    "GSI1PK": "{Email}",        // Extracts Email field
}
```

### 3. Type Registration Pattern

```go
// Register unmarshal function
RegisterType("User", func() interface{} {
    return &User{}
})

// Register index mapping
RegisterIndexMap(reflect.TypeOf(User{}), indexMap)
```

### 4. Enhanced Streaming Pattern

The improved streaming API provides efficient, configurable processing of large datasets:

```go
// Configure streaming with options
resultChan := datastore.Stream(ctx, queryParams,
    storagemodels.WithBufferSize(100),      // Buffered channel
    storagemodels.WithPageSize(25),         // DynamoDB page size
    storagemodels.WithMaxRetries(3),        // Retry transient errors
    storagemodels.WithProgressHandler(func(p StreamProgress) {
        log.Printf("Processed %d items", p.ItemsProcessed)
    }),
)

// Process results with metadata
for result := range resultChan {
    if result.Error != nil {
        log.Printf("Error at item %d: %v", result.Meta.Index, result.Error)
        continue
    }
    
    // Access typed item and metadata
    item := result.Item  // Type T
    meta := result.Meta  // Index, PageNumber, Timestamp
}
```

Key improvements:
- Single channel design for simpler API
- Configurable buffering and concurrency
- Built-in retry logic for transient errors
- Progress tracking and error handling callbacks
- Item-level metadata for debugging

## Data Flow

### Write Operation
1. Application calls `Put()` with typed entity
2. DataStore extracts field values using reflection
3. Macro expansion generates DynamoDB keys
4. EntityType attribute is added automatically
5. Item is written to DynamoDB

### Read Operation
1. Application calls `GetOne()` or `Query()`
2. Keys are constructed using macros
3. DynamoDB query is executed
4. EntityType attribute determines unmarshal function
5. Typed entity is returned to application

## Security Considerations

1. **AWS Credentials**: Never hardcode credentials
2. **Input Validation**: Validate entity data before storage
3. **Access Control**: Use IAM policies for DynamoDB access
4. **Encryption**: Enable DynamoDB encryption at rest

## Performance Optimization

1. **Batch Operations**: Use batch write/read for multiple items
2. **Streaming**: Process large datasets without loading all into memory
3. **Index Design**: Create appropriate GSIs for query patterns
4. **Connection Pooling**: AWS SDK handles connection pooling

## Extensibility

### Adding New Storage Backends
1. Implement the `DataStore[T]` interface
2. Handle type registration appropriately
3. Map operations to backend-specific calls

### Custom Query Patterns
1. Extend `QueryParameters` for new query types
2. Implement query logic in backend
3. Maintain type safety throughout

## Error Handling

EntityStore provides a comprehensive error handling system with semantic error types:

### Error Types
```go
// Common errors
errors.ErrNotFound         // Entity not found
errors.ErrAlreadyExists    // Entity already exists  
errors.ErrValidationError  // Input validation failed
errors.ErrConditionFailed  // Conditional operation failed
errors.ErrNoIndexMap       // Missing index map for type
```

### Error Checking
```go
// Using helper functions
if errors.IsNotFound(err) { /* ... */ }
if errors.IsValidationError(err) { /* ... */ }

// Using errors.Is
if errors.Is(err, errors.ErrNotFound) { /* ... */ }
```

### Features
- Semantic error types for common scenarios
- Wrapped errors maintain context
- Helper functions for error type checking
- Per-item error handling in streaming
- Context cancellation is respected