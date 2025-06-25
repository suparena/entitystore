# EntityStore v0.2.0 Query Method Bug Report

## Issue Summary
The `Query` method in `DynamodbDataStore` is not setting the `TableName` field in the AWS SDK DynamoDB query parameters, resulting in a validation error when executing queries.

## Error Details
```
operation error DynamoDB: Query, https response error StatusCode: 400, 
RequestID: MO6DDRA53BPAPCRSVJFT0NTFFJVV4KQNSO5AEMVJF66Q9ASUAAJG, 
api error ValidationException: 1 validation error detected: 
Value '' at 'tableName' failed to satisfy constraint: Member must have length greater than or equal to 1
```

## Debug Information

### Datastore Initialization
The datastore is properly initialized with the table name:
```go
feedStore, err := ddb.NewDynamodbDataStore[coreLivestreaming.Feed](
    awsAccessKey, 
    awsSecretKey, 
    region, 
    "suparena"  // Table name is passed here
)
```

### Query Execution
When calling the Query method:
```go
queryParams := &storagemodels.QueryParams{
    IndexName: &indexName,
    KeyConditionExpression: "GSI1PK = :gsi1pk",
    ExpressionAttributeValues: map[string]types.AttributeValue{
        ":gsi1pk": gsi1pkValue,
    },
    Limit: &limit,
}
results, err := feedStore.Query(ctx, queryParams)
```

Debug output shows:
```
DEBUG: feedStore type: *ddb.DynamodbDataStore[...], value: &{client:0xc0005f2000 tableName:suparena}
DEBUG: About to query with params: &{TableName: KeyConditionExpression:GSI1PK = :gsi1pk ...}
```

Note that:
- The feedStore has `tableName:"suparena"` 
- But the QueryParams has `TableName:` (empty)

## Root Cause
The `Query` method in `DynamodbDataStore` is not using the datastore's internal `tableName` field when building the AWS SDK DynamoDB query input. The `TableName` field in the actual DynamoDB query request is empty, causing the validation error.

## Expected Behavior
The `Query` method should:
1. Take the `tableName` field from the `DynamodbDataStore` instance
2. Set it in the AWS SDK's `dynamodb.QueryInput.TableName` field before executing the query

## Suggested Fix
In the `Query` method implementation, ensure the table name is set:

```go
func (d *DynamodbDataStore[T]) Query(ctx context.Context, params *storagemodels.QueryParams) ([]interface{}, error) {
    queryInput := &dynamodb.QueryInput{
        TableName: aws.String(d.tableName), // This line is likely missing or incorrect
        IndexName: params.IndexName,
        KeyConditionExpression: aws.String(params.KeyConditionExpression),
        ExpressionAttributeValues: params.ExpressionAttributeValues,
        Limit: params.Limit,
    }
    
    // ... rest of the query execution
}
```

## Affected Methods
All methods that use GSI (Global Secondary Index) queries are affected:
- Queries using GSI1, GSI2, GSI3 indexes
- Any Query operation that doesn't explicitly set TableName in params

## Test Case
To reproduce:
1. Create a DynamodbDataStore with a valid table name
2. Call Query with GSI parameters
3. Observe the validation error about empty tableName

## Environment
- EntityStore version: v0.2.0
- AWS SDK: aws-sdk-go-v2
- Go version: 1.23.1

## Workaround
Currently working around this by returning empty results instead of executing queries, but this is not a sustainable solution.

## Impact
This bug prevents any GSI-based queries from working, which is critical for applications that rely on secondary indexes for data retrieval patterns.