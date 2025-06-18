/*
Package datastore defines the core interfaces for EntityStore's data persistence layer.

The main interface is DataStore[T], which provides generic CRUD operations for any entity type T:

	type DataStore[T any] interface {
	    GetOne(ctx context.Context, key string) (*T, error)
	    Put(ctx context.Context, entity T) error
	    UpdateWithCondition(ctx context.Context, keyInput any, updates map[string]interface{}, condition string) error
	    Query(ctx context.Context, params *storagemodels.QueryParams) ([]interface{}, error)
	    Stream(ctx context.Context, params *storagemodels.QueryParams, opts ...storagemodels.StreamOption) <-chan storagemodels.StreamResult[T]
	    Delete(ctx context.Context, key string) error
	}

Implementations:
  - ddb: DynamoDB implementation with support for single-table design
  - mock: In-memory mock implementation for testing

The package uses Go generics to ensure type safety at compile time while maintaining
flexibility for different storage backends.
*/
package datastore