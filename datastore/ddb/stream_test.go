/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package ddb

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/suparena/entitystore/storagemodels"
)

// TestStreamWithOptions tests the enhanced streaming with various options
func TestStreamWithOptions(t *testing.T) {
	// Skip if no test environment
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	
	// Test with different buffer sizes
	t.Run("BufferSize", func(t *testing.T) {
		ds := createTestDataStore(t)
		
		params := &storagemodels.QueryParams{
			TableName:              "test-table",
			KeyConditionExpression: "PK = :pk",
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{Value: "TEST#123"},
			},
		}
		
		resultChan := ds.Stream(ctx, params, 
			storagemodels.WithBufferSize(10),
		)
		
		// Consume results
		count := 0
		for range resultChan {
			count++
		}
		
		if count == 0 {
			t.Log("No items found in test table")
		}
	})
	
	// Test with progress handler
	t.Run("ProgressHandler", func(t *testing.T) {
		ds := createTestDataStore(t)
		
		var progressCalled int32
		progressHandler := func(p storagemodels.StreamProgress) {
			atomic.AddInt32(&progressCalled, 1)
			t.Logf("Progress: %d items, %d pages, rate: %.2f/s", 
				p.ItemsProcessed, p.PagesProcessed, p.CurrentRate)
		}
		
		params := &storagemodels.QueryParams{
			TableName:              "test-table",
			KeyConditionExpression: "PK = :pk",
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{Value: "TEST#123"},
			},
		}
		
		resultChan := ds.Stream(ctx, params,
			storagemodels.WithProgressHandler(progressHandler),
			storagemodels.WithPageSize(5),
		)
		
		// Consume results
		for range resultChan {
			// Just consume
		}
		
		if atomic.LoadInt32(&progressCalled) == 0 {
			t.Error("Progress handler was not called")
		}
	})
	
	// Test error handling
	t.Run("ErrorHandler", func(t *testing.T) {
		ds := createTestDataStore(t)
		
		errorCount := 0
		errorHandler := func(err error) bool {
			errorCount++
			t.Logf("Error handled: %v", err)
			return true // Continue on error
		}
		
		// Use invalid query to trigger error
		params := &storagemodels.QueryParams{
			TableName:              "test-table",
			KeyConditionExpression: "INVALID = :pk",
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{Value: "TEST#123"},
			},
		}
		
		resultChan := ds.Stream(ctx, params,
			storagemodels.WithErrorHandler(errorHandler),
			storagemodels.WithMaxRetries(0), // No retries to test error handler
		)
		
		// Consume results
		for result := range resultChan {
			if result.Error != nil {
				t.Logf("Got error result: %v", result.Error)
			}
		}
	})
	
	// Test context cancellation
	t.Run("ContextCancellation", func(t *testing.T) {
		ds := createTestDataStore(t)
		
		cancelCtx, cancel := context.WithCancel(ctx)
		
		params := &storagemodels.QueryParams{
			TableName:              "test-table",
			KeyConditionExpression: "PK = :pk",
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": &types.AttributeValueMemberS{Value: "TEST#123"},
			},
		}
		
		resultChan := ds.Stream(cancelCtx, params,
			storagemodels.WithPageSize(1), // Small pages to test cancellation
		)
		
		// Cancel after receiving first result
		gotFirst := false
		for result := range resultChan {
			if !gotFirst {
				gotFirst = true
				cancel() // Cancel context
			}
			if result.Error != nil {
				// Expected - context cancelled
				break
			}
		}
		
		if !gotFirst {
			t.Log("No items found to test cancellation")
		}
	})
}

// TestStreamRetryLogic tests the retry mechanism
func TestStreamRetryLogic(t *testing.T) {
	// This would require mocking the DynamoDB client
	// For now, we'll test the retry logic indirectly
	
	t.Run("RetryableError", func(t *testing.T) {
		// Test that isRetryableError correctly identifies retryable errors
		err1 := &types.ProvisionedThroughputExceededException{}
		if !isRetryableError(err1) {
			t.Error("ProvisionedThroughputExceededException should be retryable")
		}
		
		err2 := &types.RequestLimitExceeded{}
		if !isRetryableError(err2) {
			t.Error("RequestLimitExceeded should be retryable")
		}
		
		err3 := fmt.Errorf("some other error")
		if isRetryableError(err3) {
			t.Error("Generic error should not be retryable")
		}
	})
}

// TestStreamMetadata tests that metadata is correctly populated
func TestStreamMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	
	ctx := context.Background()
	ds := createTestDataStore(t)
	
	params := &storagemodels.QueryParams{
		TableName:              "test-table",
		KeyConditionExpression: "PK = :pk",
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: "TEST#123"},
		},
	}
	
	resultChan := ds.Stream(ctx, params,
		storagemodels.WithPageSize(2),
	)
	
	var lastIndex int64 = -1
	startTime := time.Now()
	
	for result := range resultChan {
		if result.Error != nil {
			t.Errorf("Unexpected error: %v", result.Error)
			continue
		}
		
		// Check metadata
		if result.Meta.Index <= lastIndex {
			t.Errorf("Index should be increasing: got %d after %d", 
				result.Meta.Index, lastIndex)
		}
		lastIndex = result.Meta.Index
		
		if result.Meta.PageNumber < 1 {
			t.Errorf("Page number should be >= 1, got %d", result.Meta.PageNumber)
		}
		
		if result.Meta.Timestamp.Before(startTime) {
			t.Error("Timestamp should be after test start time")
		}
		
		if result.Raw == nil {
			t.Error("Raw data should not be nil")
		}
	}
}

// Helper function to create test datastore
func createTestDataStore(t *testing.T) *DynamodbDataStore[TestEntity] {
	// This would normally use environment variables or test config
	// For now, return a mock or skip if not configured
	t.Skip("Test datastore creation not implemented - needs AWS credentials")
	return nil
}

// TestEntity for testing
type TestEntity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}