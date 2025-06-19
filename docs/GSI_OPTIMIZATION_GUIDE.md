# GSI Optimization Guide for EntityStore

## Overview

EntityStore supports a single Global Secondary Index (GSI1) to optimize query patterns while keeping operational costs low. This guide explains how to make the most of this single GSI through careful key design and query optimization.

## GSI Design Pattern

The default GSI structure:
- **GSI1PK**: Typically used for alternate access patterns (e.g., `EMAIL#{Email}`)
- **GSI1SK**: Used for filtering and sorting (e.g., `STATUS#{Status}#CREATED#{CreatedAt}`)

## Query Builder API

### Basic Usage

```go
// Simple query by GSI partition key
results, err := store.QueryByGSI1PK(ctx, "user@example.com")

// Query with sort key prefix
results, err := store.QueryByGSI1PKAndSKPrefix(ctx, "user@example.com", "STATUS#active")

// Complex query with builder
results, err := store.QueryGSI().
    WithPartitionKey("user@example.com").
    WithSortKeyPrefix("STATUS#active").
    WithFilter("Country = :country", filterValues).
    WithLimit(100).
    Execute(ctx)
```

### Advanced Query Patterns

#### 1. Range Queries

```go
// Query between status values
results, err := store.QueryGSI().
    WithPartitionKey("user@example.com").
    WithSortKeyBetween("STATUS#active", "STATUS#pending").
    Execute(ctx)

// Query with date ranges (if dates are in SK)
results, err := store.QueryGSI().
    WithPartitionKey("user@example.com").
    WithSortKeyBetween("DATE#2024-01-01", "DATE#2024-12-31").
    Execute(ctx)
```

#### 2. Filtering

```go
filterValues := map[string]types.AttributeValue{
    ":country": &types.AttributeValueMemberS{Value: "USA"},
    ":score":   &types.AttributeValueMemberN{Value: "90"},
}

results, err := store.QueryGSI().
    WithPartitionKey("user@example.com").
    WithFilter("Country = :country AND Score > :score", filterValues).
    Execute(ctx)
```

#### 3. Streaming Large Results

```go
opts := storagemodels.StreamOption{
    BufferSize: 100,
    MaxRetries: 3,
}

resultCh := store.QueryGSI().
    WithPartitionKey("user@example.com").
    Stream(ctx, opts)

for result := range resultCh {
    if result.Error != nil {
        log.Printf("Stream error: %v", result.Error)
        continue
    }
    // Process result.Item
}
```

## Key Design Best Practices

### 1. Hierarchical Sort Keys

Design sort keys to support multiple access patterns:

```yaml
# Good: Supports multiple query patterns
GSI1SK: "STATUS#{Status}#DATE#{CreatedAt}#ID#{ID}"

# Queries supported:
# - All items with specific status
# - Status within date range
# - Specific item by status and date
```

### 2. Prefix-Based Filtering

Use consistent prefixes for efficient queries:

```yaml
# Email-based access
GSI1PK: "EMAIL#{Email}"
GSI1SK: "TYPE#{Type}#DATE#{Date}"

# Organization-based access
GSI1PK: "ORG#{OrgId}"
GSI1SK: "USER#{UserId}#ROLE#{Role}"
```

### 3. Composite Keys for Multiple Attributes

Combine attributes in keys for compound queries:

```yaml
# Location and status queries
GSI1PK: "LOCATION#{Country}#{City}"
GSI1SK: "STATUS#{Status}#SCORE#{Score}"
```

## Common Query Patterns

### Pattern 1: User by Email

```yaml
x-dynamodb-indexmap:
  GSI1PK: "EMAIL#{Email}"
  GSI1SK: "PROFILE"
```

```go
// Find user by email
user, err := store.QueryByGSI1PK(ctx, "user@example.com")
```

### Pattern 2: Items by Status

```yaml
x-dynamodb-indexmap:
  GSI1PK: "STATUS#{Status}"
  GSI1SK: "CREATED#{CreatedAt}#ID#{ID}"
```

```go
// Find all active items, sorted by creation date
results, err := store.QueryByGSI1PK(ctx, "active")
```

### Pattern 3: Hierarchical Data

```yaml
x-dynamodb-indexmap:
  GSI1PK: "TENANT#{TenantId}"
  GSI1SK: "PROJECT#{ProjectId}#TASK#{TaskId}"
```

```go
// Find all projects for a tenant
projects, err := store.QueryGSI().
    WithPartitionKey("tenant123").
    WithSortKeyPrefix("PROJECT#").
    Execute(ctx)

// Find specific project tasks
tasks, err := store.QueryGSI().
    WithPartitionKey("tenant123").
    WithSortKeyPrefix("PROJECT#proj456#TASK#").
    Execute(ctx)
```

### Pattern 4: Time-Based Queries

```yaml
x-dynamodb-indexmap:
  GSI1PK: "TYPE#{Type}"
  GSI1SK: "TIME#{CreatedAt}#ID#{ID}"
```

#### Basic Time Queries
```go
// Find events in date range
events, err := store.QueryGSI().
    WithPartitionKey("event").
    WithSortKeyBetween("TIME#2024-01-01T00:00:00Z", "TIME#2024-01-31T23:59:59Z").
    Execute(ctx)
```

#### Advanced Time-Based Query API
```go
// Query last 24 hours of events (newest first)
events, err := store.QueryByTimeRange("events").
    InLastHours(24).
    Latest().
    Execute(ctx)

// Query this week's data
weeklyData, err := store.QueryByTimeRange("metrics").
    ThisWeek().
    WithLimit(100).
    Execute(ctx)

// Query between specific dates
historical, err := store.QueryByTimeRange("logs").
    Between(startDate, endDate).
    Oldest(). // Chronological order
    Execute(ctx)

// Stream latest items in real-time
for result := range store.StreamLatestItems(ctx, "notifications") {
    if result.Error != nil {
        continue
    }
    processNotification(result.Item)
}

// Query with time windows (e.g., daily aggregates)
iterator := store.QueryTimeWindows("sales", monthStart, monthEnd, 24*time.Hour)
for {
    dailySales, hasMore, err := iterator.Next(ctx)
    if err != nil || !hasMore {
        break
    }
    processDailySales(dailySales)
}
```

#### Time-Based Sort Key Design
```yaml
# Millisecond precision for high-frequency data
GSI1SK: "TIME#{UnixMilli}#ID#{ID}"

# Human-readable for debugging
GSI1SK: "TIME#{CreatedAt}#ID#{ID}" # Uses RFC3339

# Hierarchical time buckets
GSI1SK: "YEAR#{Year}#MONTH#{Month}#DAY#{Day}#TIME#{Time}"

# Status with time
GSI1SK: "STATUS#{Status}#TIME#{UpdatedAt}"
```

### Pattern 5: Optimized Time-Based Access

```yaml
# For activity feeds or timelines
Activity:
  x-dynamodb-indexmap:
    GSI1PK: "USER#{UserId}"
    GSI1SK: "TIME#{Timestamp}#ACTIVITY#{ActivityId}"

# For audit logs
AuditLog:
  x-dynamodb-indexmap:
    GSI1PK: "ENTITY#{EntityType}#{EntityId}"
    GSI1SK: "TIME#{Timestamp}#ACTION#{Action}"

# For metrics and analytics
Metric:
  x-dynamodb-indexmap:
    GSI1PK: "METRIC#{MetricType}"
    GSI1SK: "TIME#{Timestamp}#SOURCE#{SourceId}"
```

#### Best Practices for Time-Based Queries

1. **Use ISO 8601/RFC3339 Format**: Ensures lexicographic sorting
   ```go
   timestamp := time.Now().Format(time.RFC3339) // "2024-01-15T14:30:00Z"
   ```

2. **Include Unique ID**: Prevents collisions for same-timestamp items
   ```yaml
   GSI1SK: "TIME#{Timestamp}#ID#{UniqueId}"
   ```

3. **Optimize for Common Access Patterns**:
   ```go
   // Most recent first (default for feeds)
   store.QueryByTimeRange("user123").Latest().WithLimit(20)
   
   // Chronological order (for history/audit)
   store.QueryByTimeRange("audit").Oldest()
   ```

4. **Use Time Windows for Large Datasets**:
   ```go
   // Process in daily chunks to avoid timeouts
   iterator := store.QueryTimeWindows("logs", start, end, 24*time.Hour)
   ```

## Performance Optimization Tips

### 1. Use Projections Wisely

Although EntityStore doesn't directly expose projection configuration, design your entities to minimize data transfer:

```go
// Use filter expressions to reduce data transfer
results, err := store.QueryGSI().
    WithPartitionKey("user@example.com").
    WithFilter("attribute_exists(RequiredField)", nil).
    Execute(ctx)
```

### 2. Implement Pagination

For large result sets, use pagination:

```go
var allResults []Entity
var lastEvaluatedKey map[string]types.AttributeValue

for {
    params := store.QueryGSI().
        WithPartitionKey("user@example.com").
        WithLimit(25).
        Build()
    
    if lastEvaluatedKey != nil {
        params.ExclusiveStartKey = lastEvaluatedKey
    }
    
    results, err := store.Query(ctx, params)
    if err != nil {
        return err
    }
    
    allResults = append(allResults, results...)
    
    // Check if more results exist
    // Note: This requires access to LastEvaluatedKey from DynamoDB response
    if len(results) < 25 {
        break
    }
}
```

### 3. Use Sparse Indexes

Design GSI keys to be sparse (not all items have the indexed attribute):

```yaml
# Only items with email will be in GSI
GSI1PK: "EMAIL#{Email}"  # Email might be optional
```

## Query Cost Optimization

### 1. Minimize Filter Expressions

Filters are applied after data retrieval, increasing read costs:

```go
// Expensive: Retrieves all items, then filters
results, err := store.QueryGSI().
    WithPartitionKey("org123").
    WithFilter("Status = :status", filterValues).
    Execute(ctx)

// Better: Use sort key for filtering
// Design: GSI1SK = "STATUS#{Status}#..."
results, err := store.QueryGSI().
    WithPartitionKey("org123").
    WithSortKeyPrefix("STATUS#active").
    Execute(ctx)
```

### 2. Batch Operations

When possible, batch GSI queries:

```go
// Instead of multiple queries
for _, email := range emails {
    results, _ := store.QueryByGSI1PK(ctx, email)
    // Process results
}

// Consider redesigning to support batch access
// Or use concurrent queries with goroutines
```

### 3. Monitor Query Metrics

Track query performance:

```go
start := time.Now()
results, err := store.QueryGSI().
    WithPartitionKey("user@example.com").
    Execute(ctx)
duration := time.Since(start)

// Log slow queries
if duration > 100*time.Millisecond {
    log.Printf("Slow GSI query: %v", duration)
}
```

## Limitations and Workarounds

### Single GSI Limitations

With only one GSI, some query patterns require workarounds:

1. **Multiple Access Patterns**: Design composite keys carefully
2. **Complex Filtering**: May require client-side filtering
3. **Multiple Sort Orders**: Consider duplicating data with different sort keys

### Workaround Strategies

1. **Denormalization**: Store data in multiple formats
2. **Composite Keys**: Combine multiple attributes
3. **Application-Level Indexes**: Maintain lookup tables
4. **Scan Operations**: For infrequent queries (use sparingly)

## Example: E-commerce Order System

```yaml
# Order entity
Order:
  x-dynamodb-indexmap:
    PK: "ORDER#{OrderId}"
    SK: "ORDER#{OrderId}"
    GSI1PK: "CUSTOMER#{CustomerId}"
    GSI1SK: "STATUS#{Status}#DATE#{OrderDate}#ORDER#{OrderId}"

# Supported queries:
# 1. All orders for a customer
# 2. Customer orders by status
# 3. Customer orders in date range
# 4. Customer orders by status in date range
```

```go
// Customer's active orders
activeOrders, _ := store.QueryGSI().
    WithPartitionKey("customer123").
    WithSortKeyPrefix("STATUS#active").
    Execute(ctx)

// Customer's orders in January 2024
janOrders, _ := store.QueryGSI().
    WithPartitionKey("customer123").
    WithSortKeyBetween(
        "STATUS#delivered#DATE#2024-01-01",
        "STATUS#delivered#DATE#2024-01-31"
    ).
    Execute(ctx)
```

## Testing GSI Queries

Always test GSI queries with realistic data volumes:

```go
func TestGSIPerformance(t *testing.T) {
    // Create test data
    for i := 0; i < 1000; i++ {
        entity := Entity{
            ID:    fmt.Sprintf("test-%d", i),
            Email: "test@example.com",
            Status: []string{"active", "inactive", "pending"}[i%3],
        }
        store.Put(ctx, entity)
    }
    
    // Measure query performance
    start := time.Now()
    results, err := store.QueryByGSI1PK(ctx, "test@example.com")
    duration := time.Since(start)
    
    t.Logf("Query returned %d results in %v", len(results), duration)
    
    // Verify performance
    if duration > 500*time.Millisecond {
        t.Errorf("Query too slow: %v", duration)
    }
}
```

## Conclusion

Optimizing queries with a single GSI requires careful planning and design. By following these patterns and best practices, you can build efficient query patterns that minimize costs while maintaining good performance. Remember to:

1. Design composite keys that support multiple access patterns
2. Use sort key prefixes for efficient filtering
3. Minimize filter expressions by encoding filters in keys
4. Monitor and optimize query performance
5. Consider application-level solutions for complex query requirements