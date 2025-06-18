# EntityStore Design Documentation

## Overview

EntityStore is a sophisticated generic data storage abstraction layer for Go, designed primarily for AWS DynamoDB. It provides type-safe operations while maintaining flexibility for multi-entity storage patterns.

## Architecture

### Core Components

#### 1. DataStore Interface
The generic interface that provides type-safe CRUD operations:

```go
type DataStore[T any] interface {
    GetOne(ctx context.Context, key string) (*T, error)
    Put(ctx context.Context, entity T) error
    UpdateWithCondition(ctx context.Context, keyInput any, updates map[string]interface{}, condition string) error
    Query(ctx context.Context, params *storagemodels.QueryParams) ([]interface{}, error)
    Stream(ctx context.Context, params *storagemodels.StreamQueryParams) (<-chan storagemodels.StreamItem, <-chan error)
    Delete(ctx context.Context, key string) error
}
```

#### 2. Storage Interface
Higher-level interface for managing collections of DataStore instances:

```go
type Storage interface {
    RegisterDataStore(key string, ds any) error
    GetDataStore(key string) (any, error)
}
```

#### 3. Type Registry System
- Maps entity type prefixes (e.g., "PL", "DR", "RatingSystem") to unmarshal functions
- Enables polymorphic storage where different entity types coexist in the same DynamoDB table
- Registration happens at initialization time

```go
// Example registration
registry.RegisterType("RatingSystem", func(item map[string]types.AttributeValue) (interface{}, error) {
    obj := &RatingSystem{}
    err := attributevalue.UnmarshalMap(item, obj)
    return obj, err
})
```

#### 4. Index Map Registry
- Associates Go types with their DynamoDB key structures
- Supports macro expansion for dynamic key generation
- Thread-safe implementation using sync.RWMutex

```go
// Example index map registration
registry.RegisterIndexMap[RatingSystem](map[string]string{
    "PK": "{ID}",  // {ID} will be replaced with actual ID value
    "SK": "{ID}",
})
```

### DynamoDB Implementation

The primary implementation (`DynamodbDataStore`) provides:

- **Full CRUD Operations**: Create, Read, Update, Delete with type safety
- **Conditional Updates**: Update items only when specific conditions are met
- **Query Support**: Query with filters, pagination, and index support
- **Streaming**: Efficient handling of large result sets
- **Index Management**: Support for Global Secondary Indexes (GSI) and Local Secondary Indexes (LSI)
- **Macro Expansion**: Dynamic key generation using placeholders

#### Key Features:

1. **Macro System**
   - Placeholders like `{ID}`, `{OrganizationID}` in index maps
   - Automatically expanded during operations
   - Supports complex key patterns

2. **Entity Type Tracking**
   - Automatically adds `EntityType` field to stored items
   - Used for polymorphic retrieval and type identification
   - Removed during retrieval to maintain clean data models

3. **Flexible Key Handling**
   - Supports both composite keys (PK + SK) and single object keys
   - Automatic detection of key patterns
   - String key expansion for simple lookups

## Code Generation

### OpenAPI Integration

EntityStore includes a powerful code generation tool that:

1. **Reads OpenAPI Specifications**
   - Looks for `x-dynamodb-indexmap` vendor extensions
   - Extracts model definitions and their DynamoDB configurations

2. **Generates Registration Code**
   - Creates type registry entries
   - Creates index map registrations
   - Outputs clean, maintainable Go code

### Usage Example:

```yaml
# In OpenAPI spec
definitions:
  RatingSystem:
    type: object
    x-dynamodb-indexmap:
      PK: "{ID}"
      SK: "{ID}"
    properties:
      ID:
        type: string
      Name:
        type: string
```

Run the generator:
```bash
./indexmap-pps -in openapi.yaml -outputdata indexmap_type_registration.go
```

## Usage Patterns

### Basic Usage

```go
// 1. Create a DataStore instance
store, err := NewDynamodbDataStore[RatingSystem](
    awsAccessKey, 
    awsSecretKey, 
    awsRegion, 
    tableName
)

// 2. Create and store an entity
rating := &RatingSystem{
    ID:          aws.String("TTOakville"),
    Name:        aws.String("Oakville Table Tennis Ranking System"),
    Description: aws.String("Rating system for table tennis players"),
    CreatedAt:   &now,
    UpdatedAt:   &now,
}
err = store.Put(ctx, *rating)

// 3. Retrieve by key
retrieved, err := store.GetOne(ctx, "TTOakville")

// 4. Update with condition
updates := map[string]interface{}{
    "Name": "Updated Oakville TT System",
}
condition := "attribute_exists(ID)"
err = store.UpdateWithCondition(ctx, rating, updates, condition)

// 5. Delete
err = store.Delete(ctx, "TTOakville")
```

### Advanced Patterns

#### Querying with Filters
```go
params := &storagemodels.QueryParams{
    TableName:                 tableName,
    KeyConditionExpression:    "PK = :pk",
    FilterExpression:          aws.String("Rating > :minRating"),
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":pk":        &types.AttributeValueMemberS{Value: "CLUB#Oakville"},
        ":minRating": &types.AttributeValueMemberN{Value: "1500"},
    },
}
results, err := store.Query(ctx, params)
```

#### Streaming Large Result Sets
```go
streamParams := &storagemodels.StreamQueryParams{
    TableName:              tableName,
    KeyConditionExpression: "PK = :pk",
    // ... other parameters
}
itemChan, errChan := store.Stream(ctx, streamParams)

for {
    select {
    case item := <-itemChan:
        if item.Item == nil {
            return // End of stream
        }
        // Process item
    case err := <-errChan:
        // Handle error
    }
}
```

## Design Philosophy

### 1. Type Safety First
- Leverages Go generics for compile-time type checking
- Reduces runtime errors and improves developer experience

### 2. Flexibility
- Supports multiple entity types in a single table
- Configurable key patterns via index maps
- Extensible through registry patterns

### 3. OpenAPI-Driven
- Models defined in OpenAPI specifications
- Single source of truth for API and storage
- Code generation reduces boilerplate

### 4. Cloud-Native
- Designed for DynamoDB's capabilities
- Supports advanced features like GSIs and conditional updates
- Efficient for serverless architectures

## Integration Status

### Current State
- Exists as a separate module under the Suparena organization
- Complete implementation with tests
- Not yet integrated into main imatch321 services

### Future Integration
- Designed as a foundational component for the iMatch ecosystem
- Will provide consistent data access patterns across services
- Enables sharing of data models between services

## Benefits

1. **Reduced Boilerplate**: Generic interfaces eliminate repetitive code
2. **Type Safety**: Compile-time checking prevents runtime errors
3. **Consistency**: Uniform data access patterns across the codebase
4. **Flexibility**: Easy to add new entity types without changing core logic
5. **Performance**: Optimized for DynamoDB's strengths
6. **Maintainability**: Clear separation of concerns and clean abstractions

## Technical Decisions

### Why DynamoDB?
- Scalability for multi-tenant SaaS applications
- Flexible schema for evolving data models
- Strong consistency options
- Cost-effective for variable workloads

### Why Generics?
- Type safety without code generation for each model
- Clean API that's intuitive to use
- Performance benefits over interface{} approaches

### Why OpenAPI Integration?
- Single source of truth for API contracts
- Automatic synchronization between API and storage layers
- Reduced manual coordination between teams

## Testing

The library includes comprehensive tests demonstrating:
- Basic CRUD operations
- Error handling
- Macro expansion
- Type registry functionality
- Integration with real DynamoDB tables

See `datastore/ddb/dynamodb_test.go` for examples.

## Conclusion

EntityStore represents a modern approach to data storage abstraction in Go, combining type safety, flexibility, and cloud-native design. Its integration with OpenAPI specifications and support for code generation make it a powerful foundation for building scalable applications.