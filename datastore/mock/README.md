# Mock DataStore

The mock package provides a thread-safe, in-memory implementation of the `DataStore` interface for testing.

## Features

- Full implementation of `DataStore[T]` interface
- Thread-safe operations
- Configurable error injection
- Custom query and stream functions
- Helper methods for test setup

## Usage

### Basic Example

```go
import (
    "context"
    "testing"
    
    "github.com/suparena/entitystore/datastore/mock"
)

func TestMyService(t *testing.T) {
    // Create a mock datastore
    mockStore := mock.New[User]()
    
    // Configure key extraction
    mockStore.WithGetKeyFunc(func(u User) string {
        return u.ID
    })
    
    // Use in your service
    service := NewUserService(mockStore)
    
    // Test your service methods
    user := User{ID: "123", Name: "John"}
    err := service.CreateUser(context.Background(), user)
    // ... assertions
}
```

### Error Injection

Simulate errors to test error handling:

```go
// Simulate validation error on Put
mockStore.WithPutError(errors.NewValidationError("email", "invalid format"))

// Simulate not found error on Delete
mockStore.WithDeleteError(errors.NewNotFoundError("User", "123"))

// Simulate condition failed on Update
mockStore.WithUpdateError(errors.NewConditionFailedError("update", "version mismatch"))
```

### Custom Query Behavior

```go
// Set custom query function
mockStore.WithQueryFunc(func(ctx context.Context, params *QueryParams) ([]interface{}, error) {
    // Implement custom query logic
    if params.FilterExpression != nil {
        // Apply filtering
    }
    return filteredResults, nil
})
```

### Custom Stream Behavior

```go
// Set custom stream function
mockStore.WithStreamFunc(func(ctx context.Context, params *QueryParams, opts ...StreamOption) <-chan StreamResult[User] {
    ch := make(chan StreamResult[User])
    go func() {
        defer close(ch)
        // Stream custom results
        ch <- StreamResult[User]{Item: user1}
        ch <- StreamResult[User]{Item: user2}
    }()
    return ch
})
```

### Helper Methods

```go
// Pre-populate data
testData := map[string]User{
    "1": {ID: "1", Name: "Alice"},
    "2": {ID: "2", Name: "Bob"},
}
mockStore.SetData(testData)

// Check count
count := mockStore.Count() // Returns 2

// Get all data
data := mockStore.GetData() // Returns copy of internal map

// Clear all data
mockStore.Clear()
```

## Testing Patterns

### Table-Driven Tests

```go
func TestUserOperations(t *testing.T) {
    tests := []struct {
        name    string
        setup   func(*mock.DataStore[User])
        user    User
        wantErr bool
    }{
        {
            name: "successful create",
            setup: func(m *mock.DataStore[User]) {
                // No special setup needed
            },
            user: User{ID: "1", Name: "Alice"},
            wantErr: false,
        },
        {
            name: "validation error",
            setup: func(m *mock.DataStore[User]) {
                m.WithPutError(errors.NewValidationError("name", "required"))
            },
            user: User{ID: "2"},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockStore := mock.New[User]()
            tt.setup(mockStore)
            
            err := mockStore.Put(context.Background(), tt.user)
            if (err != nil) != tt.wantErr {
                t.Errorf("Put() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Dependency Injection

```go
type UserRepository interface {
    GetOne(ctx context.Context, key string) (*User, error)
    Put(ctx context.Context, entity User) error
}

type UserService struct {
    repo UserRepository
}

func TestUserService(t *testing.T) {
    mockRepo := mock.New[User]().
        WithGetKeyFunc(func(u User) string { return u.ID })
    
    service := &UserService{repo: mockRepo}
    // Test service methods...
}
```

## Best Practices

1. **Always set a key extractor** for Put operations:
   ```go
   mockStore.WithGetKeyFunc(func(e Entity) string { return e.ID })
   ```

2. **Use context for cancellation** in streaming tests:
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
   defer cancel()
   ```

3. **Clear data between tests** to avoid test pollution:
   ```go
   defer mockStore.Clear()
   ```

4. **Verify error types** using the errors package helpers:
   ```go
   if !errors.IsNotFound(err) {
       t.Errorf("expected NotFound error, got %v", err)
   }
   ```