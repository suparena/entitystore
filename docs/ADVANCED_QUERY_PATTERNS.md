# Advanced Query Patterns for EntityStore

## Overview

This document outlines sophisticated query patterns that could be implemented in EntityStore to provide more advanced DynamoDB capabilities.

## Implemented Patterns ✅

### 1. GSI Query Optimization
- Fluent query builder for GSI queries
- Convenience methods for common patterns
- Sort key operations (prefix, range, etc.)
- Filter expressions
- Streaming support for GSI queries

### 2. Time-Based Query Patterns
- Specialized `TimeRangeQueryBuilder` for time-based queries
- Convenience methods: `InLastHours`, `InLastDays`, `Today`, `ThisWeek`, `ThisMonth`
- Time-based sorting: `Latest()` and `Oldest()`
- Time window iterator for processing large date ranges
- Support for `ScanIndexForward` to control sort order

## Proposed Advanced Patterns 🚀

### 1. Parallel Query Execution

#### Pattern: Parallel GSI Queries
```go
// Query multiple GSIs in parallel
results, err := store.ParallelQuery().
    AddQuery("email", store.QueryGSI().WithPartitionKey(email)).
    AddQuery("phone", store.QueryGSI2().WithPartitionKey(phone)).
    Execute(ctx)

// Access results
emailResults := results["email"]
phoneResults := results["phone"]
```

#### Pattern: Scatter-Gather
```go
// Split query across partitions
results, err := store.ScatterGatherQuery().
    WithPartitions(["USER#A*", "USER#B*", "USER#C*"]).
    WithParallelism(3).
    Execute(ctx)
```

### 2. Scan Operations

#### Basic Scan
```go
// Scan with filter
results, err := store.Scan().
    WithFilter("attribute_exists(Email)").
    WithLimit(100).
    Execute(ctx)
```

#### Parallel Scan
```go
// Parallel scan for faster processing
results, err := store.ParallelScan().
    WithSegments(4).
    WithFilter("Status = :status", filterValues).
    Stream(ctx)
```

### 3. Query Projections

#### Attribute Projection
```go
// Only fetch specific attributes
results, err := store.Query().
    WithProjection("ID", "Name", "Email").
    WithPartitionKey("USER#123").
    Execute(ctx)
```

#### Nested Projections
```go
// Project nested attributes
results, err := store.Query().
    WithProjection("ID", "Profile.Name", "Settings.Theme").
    Execute(ctx)
```

### 4. Transaction Support

#### Transactional Writes
```go
// Atomic multi-item transaction
err := store.TransactWrite().
    Put(user1).
    Update(user2, "SET Credits = Credits + :val", values).
    Delete(user3).
    ConditionCheck(user4, "attribute_exists(ID)").
    Execute(ctx)
```

#### Transactional Reads
```go
// Consistent multi-item read
results, err := store.TransactGet().
    Get("User", "123").
    Get("Order", "456").
    Get("Product", "789").
    Execute(ctx)
```

### 5. Batch Operations

#### BatchGet with Retries
```go
// Batch get with automatic retry for unprocessed items
results, err := store.BatchGetWithRetry().
    AddKeys("123", "456", "789").
    WithMaxRetries(3).
    Execute(ctx)
```

#### BatchWrite with Chunking
```go
// Automatically chunk large batches
err := store.BatchWriteChunked().
    PutItems(items...). // Can be > 25 items
    WithChunkSize(25).
    Execute(ctx)
```

### 6. Advanced Query Features

#### Count Queries
```go
// Get count without retrieving items
count, err := store.QueryCount().
    WithPartitionKey("USER#123").
    WithFilter("Status = :status", filterValues).
    Execute(ctx)
```

#### Reverse Order Queries
```go
// Query in descending order
results, err := store.Query().
    WithPartitionKey("USER#123").
    WithSortOrder(Descending).
    Execute(ctx)
```

### 7. Aggregate Query Patterns

#### Client-Side Aggregation
```go
// Sum values during streaming
sum, err := store.StreamAggregate().
    WithPartitionKey("ORDERS#2024").
    Sum("OrderTotal").
    Execute(ctx)

// Group by attribute
grouped, err := store.StreamAggregate().
    WithPartitionKey("PRODUCTS").
    GroupBy("Category").
    Count().
    Execute(ctx)
```

### 8. Complex Filter Expressions

#### Advanced Filters
```go
// Complex boolean logic
results, err := store.Query().
    WithPartitionKey("USER#123").
    WithComplexFilter(
        And(
            Or(
                Equals("Status", "active"),
                Equals("Status", "pending")
            ),
            GreaterThan("Score", 100),
            Contains("Tags", "premium")
        )
    ).
    Execute(ctx)
```

### 9. Query Optimization

#### Query Planner
```go
// Automatically choose optimal index
results, err := store.SmartQuery().
    Where("Email", "=", email).
    Where("Status", "=", "active").
    Execute(ctx) // Automatically uses best index
```

#### Query Metrics
```go
// Track query performance
results, metrics, err := store.QueryWithMetrics().
    WithPartitionKey("USER#123").
    Execute(ctx)

fmt.Printf("RCUs consumed: %d, Latency: %v\n", 
    metrics.ConsumedRCU, metrics.Latency)
```

### 10. Composite Query Patterns

#### Union Queries
```go
// Combine results from multiple queries
results, err := store.UnionQuery().
    Add(store.Query().WithPartitionKey("USER#123")).
    Add(store.QueryGSI().WithPartitionKey("TEAM#456")).
    Execute(ctx)
```

#### Intersection Queries
```go
// Find common items across queries
results, err := store.IntersectQuery().
    Add(store.Query().WithFilter("Status = :active")).
    Add(store.Query().WithFilter("Score > :threshold")).
    Execute(ctx)
```

## Implementation Priority

### High Priority (Core Functionality)
1. **Scan Operations** - Essential for many use cases
2. **Batch Operations** - Performance critical
3. **Query Projections** - Cost optimization
4. **Transaction Support** - Data consistency

### Medium Priority (Performance)
5. **Parallel Query Execution** - Scale optimization
6. **Count Queries** - Common requirement
7. **Advanced Filters** - Flexibility

### Low Priority (Nice to Have)
8. **Aggregate Patterns** - Can be done client-side
9. **Query Planner** - Advanced optimization
10. **Composite Queries** - Special use cases

## Implementation Approach

### 1. Extend DataStore Interface
```go
type DataStore[T any] interface {
    // Existing methods...
    
    // New scan methods
    Scan(ctx context.Context, params *ScanParams) ([]T, error)
    ParallelScan(ctx context.Context, params *ParallelScanParams) <-chan StreamResult[T]
    
    // New batch methods
    BatchGetWithRetry(ctx context.Context, keys []interface{}) ([]T, error)
    BatchWriteChunked(ctx context.Context, items []T) error
    
    // New transaction methods
    TransactWrite() TransactionWriter[T]
    TransactGet() TransactionReader[T]
}
```

### 2. Create Builder Pattern APIs
```go
// Scan builder
type ScanBuilder[T any] struct {
    store  *DynamodbDataStore[T]
    params *ScanParams
}

// Transaction builder
type TransactionBuilder[T any] struct {
    store       *DynamodbDataStore[T]
    operations  []TransactOperation
}
```

### 3. Add Performance Monitoring
```go
type QueryMetrics struct {
    ConsumedRCU   float64
    Latency       time.Duration
    ItemsScanned  int
    ItemsReturned int
}
```

## Testing Strategy

### Unit Tests
- Mock DynamoDB client for all new operations
- Test query builders and parameter validation
- Test retry logic and error handling

### Integration Tests
- Use DynamoDB Local for realistic testing
- Test transaction rollbacks
- Test parallel operations
- Benchmark performance improvements

### Performance Tests
```go
func BenchmarkParallelScan(b *testing.B) {
    // Compare sequential vs parallel scan
}

func BenchmarkBatchOperations(b *testing.B) {
    // Compare individual vs batch operations
}
```

## Migration Considerations

### Backward Compatibility
- All new features should be additive
- Existing APIs must not change
- Use builder pattern for new complex operations

### Gradual Adoption
```go
// Old way still works
results, err := store.Query(ctx, params)

// New way is optional
results, err := store.QueryBuilder().
    WithProjection("ID", "Name").
    WithConsistentRead().
    Execute(ctx)
```

## Conclusion

These advanced query patterns would significantly enhance EntityStore's capabilities, making it suitable for complex DynamoDB use cases while maintaining its simple, type-safe API. The implementation should be incremental, starting with the most commonly needed patterns like Scan and Batch operations.