/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package mock_test

import (
	"context"
	"testing"
	"time"
	
	"github.com/suparena/entitystore/datastore/mock"
	"github.com/suparena/entitystore/errors"
	"github.com/suparena/entitystore/storagemodels"
)

type TestEntity struct {
	ID   string
	Name string
}

func TestMockDataStore(t *testing.T) {
	ctx := context.Background()
	
	t.Run("BasicOperations", func(t *testing.T) {
		// Create mock with custom key extractor
		mockStore := mock.New[TestEntity]().
			WithGetKeyFunc(func(e TestEntity) string { return e.ID })
		
		// Test Put
		entity := TestEntity{ID: "123", Name: "Test"}
		err := mockStore.Put(ctx, entity)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}
		
		// Test GetOne
		retrieved, err := mockStore.GetOne(ctx, "123")
		if err != nil {
			t.Fatalf("GetOne failed: %v", err)
		}
		if retrieved.ID != "123" || retrieved.Name != "Test" {
			t.Fatalf("Retrieved entity mismatch: %+v", retrieved)
		}
		
		// Test Delete
		err = mockStore.Delete(ctx, "123")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		
		// Verify deletion
		_, err = mockStore.GetOne(ctx, "123")
		if !errors.IsNotFound(err) {
			t.Fatalf("Expected not found error, got: %v", err)
		}
	})
	
	t.Run("ErrorSimulation", func(t *testing.T) {
		mockStore := mock.New[TestEntity]()
		
		// Simulate Put error
		putErr := errors.NewValidationError("name", "required")
		mockStore.WithPutError(putErr)
		
		entity := TestEntity{ID: "123", Name: "Test"}
		err := mockStore.Put(ctx, entity)
		if err != putErr {
			t.Fatalf("Expected put error, got: %v", err)
		}
		
		// Simulate Delete error
		deleteErr := errors.NewConditionFailedError("delete", "version mismatch")
		mockStore.WithDeleteError(deleteErr)
		
		err = mockStore.Delete(ctx, "123")
		if err != deleteErr {
			t.Fatalf("Expected delete error, got: %v", err)
		}
	})
	
	t.Run("QueryAndStream", func(t *testing.T) {
		mockStore := mock.New[TestEntity]().
			WithGetKeyFunc(func(e TestEntity) string { return e.ID })
		
		// Add test data
		entities := []TestEntity{
			{ID: "1", Name: "One"},
			{ID: "2", Name: "Two"},
			{ID: "3", Name: "Three"},
		}
		
		for _, e := range entities {
			mockStore.Put(ctx, e)
		}
		
		// Test Query
		params := &storagemodels.QueryParams{
			TableName: "test",
		}
		results, err := mockStore.Query(ctx, params)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("Expected 3 results, got %d", len(results))
		}
		
		// Test Stream
		streamCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()
		
		resultChan := mockStore.Stream(streamCtx, params)
		count := 0
		for result := range resultChan {
			if result.Error != nil {
				t.Fatalf("Stream error: %v", result.Error)
			}
			count++
		}
		if count != 3 {
			t.Fatalf("Expected 3 streamed items, got %d", count)
		}
	})
	
	t.Run("CustomQueryFunction", func(t *testing.T) {
		mockStore := mock.New[TestEntity]()
		
		// Set custom query function that filters by name
		mockStore.WithQueryFunc(func(ctx context.Context, params *storagemodels.QueryParams) ([]interface{}, error) {
			// Simulate filtering
			return []interface{}{
				TestEntity{ID: "1", Name: "Filtered"},
			}, nil
		})
		
		params := &storagemodels.QueryParams{
			TableName: "test",
		}
		results, err := mockStore.Query(ctx, params)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
	})
	
	t.Run("HelperMethods", func(t *testing.T) {
		mockStore := mock.New[TestEntity]().
			WithGetKeyFunc(func(e TestEntity) string { return e.ID })
		
		// Test SetData
		testData := map[string]TestEntity{
			"1": {ID: "1", Name: "One"},
			"2": {ID: "2", Name: "Two"},
		}
		mockStore.SetData(testData)
		
		// Test Count
		if mockStore.Count() != 2 {
			t.Fatalf("Expected count 2, got %d", mockStore.Count())
		}
		
		// Test GetData
		data := mockStore.GetData()
		if len(data) != 2 {
			t.Fatalf("Expected 2 items in data, got %d", len(data))
		}
		
		// Test Clear
		mockStore.Clear()
		if mockStore.Count() != 0 {
			t.Fatalf("Expected count 0 after clear, got %d", mockStore.Count())
		}
	})
}

func TestMockDataStoreWithService(t *testing.T) {
	// Example of using mock in a service test
	type UserService struct {
		store interface {
			GetOne(ctx context.Context, key string) (*TestEntity, error)
			Put(ctx context.Context, entity TestEntity) error
		}
	}
	
	ctx := context.Background()
	mockStore := mock.New[TestEntity]().
		WithGetKeyFunc(func(e TestEntity) string { return e.ID })
	
	service := UserService{store: mockStore}
	
	// Test service method
	user := TestEntity{ID: "123", Name: "John"}
	err := service.store.Put(ctx, user)
	if err != nil {
		t.Fatalf("Service put failed: %v", err)
	}
	
	retrieved, err := service.store.GetOne(ctx, "123")
	if err != nil {
		t.Fatalf("Service get failed: %v", err)
	}
	if retrieved.Name != "John" {
		t.Fatalf("Expected name John, got %s", retrieved.Name)
	}
}