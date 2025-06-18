# EntityStore Migration Guide

This guide helps you migrate from the old non-generic Storage interface to the new type-safe storage system.

## Overview of Changes

### Old API (Non-Generic)
```go
// Old Storage interface used empty interface (any)
storage := entitystore.NewStorageManager()
storage.RegisterDataStore("users", userStore) // No type safety
ds, _ := storage.GetDataStore("users")
// Required type assertion
userStore := ds.(datastore.DataStore[User])
```

### New API (Type-Safe)
```go
// New MultiTypeStorage with full type safety
mts := entitystore.NewMultiTypeStorage()
entitystore.RegisterDataStore(mts, "users", userStore) // Type-safe
userStore, _ := entitystore.GetDataStore[User](mts, "users") // No type assertion needed
```

## Migration Steps

### Step 1: Update Imports

Add the new imports if needed:
```go
import (
    "github.com/suparena/entitystore"
    "github.com/suparena/entitystore/datastore"
    "github.com/suparena/entitystore/datastore/ddb"
)
```

### Step 2: Replace Storage Manager

Replace your old storage manager initialization:

**Before:**
```go
storage := entitystore.NewStorageManager()
```

**After:**
```go
mts := entitystore.NewMultiTypeStorage()
```

### Step 3: Update Registration Calls

Update all datastore registration calls to use the new generic functions:

**Before:**
```go
userStore, err := ddb.NewDynamodbDataStore[User](...)
if err != nil {
    return err
}
storage.RegisterDataStore("users", userStore)
```

**After:**
```go
userStore, err := ddb.NewDynamodbDataStore[User](...)
if err != nil {
    return err
}
entitystore.RegisterDataStore(mts, "users", userStore)
```

### Step 4: Update Retrieval Calls

Update all datastore retrieval calls to use generic functions:

**Before:**
```go
ds, err := storage.GetDataStore("users")
if err != nil {
    return err
}
userStore := ds.(datastore.DataStore[User]) // Type assertion required
```

**After:**
```go
userStore, err := entitystore.GetDataStore[User](mts, "users")
if err != nil {
    return err
}
// No type assertion needed!
```

### Step 5: Update Service Layer

If you have a service layer that uses storage, update it:

**Before:**
```go
type UserService struct {
    storage entitystore.Storage
}

func (s *UserService) GetUserStore() (datastore.DataStore[User], error) {
    ds, err := s.storage.GetDataStore("users")
    if err != nil {
        return nil, err
    }
    return ds.(datastore.DataStore[User]), nil
}
```

**After:**
```go
type UserService struct {
    storage *entitystore.MultiTypeStorage
}

func (s *UserService) GetUserStore() (datastore.DataStore[User], error) {
    return entitystore.GetDataStore[User](s.storage, "users")
}
```

## Advanced Usage

### Using TypedStorage Directly

For cases where you only work with one type, you can use TypedStorage directly:

```go
// Create a typed storage for User entities only
userStorage := entitystore.NewTypedStorage[User]()

// Register multiple user-related datastores
userStorage.Register("users-primary", primaryUserStore)
userStorage.Register("users-cache", cacheUserStore)

// Get specific datastore
store, err := userStorage.Get("users-primary")
```

### Organizing by Domain

You can organize your storage by domain:

```go
type Application struct {
    storage *entitystore.MultiTypeStorage
}

func (app *Application) RegisterUserDomain() error {
    userStore, err := ddb.NewDynamodbDataStore[User](...)
    if err != nil {
        return err
    }
    
    profileStore, err := ddb.NewDynamodbDataStore[UserProfile](...)
    if err != nil {
        return err
    }
    
    entitystore.RegisterDataStore(app.storage, "users", userStore)
    entitystore.RegisterDataStore(app.storage, "profiles", profileStore)
    
    return nil
}
```

## Benefits of Migration

1. **Type Safety**: No more runtime type assertions
2. **Compile-Time Checks**: Errors caught at compile time
3. **Better IDE Support**: Auto-completion works correctly
4. **Cleaner Code**: Less boilerplate for type conversions
5. **Thread Safety**: Built-in concurrent access protection

## Compatibility Notes

- The old Storage interface remains available for backward compatibility
- Both systems can coexist during migration
- New features will only be added to the generic storage system

## Common Issues

### Issue: "type mismatch" errors
Make sure you're using the correct type parameter when calling GetDataStore:
```go
// Wrong
store, err := entitystore.GetDataStore[Product](mts, "users") // Type mismatch

// Correct
store, err := entitystore.GetDataStore[User](mts, "users")
```

### Issue: Multiple datastores with same key
The new system allows the same key for different types:
```go
// These can coexist
entitystore.RegisterDataStore(mts, "items", userStore)    // For User type
entitystore.RegisterDataStore(mts, "items", productStore) // For Product type

// Retrieve with correct type
users, _ := entitystore.GetDataStore[User](mts, "items")
products, _ := entitystore.GetDataStore[Product](mts, "items")
```