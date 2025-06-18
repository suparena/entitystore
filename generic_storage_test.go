/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package entitystore

import (
	"context"
	"fmt"
	"testing"
	
	"github.com/suparena/entitystore/datastore"
	"github.com/suparena/entitystore/storagemodels"
)

// mockDataStore is a simple mock implementation for testing
type mockDataStore[T any] struct {
	data map[string]T
}

func newMockDataStore[T any]() datastore.DataStore[T] {
	return &mockDataStore[T]{
		data: make(map[string]T),
	}
}

func (m *mockDataStore[T]) GetOne(ctx context.Context, key string) (*T, error) {
	if v, ok := m.data[key]; ok {
		return &v, nil
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockDataStore[T]) Put(ctx context.Context, entity T) error {
	return nil
}

func (m *mockDataStore[T]) UpdateWithCondition(ctx context.Context, keyInput any, updates map[string]interface{}, condition string) error {
	return nil
}

func (m *mockDataStore[T]) Query(ctx context.Context, params *storagemodels.QueryParams) ([]interface{}, error) {
	return nil, nil
}

func (m *mockDataStore[T]) Stream(ctx context.Context, params *storagemodels.QueryParams, opts ...storagemodels.StreamOption) <-chan storagemodels.StreamResult[T] {
	ch := make(chan storagemodels.StreamResult[T])
	close(ch)
	return ch
}

func (m *mockDataStore[T]) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

// Test types
type TestUser struct {
	ID    string
	Name  string
	Email string
}

type TestProduct struct {
	ID    string
	Name  string
	Price float64
}

func TestTypedStorage(t *testing.T) {
	t.Run("BasicOperations", func(t *testing.T) {
		storage := NewTypedStorage[TestUser]()
		
		// Register datastore
		userStore := newMockDataStore[TestUser]()
		err := storage.Register("users", userStore)
		if err != nil {
			t.Fatalf("Failed to register: %v", err)
		}
		
		// Get datastore
		retrieved, err := storage.Get("users")
		if err != nil {
			t.Fatalf("Failed to get: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Retrieved store is nil")
		}
		
		// List datastores
		keys := storage.List()
		if len(keys) != 1 || keys[0] != "users" {
			t.Fatalf("Expected [users], got %v", keys)
		}
		
		// Remove datastore
		err = storage.Remove("users")
		if err != nil {
			t.Fatalf("Failed to remove: %v", err)
		}
		
		// Verify removal
		_, err = storage.Get("users")
		if err == nil {
			t.Fatal("Expected error after removal")
		}
	})
	
	t.Run("DuplicateRegistration", func(t *testing.T) {
		storage := NewTypedStorage[TestUser]()
		
		userStore1 := newMockDataStore[TestUser]()
		err := storage.Register("users", userStore1)
		if err != nil {
			t.Fatalf("First registration failed: %v", err)
		}
		
		userStore2 := newMockDataStore[TestUser]()
		err = storage.Register("users", userStore2)
		if err == nil {
			t.Fatal("Expected duplicate registration error")
		}
	})
}

func TestMultiTypeStorage(t *testing.T) {
	mts := NewMultiTypeStorage()
	
	t.Run("DifferentTypes", func(t *testing.T) {
		// Register user datastore
		userStore := newMockDataStore[TestUser]()
		err := RegisterDataStore(mts, "users", userStore)
		if err != nil {
			t.Fatalf("Failed to register user store: %v", err)
		}
		
		// Register product datastore
		productStore := newMockDataStore[TestProduct]()
		err = RegisterDataStore(mts, "products", productStore)
		if err != nil {
			t.Fatalf("Failed to register product store: %v", err)
		}
		
		// Get user datastore
		retrievedUser, err := GetDataStore[TestUser](mts, "users")
		if err != nil {
			t.Fatalf("Failed to get user store: %v", err)
		}
		if retrievedUser == nil {
			t.Fatal("User store is nil")
		}
		
		// Get product datastore
		retrievedProduct, err := GetDataStore[TestProduct](mts, "products")
		if err != nil {
			t.Fatalf("Failed to get product store: %v", err)
		}
		if retrievedProduct == nil {
			t.Fatal("Product store is nil")
		}
		
		// List stores for each type
		userKeys := ListDataStores[TestUser](mts)
		if len(userKeys) != 1 || userKeys[0] != "users" {
			t.Fatalf("Expected user keys [users], got %v", userKeys)
		}
		
		productKeys := ListDataStores[TestProduct](mts)
		if len(productKeys) != 1 || productKeys[0] != "products" {
			t.Fatalf("Expected product keys [products], got %v", productKeys)
		}
	})
	
	t.Run("SameKeyDifferentTypes", func(t *testing.T) {
		// Register with same key but different types
		userStore := newMockDataStore[TestUser]()
		err := RegisterDataStore(mts, "items", userStore)
		if err != nil {
			t.Fatalf("Failed to register user store: %v", err)
		}
		
		productStore := newMockDataStore[TestProduct]()
		err = RegisterDataStore(mts, "items", productStore)
		if err != nil {
			t.Fatalf("Failed to register product store: %v", err)
		}
		
		// Both should succeed because they're different types
		userItems, err := GetDataStore[TestUser](mts, "items")
		if err != nil || userItems == nil {
			t.Fatal("Failed to get user items")
		}
		
		productItems, err := GetDataStore[TestProduct](mts, "items")
		if err != nil || productItems == nil {
			t.Fatal("Failed to get product items")
		}
	})
}

func TestThreadSafety(t *testing.T) {
	mts := NewMultiTypeStorage()
	done := make(chan bool)
	
	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(id int) {
			store := newMockDataStore[TestUser]()
			key := fmt.Sprintf("store%d", id)
			RegisterDataStore(mts, key, store)
			done <- true
		}(i)
	}
	
	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			ListDataStores[TestUser](mts)
			done <- true
		}()
	}
	
	// Wait for completion
	for i := 0; i < 20; i++ {
		<-done
	}
	
	// Verify all stores registered
	keys := ListDataStores[TestUser](mts)
	if len(keys) != 10 {
		t.Fatalf("Expected 10 stores, got %d", len(keys))
	}
}