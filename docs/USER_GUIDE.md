# EntityStore User Guide

## Table of Contents
1. [Getting Started](#getting-started)
2. [Basic Usage](#basic-usage)
3. [Advanced Features](#advanced-features)
4. [Code Generation](#code-generation)
5. [Best Practices](#best-practices)
6. [Troubleshooting](#troubleshooting)

## Getting Started

### Installation

```bash
go get github.com/yourusername/entitystore
```

### Prerequisites

- Go 1.22 or later
- AWS account with DynamoDB access
- AWS credentials configured

### Basic Setup

```go
import (
    "github.com/yourusername/entitystore/datastore/ddb"
    "github.com/yourusername/entitystore/registry"
)
```

## Basic Usage

### 1. Define Your Entity

```go
type User struct {
    ID        string           `json:"id"`
    Email     string           `json:"email"`
    Name      string           `json:"name"`
    CreatedAt *strfmt.DateTime `json:"createdAt"`
    UpdatedAt *strfmt.DateTime `json:"updatedAt"`
}
```

### 2. Register Your Entity

```go
func init() {
    // Register unmarshal function
    registry.RegisterType("User", func() interface{} {
        return &User{}
    })
    
    // Register index mapping
    indexMap := map[string]interface{}{
        "PK": "USER#{ID}",
        "SK": "USER#{ID}",
        "GSI1PK": "{Email}",
        "GSI1SK": "USER",
    }
    registry.RegisterIndexMap(reflect.TypeOf(User{}), indexMap)
}
```

### 3. Create a DataStore

```go
// Using environment credentials
store, err := ddb.NewDynamodbDataStore[User](
    "", // AWS Access Key (empty to use env)
    "", // AWS Secret Key (empty to use env)
    "us-east-1",
    "my-table-name",
)

// Or with explicit credentials
store, err := ddb.NewDynamodbDataStore[User](
    "AKIAIOSFODNN7EXAMPLE",
    "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
    "us-east-1",
    "my-table-name",
)
```

### 4. CRUD Operations

#### Create/Update
```go
user := User{
    ID:    "123",
    Email: "user@example.com",
    Name:  "John Doe",
}

err := store.Put(context.Background(), user)
```

#### Read
```go
// Get by key
user, err := store.GetOne(context.Background(), "123")

// Query
params := storagemodels.QueryParameters{
    KeyConditionExpression: "PK = :pk",
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":pk": &types.AttributeValueMemberS{Value: "USER#123"},
    },
}
users, err := store.Query(context.Background(), params)
```

#### Update with Condition
```go
user.Name = "Jane Doe"
condition := "attribute_exists(PK)" // Only update if item exists

err := store.UpdateWithCondition(context.Background(), user, condition)
```

#### Delete
```go
err := store.Delete(context.Background(), "123")
```

## Advanced Features

### 1. Complex Key Patterns

Support for hierarchical data:

```go
type Order struct {
    UserID  string `json:"userId"`
    OrderID string `json:"orderId"`
    Total   float64 `json:"total"`
}

// Register with composite keys
indexMap := map[string]interface{}{
    "PK": "USER#{UserID}",
    "SK": "ORDER#{OrderID}",
}
```

### 2. Streaming Large Result Sets

The enhanced streaming API provides efficient processing of large datasets with configurable options:

```go
params := storagemodels.QueryParams{
    TableName: "my-table",
    KeyConditionExpression: "PK = :pk",
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":pk": &types.AttributeValueMemberS{Value: "USER#123"},
    },
}

// Basic streaming
resultChan := store.Stream(context.Background(), params)

for result := range resultChan {
    if result.Error != nil {
        log.Printf("Error processing item: %v", result.Error)
        continue
    }
    
    // Access typed item
    user := result.Item
    fmt.Printf("Got user: %+v\n", user)
    
    // Access metadata
    fmt.Printf("Item #%d from page %d\n", result.Meta.Index, result.Meta.PageNumber)
}
```

#### Streaming with Options

```go
// Configure streaming behavior
resultChan := store.Stream(context.Background(), params,
    storagemodels.WithBufferSize(100),        // Channel buffer size
    storagemodels.WithPageSize(25),           // Items per DynamoDB page
    storagemodels.WithMaxRetries(3),          // Retry failed requests
    storagemodels.WithRetryBackoff(time.Second),
    storagemodels.WithProgressHandler(func(progress storagemodels.StreamProgress) {
        log.Printf("Processed %d items at %.2f items/sec", 
            progress.ItemsProcessed, progress.CurrentRate)
    }),
    storagemodels.WithErrorHandler(func(err error) bool {
        log.Printf("Stream error: %v", err)
        return true // Continue on error
    }),
)
```

### 3. Using Secondary Indexes

```go
// Query by email using GSI
params := storagemodels.QueryParameters{
    IndexName: aws.String("GSI1"),
    KeyConditionExpression: "GSI1PK = :email",
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":email": &types.AttributeValueMemberS{Value: "user@example.com"},
    },
}

users, err := store.Query(context.Background(), params)
```

### 4. Filter Expressions

```go
params := storagemodels.QueryParameters{
    KeyConditionExpression: "PK = :pk",
    FilterExpression: aws.String("CreatedAt > :date"),
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":pk": &types.AttributeValueMemberS{Value: "USER#123"},
        ":date": &types.AttributeValueMemberS{Value: "2024-01-01"},
    },
}
```

### 5. Using Storage Manager

```go
// Create a storage manager
manager := entitystore.NewStorageManager()

// Register multiple datastores
userStore, _ := ddb.NewDynamodbDataStore[User](...)
orderStore, _ := ddb.NewDynamodbDataStore[Order](...)

manager.RegisterDataStore("users", userStore)
manager.RegisterDataStore("orders", orderStore)

// Retrieve and use
store := manager.GetDataStore("users")
```

## Code Generation

### 1. OpenAPI Specification

Add vendor extensions to your OpenAPI spec:

```yaml
components:
  schemas:
    User:
      type: object
      x-dynamodb-indexmap:
        PK: "USER#{ID}"
        SK: "USER#{ID}"
        GSI1PK: "{Email}"
        GSI1SK: "USER"
      properties:
        id:
          type: string
        email:
          type: string
        name:
          type: string
```

### 2. Generate Registration Code

```bash
# Build the generator
./build.sh

# Generate code
./indexmap-pps -input api.yaml -output generated/registry.go
```

### 3. Generated Code

The generator creates:
```go
func init() {
    // Type registration
    registry.RegisterType("User", func() interface{} {
        return &User{}
    })
    
    // Index map registration
    registry.RegisterIndexMap(reflect.TypeOf(User{}), map[string]interface{}{
        "PK": "USER#{ID}",
        "SK": "USER#{ID}",
        "GSI1PK": "{Email}",
        "GSI1SK": "USER",
    })
}
```

## Best Practices

### 1. Table Design

- Use single table design for related entities
- Design partition keys for even distribution
- Create GSIs for alternative access patterns

### 2. Error Handling

EntityStore provides semantic error types for better error handling:

```go
import "github.com/suparena/entitystore/errors"

user, err := store.GetOne(ctx, "123")
if err != nil {
    switch {
    case errors.IsNotFound(err):
        // Handle not found
        return nil, fmt.Errorf("user %s does not exist", "123")
    
    case errors.IsValidationError(err):
        // Handle validation error
        return nil, fmt.Errorf("invalid user data: %w", err)
    
    default:
        // Handle other errors
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
}
```

#### Common Error Types

```go
// Check specific error types
if errors.IsNotFound(err) { /* ... */ }
if errors.IsAlreadyExists(err) { /* ... */ }
if errors.IsValidationError(err) { /* ... */ }
if errors.IsConditionFailed(err) { /* ... */ }

// Create typed errors
err := errors.NewNotFoundError("User", "123")
err := errors.NewValidationError("email", "invalid format")
err := errors.NewConditionFailedError("update", "version mismatch")
```

### 3. Context Usage

Always pass context for cancellation support:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

user, err := store.GetOne(ctx, "123")
```

### 4. Batch Operations

For multiple items, consider batching:

```go
// TODO: Implement batch operations in the library
// For now, use concurrent goroutines with rate limiting
```

### 5. Testing

```go
func TestUserOperations(t *testing.T) {
    // Use test table
    store, err := ddb.NewDynamodbDataStore[User](
        os.Getenv("AWS_ACCESS_KEY_ID"),
        os.Getenv("AWS_SECRET_ACCESS_KEY"),
        os.Getenv("AWS_REGION"),
        os.Getenv("DDB_TEST_TABLE_NAME"),
    )
    
    // Test operations...
}
```

## Troubleshooting

### Common Issues

#### 1. "ResourceNotFoundException"
- Ensure table exists in the specified region
- Check table name spelling

#### 2. "ValidationException"
- Verify index map matches table schema
- Check attribute names and types

#### 3. Type Registration Errors
- Ensure types are registered before use
- Check for duplicate registrations

#### 4. Query Returns Empty
- Verify key condition expression
- Check macro expansion results
- Ensure GSI is specified if needed

### Debug Tips

1. Enable AWS SDK logging:
```go
cfg, _ := config.LoadDefaultConfig(context.Background(),
    config.WithClientLogMode(aws.LogRequestWithBody | aws.LogResponseWithBody),
)
```

2. Log expanded keys:
```go
// Add logging in your index map expansion
fmt.Printf("Expanded PK: %s, SK: %s\n", pk, sk)
```

3. Verify registrations:
```go
// Check if type is registered
if !registry.IsTypeRegistered("User") {
    log.Fatal("User type not registered")
}
```

## Migration Guide

### From Raw DynamoDB SDK

1. Define your entities with proper JSON tags
2. Create index maps for your access patterns
3. Replace SDK calls with DataStore methods
4. Handle errors appropriately

### From Other ORMs

1. Map your models to EntityStore entities
2. Convert query patterns to DynamoDB expressions
3. Implement type registration
4. Update error handling