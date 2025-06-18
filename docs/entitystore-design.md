# EntityStore Design Pattern Documentation

## Table of Contents
- [Overview](#overview)
- [Architecture](#architecture)
- [Core Concepts](#core-concepts)
- [Implementation Guide](#implementation-guide)
- [API Reference](#api-reference)
- [Design Patterns](#design-patterns)
- [Best Practices](#best-practices)
- [Advanced Features](#advanced-features)
- [Enhancement Roadmap](#enhancement-roadmap)

## Overview

EntityStore is a sophisticated storage abstraction layer that provides type-safe, annotation-driven data persistence for the Suparena API. It bridges the gap between OpenAPI specifications and DynamoDB storage through a repository pattern with automatic index management.

### Key Benefits

- **Type Safety**: Compile-time type checking with Go generics
- **Single Table Design**: Elegant support for DynamoDB best practices
- **Code Generation**: Automatic creation of index mappings from OpenAPI specs
- **Clean Architecture**: Complete separation of business logic from storage concerns
- **Performance**: Built-in caching with Redis integration
- **Flexibility**: Pluggable storage backends

## Architecture

### High-Level Architecture

```
┌─────────────────────┐
│   OpenAPI Spec      │ ← Define entities with x-dynamodb-indexmap
└──────────┬──────────┘
           │ generate-server.sh
           ▼
┌─────────────────────┐
│  Generated Models   │ → Type definitions & index mappings
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐     ┌───────────────────┐
│  Storage Manager    │────▶│  Type Registry    │
└──────────┬──────────┘     └───────────────────┘
           │
           ▼
┌─────────────────────┐     ┌───────────────────┐
│  DataStore[T]       │────▶│  Index Registry   │
└──────────┬──────────┘     └───────────────────┘
           │
    ┌──────┴──────┐
    ▼             ▼
┌────────┐   ┌─────────┐
│DynamoDB│   │  Redis  │
└────────┘   └─────────┘
```

### Component Responsibilities

1. **Storage Manager**: Central registry and factory for all datastores
2. **DataStore Interface**: Type-safe CRUD operations for specific entities
3. **Index Registry**: Maps entity fields to DynamoDB partition/sort keys
4. **Type Registry**: Handles entity serialization and type resolution
5. **Code Generator**: Creates index mappings from OpenAPI annotations

## Core Concepts

### 1. Index Mapping Annotations

The heart of EntityStore is the `x-dynamodb-indexmap` vendor extension in OpenAPI:

```yaml
UserProfile:
  type: object
  x-dynamodb-indexmap:
    PK: "USER#{UserId}"    # Partition key pattern
    SK: "PROFILE"          # Sort key (static value)
  properties:
    UserId:
      type: string
    Name:
      type: string
```

#### Pattern Syntax
- `{FieldName}` - Direct field value extraction
- `PREFIX#{FieldName}` - Composite key with prefix
- `"STATIC_VALUE"` - Literal string value
- `{Field1}#{Field2}` - Multiple field composition

### 2. Single Table Design

EntityStore elegantly supports DynamoDB single-table design patterns:

```yaml
# Different entity types in the same table
Competition:
  x-dynamodb-indexmap:
    PK: "{CompetitionID}"
    SK: "{ID}"

Event:
  x-dynamodb-indexmap:
    PK: "COMP#{CompID}"
    SK: "EVENT#{EventID}"

Match:
  x-dynamodb-indexmap:
    PK: "COMP#{CompID}"
    SK: "MATCH#{MatchID}"

UserProfile:
  x-dynamodb-indexmap:
    PK: "USER#{UserId}"
    SK: "PROFILE"
```

### 3. Type Registration

Entities are registered based on their source:

```go
// OpenAPI-generated models (automatic)
"NewsletterSignupRecord" → models.NewsletterSignupRecord

// Core package types (manual)
"core.UserProfile" → core.UserProfile
"core.Competition" → core.Competition
```

## Implementation Guide

### Step 1: Define Entity in OpenAPI

```yaml
# openapi.yaml
definitions:
  UserProfile:
    type: object
    x-dynamodb-indexmap:
      PK: "USER#{UserId}"
      SK: "PROFILE"
    properties:
      UserId:
        type: string
        description: Unique user identifier
      Name:
        type: string
        description: User's display name
      Email:
        type: string
        format: email
      CreatedAt:
        type: string
        format: date-time
    required:
      - UserId
      - Name
```

### Step 2: Generate Code

```bash
cd openapi
./generate-server.sh
```

This generates:
- Model structs in `models/`
- Index mappings in `models/indexmap_type_registration.go`
- API operations in `restapi/operations/`

### Step 3: Register Datastore

```go
// configure_suparena_backend.go
func init() {
    // Create storage manager
    storage := entitystore.NewStorageManager()
    
    // Create typed datastore
    userProfileStore, err := ddb.NewDynamodbDataStore[core.UserProfile](
        awsAccessKey,
        awsSecretKey,
        region,
        tableName,
    )
    if err != nil {
        log.Fatalf("Failed to create user profile datastore: %v", err)
    }
    
    // Register with storage manager
    storage.RegisterDataStore("core.UserProfile", userProfileStore)
}
```

### Step 4: Use in Service Layer

```go
// userprofile/userprofile_service.go
type Service struct {
    store entitystore.Storage
    logger logger.LoggerService
}

func (s *Service) CreateProfile(ctx context.Context, profile *core.UserProfile) error {
    // Get typed datastore
    ds, err := s.store.GetDataStore("core.UserProfile")
    if err != nil {
        return fmt.Errorf("failed to get datastore: %w", err)
    }
    
    // Type assertion to specific datastore
    userStore := ds.(datastore.DataStore[core.UserProfile])
    
    // Perform operation
    profile.CreatedAt = time.Now()
    profile.UpdatedAt = time.Now()
    
    return userStore.Put(ctx, profile)
}

func (s *Service) GetProfile(ctx context.Context, userId string) (*core.UserProfile, error) {
    ds, _ := s.store.GetDataStore("core.UserProfile")
    userStore := ds.(datastore.DataStore[core.UserProfile])
    
    return userStore.GetOne(ctx, userId)
}
```

## API Reference

### Storage Manager

```go
type Storage interface {
    // Register a datastore for an entity type
    RegisterDataStore(entityType string, dataStore interface{})
    
    // Retrieve a registered datastore
    GetDataStore(entityType string) (interface{}, error)
    
    // List all registered entity types
    GetRegisteredTypes() []string
}
```

### DataStore Interface

```go
type DataStore[T any] interface {
    // Create or update an entity
    Put(ctx context.Context, entity T) error
    
    // Retrieve a single entity by key
    GetOne(ctx context.Context, key interface{}) (T, error)
    
    // Delete an entity
    Delete(ctx context.Context, key interface{}) error
    
    // Query entities
    Query(ctx context.Context, params *QueryParams) ([]T, error)
    
    // Batch operations
    BatchGet(ctx context.Context, keys []interface{}) ([]T, error)
    BatchPut(ctx context.Context, entities []T) error
    BatchDelete(ctx context.Context, keys []interface{}) error
}
```

### Query Parameters

```go
type QueryParams struct {
    // DynamoDB key condition expression
    KeyCondition string
    
    // Filter expression (applied after query)
    FilterExpression string
    
    // Attribute values for expressions
    ExpressionAttributeValues map[string]types.AttributeValue
    
    // Attribute name substitutions
    ExpressionAttributeNames map[string]string
    
    // Pagination
    Limit int32
    ExclusiveStartKey map[string]types.AttributeValue
    
    // Index name for GSI queries
    IndexName string
    
    // Sort order
    ScanIndexForward *bool
}
```

## Design Patterns

### 1. Repository Pattern

Each service encapsulates storage operations:

```go
type UserRepository interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id string) (*User, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id string) error
    FindByEmail(ctx context.Context, email string) (*User, error)
}

type userRepository struct {
    store entitystore.Storage
}

func (r *userRepository) Create(ctx context.Context, user *User) error {
    ds, _ := r.store.GetDataStore("core.User")
    return ds.(datastore.DataStore[User]).Put(ctx, user)
}
```

### 2. Unit of Work Pattern

Batch operations for consistency:

```go
func (s *Service) TransferOwnership(ctx context.Context, compID, newOwnerID string) error {
    // Get datastores
    compDS, _ := s.store.GetDataStore("core.Competition")
    eventDS, _ := s.store.GetDataStore("core.Event")
    
    // Fetch competition
    comp, err := compDS.(datastore.DataStore[Competition]).GetOne(ctx, compID)
    if err != nil {
        return err
    }
    
    // Update owner
    comp.OwnerID = newOwnerID
    
    // Fetch and update all events
    events, _ := s.getCompetitionEvents(ctx, compID)
    for i := range events {
        events[i].OwnerID = newOwnerID
    }
    
    // Batch update
    compDS.(datastore.DataStore[Competition]).Put(ctx, comp)
    eventDS.(datastore.DataStore[Event]).BatchPut(ctx, events)
    
    return nil
}
```

### 3. Cache-Aside Pattern

Integrate caching transparently:

```go
func (s *Service) GetProfileWithCache(ctx context.Context, userId string) (*UserProfile, error) {
    // Check cache
    cacheKey := fmt.Sprintf("user:%s", userId)
    if cached, err := s.redis.Get(ctx, cacheKey).Result(); err == nil {
        var profile UserProfile
        if err := json.Unmarshal([]byte(cached), &profile); err == nil {
            return &profile, nil
        }
    }
    
    // Load from datastore
    ds, _ := s.store.GetDataStore("core.UserProfile")
    profile, err := ds.(datastore.DataStore[UserProfile]).GetOne(ctx, userId)
    if err != nil {
        return nil, err
    }
    
    // Update cache
    if data, err := json.Marshal(profile); err == nil {
        s.redis.Set(ctx, cacheKey, data, 5*time.Minute)
    }
    
    return &profile, nil
}
```

## Best Practices

### 1. Entity Design

```yaml
# Good: Clear prefix for entity type
UserProfile:
  x-dynamodb-indexmap:
    PK: "USER#{UserId}"
    SK: "PROFILE"

# Good: Hierarchical keys for related entities
Order:
  x-dynamodb-indexmap:
    PK: "USER#{UserId}"
    SK: "ORDER#{OrderId}"
```

### 2. Error Handling

```go
// Always check type assertions
ds, err := s.store.GetDataStore("core.UserProfile")
if err != nil {
    return nil, fmt.Errorf("datastore not found: %w", err)
}

userStore, ok := ds.(datastore.DataStore[core.UserProfile])
if !ok {
    return nil, fmt.Errorf("invalid datastore type")
}
```

### 3. Query Optimization

```go
// Use GSI for alternate access patterns
params := &QueryParams{
    IndexName: "email-index",
    KeyCondition: "Email = :email",
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":email": &types.AttributeValueMemberS{Value: email},
    },
    Limit: 1,
}
```

### 4. Batch Operations

```go
// Process in chunks to respect DynamoDB limits
func batchProcess(items []Item) error {
    const chunkSize = 25 // DynamoDB batch limit
    
    for i := 0; i < len(items); i += chunkSize {
        end := i + chunkSize
        if end > len(items) {
            end = len(items)
        }
        
        chunk := items[i:end]
        if err := dataStore.BatchPut(ctx, chunk); err != nil {
            return fmt.Errorf("batch failed at chunk %d: %w", i/chunkSize, err)
        }
    }
    return nil
}
```

## Advanced Features

### 1. Custom Index Patterns

For complex key generation:

```go
// core/indexmap_type_registration.go
func init() {
    registry.RegisterIndexMap[MatchResult](func() map[string]string {
        return map[string]string{
            "PK": "MATCH#{MatchId}",
            "SK": "RESULT#{Timestamp}",
            "GSI1PK": "PLAYER#{PlayerId}",
            "GSI1SK": "MATCH#{MatchId}",
        }
    })
}
```

### 2. Conditional Operations

```go
// Update only if version matches (optimistic locking)
params := &UpdateParams{
    ConditionExpression: "Version = :currentVersion",
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":currentVersion": &types.AttributeValueMemberN{Value: "1"},
        ":newVersion": &types.AttributeValueMemberN{Value: "2"},
    },
}
```

### 3. Stream Processing

```go
// Process DynamoDB streams
func processStream(record *dynamodbstreams.Record) {
    entityType := extractEntityType(record.Keys)
    
    ds, _ := storage.GetDataStore(entityType)
    // Handle change event
}
```

## Enhancement Roadmap

### 1. Additional Storage Backends

```go
// PostgreSQL implementation
type PostgresDataStore[T any] struct {
    db *sql.DB
    tableName string
}

func (s *PostgresDataStore[T]) Put(ctx context.Context, entity T) error {
    // SQL implementation
}

// MongoDB implementation
type MongoDataStore[T any] struct {
    collection *mongo.Collection
}

func (s *MongoDataStore[T]) Put(ctx context.Context, entity T) error {
    // MongoDB implementation
}
```

### 2. Query Builder API

```go
// Fluent query interface
type QueryBuilder[T any] struct {
    store DataStore[T]
    conditions []Condition
}

func (qb *QueryBuilder[T]) Where(field string) *ConditionBuilder[T] {
    return &ConditionBuilder[T]{qb: qb, field: field}
}

// Usage
users, err := userStore.Query().
    Where("Location.City").Equals("Toronto").
    And("Age").GreaterThan(18).
    OrderBy("CreatedAt").Desc().
    Limit(10).
    Execute(ctx)
```

### 3. Automatic Caching Layer

```go
type CachedDataStore[T any] struct {
    store DataStore[T]
    cache Cache
    ttl   time.Duration
}

func WithCache[T any](store DataStore[T], cache Cache, ttl time.Duration) DataStore[T] {
    return &CachedDataStore[T]{
        store: store,
        cache: cache,
        ttl:   ttl,
    }
}
```

### 4. Transaction Support

```go
type Transaction struct {
    operations []Operation
    storage    Storage
}

func (s *Storage) BeginTransaction() *Transaction {
    return &Transaction{storage: s}
}

func (tx *Transaction) Put(entityType string, entity interface{}) *Transaction {
    tx.operations = append(tx.operations, Operation{
        Type:   "PUT",
        Entity: entity,
        EntityType: entityType,
    })
    return tx
}

func (tx *Transaction) Commit(ctx context.Context) error {
    // Execute all operations atomically
}
```

### 5. Migration Framework

```go
type Migrator struct {
    storage    Storage
    migrations []Migration
}

type Migration struct {
    Version string
    Up      func(item map[string]interface{}) error
    Down    func(item map[string]interface{}) error
}

func (m *Migrator) Run(ctx context.Context) error {
    // Apply migrations in order
}
```

### 6. Observability

```go
// Metrics middleware
type MetricsDataStore[T any] struct {
    store   DataStore[T]
    metrics MetricsCollector
}

func (m *MetricsDataStore[T]) Put(ctx context.Context, entity T) error {
    start := time.Now()
    err := m.store.Put(ctx, entity)
    
    m.metrics.RecordLatency("put", time.Since(start))
    if err != nil {
        m.metrics.IncrementErrors("put")
    }
    return err
}
```

### 7. Schema Validation

```go
// Runtime validation against OpenAPI schema
type ValidatingDataStore[T any] struct {
    store     DataStore[T]
    validator SchemaValidator
}

func (v *ValidatingDataStore[T]) Put(ctx context.Context, entity T) error {
    if err := v.validator.Validate(entity); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    return v.store.Put(ctx, entity)
}
```

## Testing

### Unit Testing with Mocks

```go
// Mock datastore for testing
type MockDataStore[T any] struct {
    PutFunc    func(ctx context.Context, entity T) error
    GetOneFunc func(ctx context.Context, key interface{}) (T, error)
}

func (m *MockDataStore[T]) Put(ctx context.Context, entity T) error {
    return m.PutFunc(ctx, entity)
}

// Test example
func TestUserService_CreateProfile(t *testing.T) {
    mockStore := &MockDataStore[UserProfile]{
        PutFunc: func(ctx context.Context, entity UserProfile) error {
            assert.Equal(t, "testuser", entity.UserId)
            return nil
        },
    }
    
    storage := entitystore.NewStorageManager()
    storage.RegisterDataStore("core.UserProfile", mockStore)
    
    service := NewUserService(storage)
    err := service.CreateProfile(ctx, &UserProfile{UserId: "testuser"})
    assert.NoError(t, err)
}
```

### Integration Testing

```go
// Test with local DynamoDB
func TestIntegration_UserProfile(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Setup local DynamoDB
    store := setupLocalDynamoDB(t)
    
    // Test full CRUD cycle
    profile := &UserProfile{UserId: "test123"}
    
    // Create
    err := store.Put(context.Background(), profile)
    require.NoError(t, err)
    
    // Read
    retrieved, err := store.GetOne(context.Background(), "test123")
    require.NoError(t, err)
    assert.Equal(t, profile.UserId, retrieved.UserId)
    
    // Delete
    err = store.Delete(context.Background(), "test123")
    require.NoError(t, err)
}
```

## Troubleshooting

### Common Issues

1. **"datastore with key 'X' not found"**
   - Ensure datastore is registered in `configure_suparena_backend.go`
   - Check entity type name matches exactly

2. **"no index map found for entity type"**
   - Verify `x-dynamodb-indexmap` annotation in OpenAPI spec
   - Run `generate-server.sh` after changes
   - Check `indexmap_type_registration.go` for manual registrations

3. **Type assertion failures**
   - Ensure correct generic type in datastore creation
   - Verify entity type matches registered type

4. **DynamoDB errors**
   - Check AWS credentials and permissions
   - Verify table exists and has correct indexes
   - Ensure key patterns match actual field names

## Conclusion

The EntityStore pattern provides a robust foundation for data persistence in microservices, offering:

- Type safety through Go generics
- Clean separation of concerns
- Efficient DynamoDB integration
- Extensibility for future requirements
- Excellent testability

By following the patterns and practices outlined in this document, developers can build maintainable, performant, and type-safe data access layers for their services.