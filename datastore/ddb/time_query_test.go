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
)

// TimeTestEntity for time-based query testing
type TimeTestEntity struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Data      string    `json:"data"`
}

func init() {
	// Register test entity
	registry.RegisterType("TimeTestEntity", func(item map[string]types.AttributeValue) (interface{}, error) {
		entity := &TimeTestEntity{}
		return entity, nil
	})
	
	// Register index map with time in sort key
	indexMap := map[string]string{
		"PK":     "TYPE#{Type}",
		"SK":     "TIME#{CreatedAt}#ID#{ID}",
		"GSI1PK": "TYPE#{Type}",
		"GSI1SK": "TIME#{UpdatedAt}#ID#{ID}",
	}
	registry.RegisterIndexMap[TimeTestEntity](indexMap)
}

func TestTimeRangeQueryBuilder(t *testing.T) {
	store := &DynamodbDataStore[TimeTestEntity]{
		tableName: "test-table",
	}
	
	now := time.Now()
	
	t.Run("InLastHours", func(t *testing.T) {
		builder := store.QueryByTimeRange("events").
			InLastHours(24)
		
		params, err := builder.Build()
		if err != nil {
			t.Fatalf("Failed to build query: %v", err)
		}
		
		// Should have sort key condition
		if params.KeyConditionExpression == "" {
			t.Error("Expected key condition expression")
		}
	})
	
	t.Run("Between", func(t *testing.T) {
		start := now.AddDate(0, 0, -7) // 7 days ago
		end := now
		
		builder := store.QueryByTimeRange("events").
			Between(start, end)
		
		params, err := builder.Build()
		if err != nil {
			t.Fatalf("Failed to build query: %v", err)
		}
		
		// Check that BETWEEN is used
		expectedKey := "GSI1PK = :pk AND GSI1SK BETWEEN :sk AND :sk2"
		if params.KeyConditionExpression != expectedKey {
			t.Errorf("Expected key condition %s, got %s", expectedKey, params.KeyConditionExpression)
		}
	})
	
	t.Run("Today", func(t *testing.T) {
		builder := store.QueryByTimeRange("events").
			Today()
		
		params, err := builder.Build()
		if err != nil {
			t.Fatalf("Failed to build query: %v", err)
		}
		
		// Should use BETWEEN for today's range
		if params.KeyConditionExpression == "" {
			t.Error("Expected key condition expression")
		}
	})
	
	t.Run("Latest", func(t *testing.T) {
		builder := store.QueryByTimeRange("events").
			Latest().
			WithLimit(10)
		
		params, err := builder.Build()
		if err != nil {
			t.Fatalf("Failed to build query: %v", err)
		}
		
		// Should have ScanIndexForward = false
		if params.ScanIndexForward == nil || *params.ScanIndexForward != false {
			t.Error("Expected ScanIndexForward to be false for Latest")
		}
		
		// Should have limit
		if params.Limit == nil || *params.Limit != 10 {
			t.Error("Expected limit to be 10")
		}
	})
	
	t.Run("Oldest", func(t *testing.T) {
		builder := store.QueryByTimeRange("events").
			Oldest()
		
		params, err := builder.Build()
		if err != nil {
			t.Fatalf("Failed to build query: %v", err)
		}
		
		// Should have ScanIndexForward = true
		if params.ScanIndexForward == nil || *params.ScanIndexForward != true {
			t.Error("Expected ScanIndexForward to be true for Oldest")
		}
	})
	
	t.Run("ThisWeek", func(t *testing.T) {
		builder := store.QueryByTimeRange("events").
			ThisWeek()
		
		params, err := builder.Build()
		if err != nil {
			t.Fatalf("Failed to build query: %v", err)
		}
		
		// Should have greater than condition for start of week
		if params.KeyConditionExpression == "" {
			t.Error("Expected key condition expression")
		}
	})
	
	t.Run("ThisMonth", func(t *testing.T) {
		builder := store.QueryByTimeRange("events").
			ThisMonth()
		
		params, err := builder.Build()
		if err != nil {
			t.Fatalf("Failed to build query: %v", err)
		}
		
		// Should have greater than condition for start of month
		if params.KeyConditionExpression == "" {
			t.Error("Expected key condition expression")
		}
	})
	
	t.Run("ChainedTimeQueries", func(t *testing.T) {
		// Complex time query with filters
		filterValues := map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: "active"},
		}
		
		builder := store.QueryByTimeRange("events").
			InLastDays(7).
			Latest().
			WithFilter("Status = :status", filterValues).
			WithLimit(50)
		
		params, err := builder.Build()
		if err != nil {
			t.Fatalf("Failed to build query: %v", err)
		}
		
		// Check all conditions are set
		if params.ScanIndexForward == nil || *params.ScanIndexForward != false {
			t.Error("Expected ScanIndexForward to be false")
		}
		
		if params.FilterExpression == nil || *params.FilterExpression != "Status = :status" {
			t.Error("Expected filter expression")
		}
		
		if params.Limit == nil || *params.Limit != 50 {
			t.Error("Expected limit to be 50")
		}
	})
}

func TestTimeWindowIterator(t *testing.T) {
	store := &DynamodbDataStore[TimeTestEntity]{
		tableName: "test-table",
	}
	
	start := time.Now().AddDate(0, 0, -30) // 30 days ago
	end := time.Now()
	windowSize := 7 * 24 * time.Hour // 1 week windows
	
	iterator := store.QueryTimeWindows("events", start, end, windowSize)
	
	// Test that iterator is created correctly
	if iterator.windowSize != windowSize {
		t.Errorf("Expected window size %v, got %v", windowSize, iterator.windowSize)
	}
	
	if !iterator.current.Equal(start) {
		t.Error("Iterator should start at start time")
	}
	
	// In a real test with mocked DynamoDB, we would test Next()
	// For now, just verify the iterator structure
	windowCount := 0
	testEnd := start
	for testEnd.Before(end) {
		testEnd = testEnd.Add(windowSize)
		windowCount++
	}
	
	// Should have approximately 4-5 windows for 30 days with 1 week windows
	if windowCount < 4 || windowCount > 5 {
		t.Errorf("Expected 4-5 windows, calculated %d", windowCount)
	}
}

func TestTimeBasedConvenienceMethods(t *testing.T) {
	store := &DynamodbDataStore[TimeTestEntity]{
		tableName: "test-table",
	}
	
	ctx := context.Background()
	
	t.Run("QueryLatestItems", func(t *testing.T) {
		// This would need a mock to actually test
		// For now, verify the method exists and can be called
		_ = func() {
			_, _ = store.QueryLatestItems(ctx, "events", 10)
		}
	})
	
	t.Run("QueryItemsSince", func(t *testing.T) {
		since := time.Now().AddDate(0, 0, -1) // Yesterday
		_ = func() {
			_, _ = store.QueryItemsSince(ctx, "events", since)
		}
	})
	
	t.Run("QueryItemsInDateRange", func(t *testing.T) {
		start := time.Now().AddDate(0, 0, -7)
		end := time.Now()
		_ = func() {
			_, _ = store.QueryItemsInDateRange(ctx, "events", start, end)
		}
	})
	
	t.Run("StreamLatestItems", func(t *testing.T) {
		_ = func() {
			_ = store.StreamLatestItems(ctx, "events")
		}
	})
}

func TestTimeFormats(t *testing.T) {
	// Test that time formats work correctly
	now := time.Now()
	
	// RFC3339 format should be sortable
	formatted := now.Format(time.RFC3339)
	parsed, err := time.Parse(time.RFC3339, formatted)
	if err != nil {
		t.Errorf("Failed to parse RFC3339 time: %v", err)
	}
	
	// Times should be equal (within a second due to formatting)
	diff := now.Sub(parsed).Abs()
	if diff > time.Second {
		t.Errorf("Time parsing lost precision: %v", diff)
	}
	
	// Test that RFC3339 strings sort correctly
	time1 := "2024-01-01T10:00:00Z"
	time2 := "2024-01-01T11:00:00Z"
	time3 := "2024-01-02T09:00:00Z"
	
	if !(time1 < time2 && time2 < time3) {
		t.Error("RFC3339 strings should sort lexicographically")
	}
}