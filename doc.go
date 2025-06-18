/*
Package entitystore provides a sophisticated storage abstraction layer for Go applications,
offering type-safe, annotation-driven data persistence with support for multiple storage backends.

The library follows a design-time → build-time → runtime workflow:
  - Design-time: Define entities and annotate OpenAPI specs
  - Build-time: Generate type registrations and index mappings
  - Runtime: Use type-safe storage operations

Key Features:
  - Type-safe operations using Go generics
  - Multiple storage backend support (DynamoDB, Redis, more planned)
  - Annotation-driven code generation from OpenAPI specs
  - Enhanced streaming with retry logic and progress tracking
  - Semantic error types for better error handling
  - Thread-safe storage management
  - Comprehensive mock implementations for testing

Basic Usage:

	// Create a storage manager
	mts := entitystore.NewMultiTypeStorage()
	
	// Register a typed datastore
	userStore, _ := ddb.NewDynamodbDataStore[User](...)
	entitystore.RegisterDataStore(mts, "users", userStore)
	
	// Retrieve and use the datastore
	store, _ := entitystore.GetDataStore[User](mts, "users")
	user := User{ID: "123", Name: "John"}
	err := store.Put(ctx, user)

For more information, see the documentation at https://github.com/suparena/entitystore
*/
package entitystore