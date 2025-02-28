/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package storagemodels

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// StreamItem wraps a processed item along with its raw DynamoDB attributes.
type StreamItem struct {
	// Item is the unmarshaled object, which could be a pointer to a concrete type or a generic map.
	Item interface{}
	// Raw is the original DynamoDB item map.
	Raw map[string]types.AttributeValue
}

// StreamQueryParams defines parameters for a DynamoDB query operation.
type StreamQueryParams struct {
	// TableName is the DynamoDB table name.
	TableName string
	// KeyConditionExpression is the primary condition for the query.
	KeyConditionExpression string
	// FilterExpression is an optional filter expression.
	FilterExpression *string
	// ExpressionAttributeValues contains the values for expression placeholders.
	ExpressionAttributeValues map[string]types.AttributeValue
	// IndexName is optional if you wish to query a secondary index.
	IndexName *string
	// Limit defines an optional limit per query page.
	Limit *int32
}

// QueryParams defines parameters for a DynamoDB Query operation.
type QueryParams struct {
	// TableName is the DynamoDB table name.
	TableName string
	// KeyConditionExpression is the primary condition for the query.
	KeyConditionExpression string
	// FilterExpression is an optional filter expression.
	FilterExpression *string
	// ExpressionAttributeValues contains the values for the expression placeholders.
	ExpressionAttributeValues map[string]types.AttributeValue
	// IndexName is optional if you wish to query a secondary index.
	IndexName *string
	// Limit defines an optional limit per query page.
	Limit *int32
}
