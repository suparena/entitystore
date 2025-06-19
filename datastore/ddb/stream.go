/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package ddb

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/suparena/entitystore/registry"
	"github.com/suparena/entitystore/storagemodels"
)

// Stream performs an enhanced streaming query against DynamoDB with configurable options
func (d *DynamodbDataStore[T]) Stream(ctx context.Context, params *storagemodels.QueryParams, opts ...storagemodels.StreamOption) <-chan storagemodels.StreamResult[T] {
	// Apply options
	options := storagemodels.DefaultStreamOptions()
	for _, opt := range opts {
		opt(&options)
	}

	// Create buffered result channel
	resultCh := make(chan storagemodels.StreamResult[T], options.BufferSize)

	// Start streaming in background
	go d.streamWorker(ctx, params, options, resultCh)

	return resultCh
}

// streamWorker handles the actual streaming logic
func (d *DynamodbDataStore[T]) streamWorker(
	ctx context.Context,
	params *storagemodels.QueryParams,
	options storagemodels.StreamOptions,
	resultCh chan<- storagemodels.StreamResult[T],
) {
	defer close(resultCh)

	// Initialize progress tracking
	var itemIndex int64
	var pageNumber int
	startTime := time.Now()
	var errors []error
	var mu sync.Mutex

	// Progress reporting helper
	reportProgress := func(lastKey map[string]types.AttributeValue) {
		if options.ProgressHandler != nil {
			progress := storagemodels.StreamProgress{
				ItemsProcessed: atomic.LoadInt64(&itemIndex),
				PagesProcessed: pageNumber,
				LastKey:        lastKey,
				Errors:         errors,
				StartTime:      startTime,
			}
			
			// Calculate rate
			elapsed := time.Since(startTime).Seconds()
			if elapsed > 0 {
				progress.CurrentRate = float64(progress.ItemsProcessed) / elapsed
			}
			
			options.ProgressHandler(progress)
		}
	}

	// Build query input
	input := &dynamodb.QueryInput{
		TableName:                 &params.TableName,
		KeyConditionExpression:    &params.KeyConditionExpression,
		ExpressionAttributeValues: params.ExpressionAttributeValues,
		FilterExpression:          params.FilterExpression,
		IndexName:                 params.IndexName,
		Limit:                     aws.Int32(options.PageSize),
		ScanIndexForward:          params.ScanIndexForward,
	}

	var lastEvaluatedKey map[string]types.AttributeValue

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return
		default:
		}

		if lastEvaluatedKey != nil {
			input.ExclusiveStartKey = lastEvaluatedKey
		}

		// Execute query with retry logic
		out, err := d.queryWithRetry(ctx, input, options)
		if err != nil {
			// Handle error with error handler if provided
			if options.ErrorHandler != nil {
				if !options.ErrorHandler(err) {
					// Error handler says to stop
					resultCh <- storagemodels.StreamResult[T]{
						Error: fmt.Errorf("query failed after retries: %w", err),
						Meta: storagemodels.StreamMeta{
							Index:      atomic.LoadInt64(&itemIndex),
							PageNumber: pageNumber,
							Timestamp:  time.Now(),
						},
					}
					return
				}
			} else {
				// No error handler, send error and stop
				resultCh <- storagemodels.StreamResult[T]{
					Error: fmt.Errorf("query failed: %w", err),
					Meta: storagemodels.StreamMeta{
						Index:      atomic.LoadInt64(&itemIndex),
						PageNumber: pageNumber,
						Timestamp:  time.Now(),
					},
				}
				return
			}

			// Record error and continue
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
			continue
		}

		pageNumber++

		// Process items in current page
		for _, item := range out.Items {
			// Check context cancellation
			select {
			case <-ctx.Done():
				return
			default:
			}

			result := d.processItem(item, atomic.LoadInt64(&itemIndex), pageNumber)
			atomic.AddInt64(&itemIndex, 1)

			// Send result
			select {
			case <-ctx.Done():
				return
			case resultCh <- result:
			}

			// Record any item-level errors
			if result.Error != nil {
				mu.Lock()
				errors = append(errors, result.Error)
				mu.Unlock()
			}
		}

		// Report progress after each page
		reportProgress(out.LastEvaluatedKey)

		// Check for more pages
		if out.LastEvaluatedKey == nil || len(out.LastEvaluatedKey) == 0 {
			break
		}
		lastEvaluatedKey = out.LastEvaluatedKey
	}

	// Final progress report
	reportProgress(nil)
}

// queryWithRetry executes a query with configurable retry logic
func (d *DynamodbDataStore[T]) queryWithRetry(
	ctx context.Context,
	input *dynamodb.QueryInput,
	options storagemodels.StreamOptions,
) (*dynamodb.QueryOutput, error) {
	var lastErr error

	for attempt := 0; attempt <= options.MaxRetries; attempt++ {
		// Check context before retry
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Execute query
		out, err := d.client.Query(ctx, input)
		if err == nil {
			return out, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return nil, err
		}

		// Don't sleep after last attempt
		if attempt < options.MaxRetries {
			// Exponential backoff with jitter
			backoff := time.Duration(attempt+1) * options.RetryBackoff
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}
	}

	return nil, fmt.Errorf("query failed after %d retries: %w", options.MaxRetries, lastErr)
}

// processItem converts a DynamoDB item to a typed result
func (d *DynamodbDataStore[T]) processItem(
	item map[string]types.AttributeValue,
	index int64,
	pageNumber int,
) storagemodels.StreamResult[T] {
	timestamp := time.Now()
	meta := storagemodels.StreamMeta{
		Index:      index,
		PageNumber: pageNumber,
		Timestamp:  timestamp,
	}

	// Make a copy of the raw item
	rawCopy := make(map[string]types.AttributeValue, len(item))
	for k, v := range item {
		rawCopy[k] = v
	}

	// Extract EntityType
	var entityType string
	if attr, ok := item["EntityType"]; ok {
		if err := attributevalue.Unmarshal(attr, &entityType); err != nil {
			return storagemodels.StreamResult[T]{
				Error: fmt.Errorf("failed to unmarshal EntityType: %w", err),
				Raw:   rawCopy,
				Meta:  meta,
			}
		}
		// Remove EntityType from item before unmarshaling
		delete(item, "EntityType")
	}

	// Try to unmarshal as type T first
	var result T
	if err := attributevalue.UnmarshalMap(item, &result); err == nil {
		return storagemodels.StreamResult[T]{
			Item: result,
			Raw:  rawCopy,
			Meta: meta,
		}
	}

	// If direct unmarshal fails and we have EntityType, try registry
	if entityType != "" {
		unmarshalFn, err := registry.GetUnmarshalFunc(entityType)
		if err == nil {
			obj, err := unmarshalFn(item)
			if err == nil {
				// Type assertion to T
				if typedObj, ok := obj.(T); ok {
					return storagemodels.StreamResult[T]{
						Item: typedObj,
						Raw:  rawCopy,
						Meta: meta,
					}
				}
			}
		}
	}

	// If all else fails, return error
	return storagemodels.StreamResult[T]{
		Error: fmt.Errorf("failed to unmarshal item to type %T", result),
		Raw:   rawCopy,
		Meta:  meta,
	}
}

// isRetryableError determines if a DynamoDB error is retryable
func isRetryableError(err error) bool {
	// Check for specific retryable DynamoDB errors
	switch err.(type) {
	case *types.ProvisionedThroughputExceededException:
		return true
	case *types.RequestLimitExceeded:
		return true
	case *types.InternalServerError:
		return true
	}

	// Check for AWS SDK retryable errors
	if awsErr, ok := err.(interface{ IsRetryable() bool }); ok {
		return awsErr.IsRetryable()
	}

	return false
}