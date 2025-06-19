/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package ddb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/suparena/entitystore/storagemodels"
)

// TimeRangeQueryBuilder provides specialized methods for time-based queries
type TimeRangeQueryBuilder[T any] struct {
	*GSIQueryBuilder[T]
	timeField string
}

// QueryByTimeRange creates a new time-based query builder
func (d *DynamodbDataStore[T]) QueryByTimeRange(partitionKey string) *TimeRangeQueryBuilder[T] {
	return &TimeRangeQueryBuilder[T]{
		GSIQueryBuilder: d.QueryGSI().WithPartitionKey(partitionKey),
		timeField:       "CreatedAt", // Default time field
	}
}

// WithTimeField specifies which time field to use for sorting (default: CreatedAt)
func (q *TimeRangeQueryBuilder[T]) WithTimeField(field string) *TimeRangeQueryBuilder[T] {
	q.timeField = field
	return q
}

// InLastHours queries items created/updated in the last N hours
func (q *TimeRangeQueryBuilder[T]) InLastHours(hours int) *TimeRangeQueryBuilder[T] {
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)
	q.WithSortKeyGreaterThan(startTime.Format(time.RFC3339))
	return q
}

// InLastDays queries items created/updated in the last N days
func (q *TimeRangeQueryBuilder[T]) InLastDays(days int) *TimeRangeQueryBuilder[T] {
	startTime := time.Now().AddDate(0, 0, -days)
	q.WithSortKeyGreaterThan(startTime.Format(time.RFC3339))
	return q
}

// Between queries items between two timestamps
func (q *TimeRangeQueryBuilder[T]) Between(start, end time.Time) *TimeRangeQueryBuilder[T] {
	q.WithSortKeyBetween(start.Format(time.RFC3339), end.Format(time.RFC3339))
	return q
}

// After queries items after a specific timestamp
func (q *TimeRangeQueryBuilder[T]) After(timestamp time.Time) *TimeRangeQueryBuilder[T] {
	q.WithSortKeyGreaterThan(timestamp.Format(time.RFC3339))
	return q
}

// Before queries items before a specific timestamp
func (q *TimeRangeQueryBuilder[T]) Before(timestamp time.Time) *TimeRangeQueryBuilder[T] {
	q.WithSortKeyLessThan(timestamp.Format(time.RFC3339))
	return q
}

// Today queries items created/updated today
func (q *TimeRangeQueryBuilder[T]) Today() *TimeRangeQueryBuilder[T] {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	return q.Between(startOfDay, endOfDay)
}

// ThisWeek queries items from the current week
func (q *TimeRangeQueryBuilder[T]) ThisWeek() *TimeRangeQueryBuilder[T] {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday as last day of week
	}
	startOfWeek := now.AddDate(0, 0, -weekday+1)
	startOfWeek = time.Date(startOfWeek.Year(), startOfWeek.Month(), startOfWeek.Day(), 0, 0, 0, 0, startOfWeek.Location())
	return q.After(startOfWeek)
}

// ThisMonth queries items from the current month
func (q *TimeRangeQueryBuilder[T]) ThisMonth() *TimeRangeQueryBuilder[T] {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	return q.After(startOfMonth)
}

// WithTimeOrder sets the sort order for time-based results
func (q *TimeRangeQueryBuilder[T]) WithTimeOrder(ascending bool) *TimeRangeQueryBuilder[T] {
	q.params.ScanIndexForward = aws.Bool(ascending)
	return q
}

// Latest returns results in descending time order (newest first)
func (q *TimeRangeQueryBuilder[T]) Latest() *TimeRangeQueryBuilder[T] {
	return q.WithTimeOrder(false)
}

// Oldest returns results in ascending time order (oldest first)
func (q *TimeRangeQueryBuilder[T]) Oldest() *TimeRangeQueryBuilder[T] {
	return q.WithTimeOrder(true)
}

// Execute runs the query and returns results
func (q *TimeRangeQueryBuilder[T]) Execute(ctx context.Context) ([]T, error) {
	return q.GSIQueryBuilder.Execute(ctx)
}

// Build constructs the final query parameters
func (q *TimeRangeQueryBuilder[T]) Build() (*storagemodels.QueryParams, error) {
	return q.GSIQueryBuilder.Build()
}

// Stream executes the query as a stream
func (q *TimeRangeQueryBuilder[T]) Stream(ctx context.Context, opts ...storagemodels.StreamOption) <-chan storagemodels.StreamResult[T] {
	return q.GSIQueryBuilder.Stream(ctx, opts...)
}

// WithLimit sets the query limit
func (q *TimeRangeQueryBuilder[T]) WithLimit(limit int32) *TimeRangeQueryBuilder[T] {
	q.GSIQueryBuilder.WithLimit(limit)
	return q
}

// WithFilter adds a filter expression
func (q *TimeRangeQueryBuilder[T]) WithFilter(expression string, values map[string]types.AttributeValue) *TimeRangeQueryBuilder[T] {
	q.GSIQueryBuilder.WithFilter(expression, values)
	return q
}

// StreamByTime streams results ordered by time with automatic pagination
func (q *TimeRangeQueryBuilder[T]) StreamByTime(ctx context.Context, opts ...storagemodels.StreamOption) <-chan storagemodels.StreamResult[T] {
	// Ensure we have time ordering
	if q.params.ScanIndexForward == nil {
		q.Latest() // Default to newest first
	}
	
	return q.Stream(ctx, opts...)
}

// TimeWindowIterator provides an iterator for processing time-based windows
type TimeWindowIterator[T any] struct {
	store       *DynamodbDataStore[T]
	partitionKey string
	windowSize  time.Duration
	startTime   time.Time
	endTime     time.Time
	current     time.Time
}

// QueryTimeWindows creates an iterator for querying in time windows
func (d *DynamodbDataStore[T]) QueryTimeWindows(partitionKey string, start, end time.Time, windowSize time.Duration) *TimeWindowIterator[T] {
	return &TimeWindowIterator[T]{
		store:        d,
		partitionKey: partitionKey,
		windowSize:   windowSize,
		startTime:    start,
		endTime:      end,
		current:      start,
	}
}

// Next returns the next window of results
func (it *TimeWindowIterator[T]) Next(ctx context.Context) ([]T, bool, error) {
	if it.current.After(it.endTime) || it.current.Equal(it.endTime) {
		return nil, false, nil // No more windows
	}
	
	windowEnd := it.current.Add(it.windowSize)
	if windowEnd.After(it.endTime) {
		windowEnd = it.endTime
	}
	
	// Query this time window
	results, err := it.store.QueryByTimeRange(it.partitionKey).
		Between(it.current, windowEnd).
		Execute(ctx)
	
	if err != nil {
		return nil, false, fmt.Errorf("failed to query time window: %w", err)
	}
	
	// Move to next window
	it.current = windowEnd
	
	hasMore := it.current.Before(it.endTime)
	return results, hasMore, nil
}

// Common time-based query patterns as convenience methods

// QueryLatestItems queries the N most recent items
func (d *DynamodbDataStore[T]) QueryLatestItems(ctx context.Context, partitionKey string, limit int32) ([]T, error) {
	return d.QueryByTimeRange(partitionKey).
		Latest().
		WithLimit(limit).
		Execute(ctx)
}

// QueryItemsSince queries all items created/updated since a timestamp
func (d *DynamodbDataStore[T]) QueryItemsSince(ctx context.Context, partitionKey string, since time.Time) ([]T, error) {
	return d.QueryByTimeRange(partitionKey).
		After(since).
		Latest().
		Execute(ctx)
}

// QueryItemsInDateRange queries items within a date range
func (d *DynamodbDataStore[T]) QueryItemsInDateRange(ctx context.Context, partitionKey string, start, end time.Time) ([]T, error) {
	return d.QueryByTimeRange(partitionKey).
		Between(start, end).
		Oldest(). // Chronological order
		Execute(ctx)
}

// StreamLatestItems streams items in reverse chronological order
func (d *DynamodbDataStore[T]) StreamLatestItems(ctx context.Context, partitionKey string, opts ...storagemodels.StreamOption) <-chan storagemodels.StreamResult[T] {
	return d.QueryByTimeRange(partitionKey).
		Latest().
		StreamByTime(ctx, opts...)
}

// TimeBasedPagination helps with time-based pagination
type TimeBasedPagination struct {
	PageSize      int32
	LastTimestamp *time.Time
	Direction     string // "forward" or "backward"
}

// QueryWithTimePagination queries with time-based pagination
func (d *DynamodbDataStore[T]) QueryWithTimePagination(ctx context.Context, partitionKey string, pagination TimeBasedPagination) ([]T, *time.Time, error) {
	builder := d.QueryByTimeRange(partitionKey).WithLimit(pagination.PageSize)
	
	if pagination.LastTimestamp != nil {
		if pagination.Direction == "backward" {
			builder.Before(*pagination.LastTimestamp)
		} else {
			builder.After(*pagination.LastTimestamp)
		}
	}
	
	if pagination.Direction == "backward" {
		builder.Latest()
	} else {
		builder.Oldest()
	}
	
	results, err := builder.Execute(ctx)
	if err != nil {
		return nil, nil, err
	}
	
	// Extract last timestamp from results for next page
	// This would need reflection or interface to get the timestamp field
	var lastTime *time.Time
	if len(results) > 0 {
		// Implementation would extract timestamp from last result
		// For now, returning nil
	}
	
	return results, lastTime, nil
}