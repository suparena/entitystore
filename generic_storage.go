/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package entitystore

import (
	"fmt"
	"reflect"
	"sync"
	
	"github.com/suparena/entitystore/datastore"
)

// TypedStorage provides type-safe storage operations for a specific type T
type TypedStorage[T any] struct {
	mu     sync.RWMutex
	stores map[string]datastore.DataStore[T]
}

// NewTypedStorage creates a new TypedStorage for type T
func NewTypedStorage[T any]() *TypedStorage[T] {
	return &TypedStorage[T]{
		stores: make(map[string]datastore.DataStore[T]),
	}
}

// Register adds a datastore with the given key
func (ts *TypedStorage[T]) Register(key string, ds datastore.DataStore[T]) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	if _, exists := ts.stores[key]; exists {
		return fmt.Errorf("datastore with key %q already registered", key)
	}
	
	ts.stores[key] = ds
	return nil
}

// Get retrieves a datastore by key
func (ts *TypedStorage[T]) Get(key string) (datastore.DataStore[T], error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	
	ds, exists := ts.stores[key]
	if !exists {
		return nil, fmt.Errorf("datastore with key %q not found", key)
	}
	
	return ds, nil
}

// Remove deletes a datastore by key
func (ts *TypedStorage[T]) Remove(key string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	if _, exists := ts.stores[key]; !exists {
		return fmt.Errorf("datastore with key %q not found", key)
	}
	
	delete(ts.stores, key)
	return nil
}

// List returns all registered datastore keys
func (ts *TypedStorage[T]) List() []string {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	
	keys := make([]string, 0, len(ts.stores))
	for k := range ts.stores {
		keys = append(keys, k)
	}
	return keys
}

// MultiTypeStorage manages TypedStorage instances for different types
type MultiTypeStorage struct {
	mu       sync.RWMutex
	storages map[reflect.Type]interface{}
}

// NewMultiTypeStorage creates a new MultiTypeStorage
func NewMultiTypeStorage() *MultiTypeStorage {
	return &MultiTypeStorage{
		storages: make(map[reflect.Type]interface{}),
	}
}

// GetTypedStorage returns a TypedStorage for the specified type, creating it if necessary
func GetTypedStorage[T any](mts *MultiTypeStorage) *TypedStorage[T] {
	mts.mu.Lock()
	defer mts.mu.Unlock()
	
	var zero T
	typ := reflect.TypeOf(zero)
	
	if storage, exists := mts.storages[typ]; exists {
		return storage.(*TypedStorage[T])
	}
	
	// Create new typed storage
	newStorage := NewTypedStorage[T]()
	mts.storages[typ] = newStorage
	return newStorage
}

// Example helper functions for common operations

// RegisterDataStore is a convenience function to register a datastore for type T
func RegisterDataStore[T any](mts *MultiTypeStorage, key string, ds datastore.DataStore[T]) error {
	storage := GetTypedStorage[T](mts)
	return storage.Register(key, ds)
}

// GetDataStore is a convenience function to get a datastore for type T
func GetDataStore[T any](mts *MultiTypeStorage, key string) (datastore.DataStore[T], error) {
	storage := GetTypedStorage[T](mts)
	return storage.Get(key)
}

// RemoveDataStore is a convenience function to remove a datastore for type T
func RemoveDataStore[T any](mts *MultiTypeStorage, key string) error {
	storage := GetTypedStorage[T](mts)
	return storage.Remove(key)
}

// ListDataStores is a convenience function to list all datastores for type T
func ListDataStores[T any](mts *MultiTypeStorage) []string {
	storage := GetTypedStorage[T](mts)
	return storage.List()
}