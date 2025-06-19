# EntityStore Quick Reference Guide

A concise reference for common EntityStore operations in the Suparena API.

## Table of Contents
- [Quick Start](#quick-start)
- [Common Operations](#common-operations)
- [Index Patterns](#index-patterns)
- [Query Examples](#query-examples)
- [Error Handling](#error-handling)
- [Testing](#testing)
- [Cheat Sheet](#cheat-sheet)

## Quick Start

### 1. Define Entity in OpenAPI

```yaml
# openapi/openapi.yaml
MyEntity:
  type: object
  x-dynamodb-indexmap:
    PK: "ENTITY#{EntityId}"
    SK: "METADATA"
  properties:
    EntityId: {type: string}
    Name: {type: string}
```

### 2. Generate Code

```bash
cd openapi && ./generate-server.sh
```

### 3. Register Datastore

```go
// In configure_suparena_backend.go
entityStore, _ := ddb.NewDynamodbDataStore[models.MyEntity](...)
storage.RegisterDataStore("models.MyEntity", entityStore)
```

### 4. Use in Service

```go
ds, _ := storage.GetDataStore("models.MyEntity")
store := ds.(datastore.DataStore[models.MyEntity])
```

## Common Operations

### Create/Update (Put)

```go
entity := &models.MyEntity{
    EntityId: "123",
    Name: "Test Entity",
}
err := store.Put(ctx, entity)
```

### Read (GetOne)

```go
entity, err := store.GetOne(ctx, "123")
if err != nil {
    // Handle not found
}
```

### Delete

```go
err := store.Delete(ctx, "123")
```

### Query

```go
// Query by partition key
params := &QueryParams{
    KeyCondition: "PK = :pk",
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":pk": &types.AttributeValueMemberS{Value: "ENTITY#123"},
    },
}
results, err := store.Query(ctx, params)
```

### Streaming

```go
// Basic streaming
params := &storagemodels.QueryParams{
    TableName: "my-table",
    KeyConditionExpression: "PK = :pk",
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":pk": &types.AttributeValueMemberS{Value: "USER#123"},
    },
}

// Stream with default options
for result := range store.Stream(ctx, params) {
    if result.Error != nil {
        log.Printf("Error: %v", result.Error)
        continue
    }
    process(result.Item)
}

// Stream with options
for result := range store.Stream(ctx, params,
    storagemodels.WithBufferSize(100),
    storagemodels.WithPageSize(25),
    storagemodels.WithProgressHandler(func(p StreamProgress) {
        log.Printf("Progress: %d items", p.ItemsProcessed)
    }),
) {
    // Process results
}
```

### Batch Operations

```go
// Batch get
keys := []interface{}{"123", "456", "789"}
entities, err := store.BatchGet(ctx, keys)

// Batch put
entities := []models.MyEntity{entity1, entity2, entity3}
err := store.BatchPut(ctx, entities)

// Batch delete
err := store.BatchDelete(ctx, keys)
```

## Index Patterns

### Basic Patterns

```yaml
# Simple primary key
PK: "{UserId}"
SK: "{UserId}"

# Composite key with prefix
PK: "USER#{UserId}"
SK: "PROFILE"

# Hierarchical data
PK: "ORG#{OrgId}"
SK: "USER#{UserId}"

# Time-series data
PK: "DEVICE#{DeviceId}"
SK: "DATA#{Timestamp}"
```

### Common Entity Patterns

```yaml
# User profiles
UserProfile:
  x-dynamodb-indexmap:
    PK: "USER#{UserId}"
    SK: "PROFILE"

# User's orders
Order:
  x-dynamodb-indexmap:
    PK: "USER#{UserId}"
    SK: "ORDER#{OrderId}"

# Organization hierarchy
Team:
  x-dynamodb-indexmap:
    PK: "ORG#{OrgId}"
    SK: "TEAM#{TeamId}"

# Many-to-many relationships
UserTeam:
  x-dynamodb-indexmap:
    PK: "USER#{UserId}"
    SK: "TEAM#{TeamId}"
    GSI1PK: "TEAM#{TeamId}"
    GSI1SK: "USER#{UserId}"
```

## Query Examples

### Get All Items for a User

```go
params := &QueryParams{
    KeyCondition: "PK = :pk",
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":pk": &types.AttributeValueMemberS{Value: "USER#123"},
    },
}
items, _ := store.Query(ctx, params)
```

### Query with Filter

```go
params := &QueryParams{
    KeyCondition: "PK = :pk",
    FilterExpression: "CreatedAt > :date",
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":pk": &types.AttributeValueMemberS{Value: "USER#123"},
        ":date": &types.AttributeValueMemberS{Value: "2024-01-01"},
    },
}
```

### Query with Prefix

```go
params := &QueryParams{
    KeyCondition: "PK = :pk AND begins_with(SK, :prefix)",
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":pk": &types.AttributeValueMemberS{Value: "USER#123"},
        ":prefix": &types.AttributeValueMemberS{Value: "ORDER#"},
    },
}
```

### Query with GSI

```go
// Basic GSI query
params := &QueryParams{
    IndexName: "GSI1",
    KeyCondition: "GSI1PK = :pk",
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":pk": &types.AttributeValueMemberS{Value: "TEAM#456"},
    },
}

// Using GSI Query Builder (New!)
results, err := store.QueryByGSI1PK(ctx, "user@example.com")

// GSI query with sort key prefix
results, err := store.QueryByGSI1PKAndSKPrefix(ctx, "user@example.com", "STATUS#active")

// Complex GSI query with builder
results, err := store.QueryGSI().
    WithPartitionKey("user@example.com").
    WithSortKeyPrefix("STATUS#active").
    WithFilter("Country = :country", filterValues).
    WithLimit(100).
    Execute(ctx)

// GSI query with sort key range
results, err := store.QueryGSI().
    WithPartitionKey("user@example.com").
    WithSortKeyBetween("DATE#2024-01-01", "DATE#2024-12-31").
    Execute(ctx)

// Stream GSI query results
resultCh := store.QueryGSI().
    WithPartitionKey("user@example.com").
    Stream(ctx, storagemodels.WithBufferSize(100))

for result := range resultCh {
    if result.Error != nil {
        continue
    }
    process(result.Item)
}

// Time-based queries (New!)
// Get last 24 hours of activity
recent, err := store.QueryByTimeRange("user123").
    InLastHours(24).
    Latest().
    Execute(ctx)

// Query this week's data
weekly, err := store.QueryByTimeRange("metrics").
    ThisWeek().
    WithLimit(100).
    Execute(ctx)

// Stream latest items (newest first)
for result := range store.StreamLatestItems(ctx, "notifications") {
    processNotification(result.Item)
}

// Process in time windows
iterator := store.QueryTimeWindows("logs", start, end, 24*time.Hour)
for {
    dailyLogs, hasMore, err := iterator.Next(ctx)
    if !hasMore { break }
    processDailyBatch(dailyLogs)
}
```

### Pagination

```go
params := &QueryParams{
    KeyCondition: "PK = :pk",
    Limit: 10,
    ExclusiveStartKey: lastEvaluatedKey, // From previous response
}
```

## Error Handling

### Common Errors and Solutions

```go
// Datastore not found
ds, err := storage.GetDataStore("core.UserProfile")
if err != nil {
    return fmt.Errorf("datastore not registered: %w", err)
}

// Type assertion safety
userStore, ok := ds.(datastore.DataStore[core.UserProfile])
if !ok {
    return fmt.Errorf("invalid datastore type")
}

// Not found handling
profile, err := userStore.GetOne(ctx, userId)
if err != nil {
    if strings.Contains(err.Error(), "not found") {
        return ErrUserNotFound
    }
    return fmt.Errorf("failed to get profile: %w", err)
}

// Conditional update failure
err = userStore.Put(ctx, entity)
if err != nil {
    var ccf *types.ConditionalCheckFailedException
    if errors.As(err, &ccf) {
        return ErrConcurrentModification
    }
    return err
}
```

## Testing

### Mock Datastore

```go
type MockDataStore[T any] struct {
    entities map[string]T
}

func (m *MockDataStore[T]) Put(ctx context.Context, entity T) error {
    // Extract key from entity and store
    return nil
}

func (m *MockDataStore[T]) GetOne(ctx context.Context, key interface{}) (T, error) {
    var zero T
    entity, ok := m.entities[key.(string)]
    if !ok {
        return zero, errors.New("not found")
    }
    return entity, nil
}
```

### Test Setup

```go
func setupTestStorage() entitystore.Storage {
    storage := entitystore.NewStorageManager()
    
    // Register mock datastores
    userMock := &MockDataStore[core.UserProfile]{
        entities: make(map[string]core.UserProfile),
    }
    storage.RegisterDataStore("core.UserProfile", userMock)
    
    return storage
}
```

### Integration Test

```go
func TestUserProfileCRUD(t *testing.T) {
    // Use test containers for DynamoDB Local
    ctx := context.Background()
    container := setupDynamoDBLocal(t)
    
    // Create real datastore
    store, _ := ddb.NewDynamodbDataStore[core.UserProfile](
        "test", "test", "local", "test-table",
    )
    
    // Test operations
    profile := &core.UserProfile{UserId: "test123"}
    
    // Create
    err := store.Put(ctx, profile)
    require.NoError(t, err)
    
    // Read
    retrieved, err := store.GetOne(ctx, "test123")
    require.NoError(t, err)
    assert.Equal(t, profile.UserId, retrieved.UserId)
}
```

## Cheat Sheet

### Entity Definition Template

```yaml
EntityName:
  type: object
  x-dynamodb-indexmap:
    PK: "PREFIX#{PrimaryKey}"
    SK: "TYPE#[{SortKey}|STATIC]"
  properties:
    PrimaryKey: {type: string}
    SortKey: {type: string}
    # ... other fields
```

### Service Method Template

```go
func (s *Service) OperationName(ctx context.Context, param string) error {
    // Get datastore
    ds, err := s.storage.GetDataStore("namespace.EntityName")
    if err != nil {
        return fmt.Errorf("failed to get datastore: %w", err)
    }
    
    // Type assertion
    store, ok := ds.(datastore.DataStore[EntityType])
    if !ok {
        return fmt.Errorf("invalid datastore type")
    }
    
    // Perform operation
    entity := &EntityType{Field: param}
    return store.Put(ctx, entity)
}
```

### Common Imports

```go
import (
    "context"
    "fmt"
    
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
    "github.com/suparena/entitystore"
    "github.com/suparena/entitystore/datastore"
    "github.com/suparena/entitystore/datastore/ddb"
)
```

### Registration in configure_suparena_backend.go

```go
// For OpenAPI models
modelStore, _ := ddb.NewDynamodbDataStore[models.EntityName](...)
storage.RegisterDataStore("models.EntityName", modelStore)

// For core types
coreStore, _ := ddb.NewDynamodbDataStore[core.EntityName](...)
storage.RegisterDataStore("core.EntityName", coreStore)
```

### Quick Debugging

```bash
# Check if entity is registered
grep -r "RegisterDataStore.*EntityName" .

# Check index mapping
grep -r "x-dynamodb-indexmap" -A 3 openapi/

# Find datastore usage
grep -r "GetDataStore.*EntityName" .

# Check generated code
ls openapi/models/*entity_name*
```

### Performance Tips

1. **Batch Operations**: Use for bulk updates (25 items max)
2. **Projections**: Request only needed attributes
3. **Consistent Reads**: Use only when necessary
4. **Caching**: Add Redis layer for frequently accessed data
5. **Indexes**: Design GSIs for alternate access patterns

### Common Gotchas

1. **Entity type names are case-sensitive**
2. **Index patterns must match exact field names**
3. **DynamoDB limits: 400KB item size, 25 batch items**
4. **Type assertions always need checking**
5. **Generated code needs regeneration after OpenAPI changes**

This quick reference should help you work efficiently with EntityStore in the Suparena API!