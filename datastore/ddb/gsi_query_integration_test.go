// +build integration

/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package ddb

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/suparena/entitystore/registry"
	"github.com/suparena/entitystore/storagemodels"
)

// TestEntity for GSI integration testing
type GSIIntegrationTestEntity struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	Country   string    `json:"country"`
	Score     int       `json:"score"`
	CreatedAt time.Time `json:"createdAt"`
}

func init() {
	// Register test entity
	registry.RegisterType("GSIIntegrationTestEntity", func(item map[string]types.AttributeValue) (interface{}, error) {
		entity := &GSIIntegrationTestEntity{}
		// In real code, you would unmarshal the item into entity
		return entity, nil
	})

	// Register index map with GSI patterns
	indexMap := map[string]string{
		"PK":     "ENTITY#{ID}",
		"SK":     "ENTITY#{ID}",
		"GSI1PK": "EMAIL#{Email}",
		"GSI1SK": "STATUS#{Status}#CREATED#{CreatedAt}",
	}
	registry.RegisterIndexMap[GSIIntegrationTestEntity](indexMap)
}

func TestGSIQueryIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup test store
	store := setupTestStore[GSIIntegrationTestEntity](t)
	ctx := context.Background()

	// Create test data
	testEntities := []GSIIntegrationTestEntity{
		{
			ID:        "user1",
			Email:     "user1@example.com",
			Status:    "active",
			Country:   "USA",
			Score:     100,
			CreatedAt: time.Now().Add(-24 * time.Hour),
		},
		{
			ID:        "user2",
			Email:     "user2@example.com",
			Status:    "active",
			Country:   "UK",
			Score:     90,
			CreatedAt: time.Now().Add(-12 * time.Hour),
		},
		{
			ID:        "user3",
			Email:     "user1@example.com", // Same email, different user
			Status:    "inactive",
			Country:   "USA",
			Score:     80,
			CreatedAt: time.Now().Add(-6 * time.Hour),
		},
		{
			ID:        "user4",
			Email:     "user3@example.com",
			Status:    "pending",
			Country:   "Canada",
			Score:     95,
			CreatedAt: time.Now().Add(-3 * time.Hour),
		},
	}

	// Insert test data
	for _, entity := range testEntities {
		err := store.Put(ctx, entity)
		if err != nil {
			t.Fatalf("Failed to put entity: %v", err)
		}
	}

	// Wait for eventual consistency
	time.Sleep(1 * time.Second)

	t.Run("QueryByGSI1PK", func(t *testing.T) {
		// Query by email
		results, err := store.QueryByGSI1PK(ctx, "user1@example.com")
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		// Should find 2 users with this email
		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}

		// Verify results
		for _, result := range results {
			if result.Email != "user1@example.com" {
				t.Errorf("Expected email user1@example.com, got %s", result.Email)
			}
		}
	})

	t.Run("QueryByGSI1PKAndSKPrefix", func(t *testing.T) {
		// Query by email and status prefix
		results, err := store.QueryByGSI1PKAndSKPrefix(ctx, "user1@example.com", "STATUS#active")
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		// Should find 1 active user with this email
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}

		if len(results) > 0 && results[0].Status != "active" {
			t.Errorf("Expected status active, got %s", results[0].Status)
		}
	})

	t.Run("QueryWithGSIBuilder", func(t *testing.T) {
		// Complex query using builder
		results, err := store.QueryGSI().
			WithPartitionKey("user2@example.com").
			WithSortKeyPrefix("STATUS#act").
			WithLimit(10).
			Execute(ctx)

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		// Should find 1 result
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("QueryWithFilter", func(t *testing.T) {
		// Query with additional filter
		filterValues := map[string]types.AttributeValue{
			":country": &types.AttributeValueMemberS{Value: "USA"},
			":score":   &types.AttributeValueMemberN{Value: "85"},
		}

		results, err := store.QueryGSI().
			WithPartitionKey("user1@example.com").
			WithFilter("Country = :country AND Score > :score", filterValues).
			Execute(ctx)

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		// Should find 1 result (user1 with score 100 > 85)
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}

		if len(results) > 0 {
			if results[0].Country != "USA" {
				t.Errorf("Expected country USA, got %s", results[0].Country)
			}
			if results[0].Score <= 85 {
				t.Errorf("Expected score > 85, got %d", results[0].Score)
			}
		}
	})

	t.Run("QueryWithSortKeyRange", func(t *testing.T) {
		// Query all emails with status between active and pending
		results, err := store.QueryGSI().
			WithPartitionKey("user1@example.com").
			WithSortKeyBetween("STATUS#active", "STATUS#pending").
			Execute(ctx)

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		// Should find both active and inactive users
		if len(results) < 1 {
			t.Errorf("Expected at least 1 result, got %d", len(results))
		}
	})

	t.Run("StreamGSIQuery", func(t *testing.T) {
		// Stream results using GSI
		// Use stream options

		resultCh := store.QueryGSI().
			WithPartitionKey("user1@example.com").
			Stream(ctx, storagemodels.WithBufferSize(10))

		count := 0
		for result := range resultCh {
			if result.Error != nil {
				t.Errorf("Stream error: %v", result.Error)
				continue
			}
			count++
			if result.Item.Email != "user1@example.com" {
				t.Errorf("Expected email user1@example.com, got %s", result.Item.Email)
			}
		}

		if count != 2 {
			t.Errorf("Expected 2 streamed results, got %d", count)
		}
	})

	t.Run("EmptyGSIQueryResult", func(t *testing.T) {
		// Query for non-existent email
		results, err := store.QueryByGSI1PK(ctx, "nonexistent@example.com")
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		// Should return empty results, not error
		if len(results) != 0 {
			t.Errorf("Expected 0 results, got %d", len(results))
		}
	})

	// Cleanup test data
	for _, entity := range testEntities {
		_ = store.Delete(ctx, entity.ID)
	}
}

func TestGSIQueryPatterns(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	store := setupTestStore[GSIIntegrationTestEntity](t)
	ctx := context.Background()

	// Test common query patterns
	t.Run("EmailLookupPattern", func(t *testing.T) {
		// Common pattern: Look up all records for an email
		email := "test@example.com"
		
		// Create test data with same email
		entities := []GSIIntegrationTestEntity{
			{ID: "1", Email: email, Status: "active"},
			{ID: "2", Email: email, Status: "inactive"},
			{ID: "3", Email: email, Status: "pending"},
		}

		for _, e := range entities {
			_ = store.Put(ctx, e)
		}

		// Query all records for this email
		results, err := store.QueryByGSI1PK(ctx, email)
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		// Cleanup
		for _, e := range entities {
			_ = store.Delete(ctx, e.ID)
		}
	})

	t.Run("StatusFilterPattern", func(t *testing.T) {
		// Common pattern: Find all active users
		entities := []GSIIntegrationTestEntity{
			{ID: "1", Email: "a@test.com", Status: "active"},
			{ID: "2", Email: "b@test.com", Status: "active"},
			{ID: "3", Email: "c@test.com", Status: "inactive"},
		}

		for _, e := range entities {
			_ = store.Put(ctx, e)
		}

		// This requires scanning or using a different access pattern
		// With single GSI, we need to query by email first
		// This demonstrates the limitation of single GSI

		// Cleanup
		for _, e := range entities {
			_ = store.Delete(ctx, e.ID)
		}
	})
}

// setupTestStore creates a test DynamoDB store
func setupTestStore[T any](t *testing.T) *DynamodbDataStore[T] {
	// This would use the test setup from your existing integration tests
	// For now, returning a placeholder
	t.Helper()
	
	// Mock setup - in real integration tests, this would connect to DynamoDB Local
	// or use the existing test infrastructure
	t.Skip("Integration test setup not implemented")
	
	return nil
}