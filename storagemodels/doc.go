/*
Package storagemodels defines the data structures used throughout EntityStore.

Key Types:

QueryParams:
Parameters for querying the datastore:

	params := &QueryParams{
	    TableName:              "my-table",
	    KeyConditionExpression: "PK = :pk",
	    ExpressionAttributeValues: map[string]types.AttributeValue{
	        ":pk": &types.AttributeValueMemberS{Value: "USER#123"},
	    },
	    FilterExpression: aws.String("Status = :status"),
	    IndexName:        aws.String("GSI1"),
	    Limit:            aws.Int32(100),
	}

StreamResult:
Results from streaming operations with metadata:

	type StreamResult[T any] struct {
	    Item  T                               // The typed entity
	    Raw   map[string]types.AttributeValue // Raw DynamoDB attributes
	    Error error                           // Item-specific error, if any
	    Meta  StreamMeta                      // Metadata about this item
	}

StreamOptions:
Configuration for streaming behavior:

	opts := []StreamOption{
	    WithBufferSize(100),
	    WithPageSize(25),
	    WithMaxRetries(3),
	    WithProgressHandler(progressFunc),
	}

These types provide a consistent interface across different storage implementations.
*/
package storagemodels