/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package ddb

import (
	"context"
	"fmt"
	"strings"
	
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/suparena/entitystore/registry"
	"github.com/suparena/entitystore/storagemodels"
)

// GSIQueryBuilder provides a fluent interface for building GSI queries
type GSIQueryBuilder[T any] struct {
	store      *DynamodbDataStore[T]
	params     *storagemodels.QueryParams
	indexName  string
	pkValue    string
	skValue    string
	skOperator string // "=", "begins_with", ">", "<", ">=", "<="
	filters    []string
	filterVals map[string]types.AttributeValue
}

// QueryGSI creates a new GSI query builder
func (d *DynamodbDataStore[T]) QueryGSI() *GSIQueryBuilder[T] {
	return &GSIQueryBuilder[T]{
		store:      d,
		indexName:  "GSI1", // Default to GSI1
		filterVals: make(map[string]types.AttributeValue),
		params: &storagemodels.QueryParams{
			TableName:                 d.tableName,
			ExpressionAttributeValues: make(map[string]types.AttributeValue),
		},
	}
}

// WithPartitionKey sets the GSI partition key value
func (q *GSIQueryBuilder[T]) WithPartitionKey(value string) *GSIQueryBuilder[T] {
	q.pkValue = value
	return q
}

// WithSortKey sets the GSI sort key value with equals operator
func (q *GSIQueryBuilder[T]) WithSortKey(value string) *GSIQueryBuilder[T] {
	q.skValue = value
	q.skOperator = "="
	return q
}

// WithSortKeyPrefix sets the GSI sort key to use begins_with operator
func (q *GSIQueryBuilder[T]) WithSortKeyPrefix(prefix string) *GSIQueryBuilder[T] {
	q.skValue = prefix
	q.skOperator = "begins_with"
	return q
}

// WithSortKeyGreaterThan sets the GSI sort key to use > operator
func (q *GSIQueryBuilder[T]) WithSortKeyGreaterThan(value string) *GSIQueryBuilder[T] {
	q.skValue = value
	q.skOperator = ">"
	return q
}

// WithSortKeyLessThan sets the GSI sort key to use < operator
func (q *GSIQueryBuilder[T]) WithSortKeyLessThan(value string) *GSIQueryBuilder[T] {
	q.skValue = value
	q.skOperator = "<"
	return q
}

// WithSortKeyBetween sets the GSI sort key to use BETWEEN operator
func (q *GSIQueryBuilder[T]) WithSortKeyBetween(start, end string) *GSIQueryBuilder[T] {
	q.skValue = start
	q.skOperator = "BETWEEN"
	q.params.ExpressionAttributeValues[":sk2"] = &types.AttributeValueMemberS{Value: end}
	return q
}

// WithFilter adds a filter expression
func (q *GSIQueryBuilder[T]) WithFilter(expression string, values map[string]types.AttributeValue) *GSIQueryBuilder[T] {
	q.filters = append(q.filters, expression)
	for k, v := range values {
		q.filterVals[k] = v
	}
	return q
}

// WithLimit sets the query limit
func (q *GSIQueryBuilder[T]) WithLimit(limit int32) *GSIQueryBuilder[T] {
	q.params.Limit = aws.Int32(limit)
	return q
}

// Build constructs the final query parameters
func (q *GSIQueryBuilder[T]) Build() (*storagemodels.QueryParams, error) {
	// Validate required fields
	if q.pkValue == "" {
		return nil, fmt.Errorf("GSI partition key value is required")
	}
	
	// Get index map to build the actual key values
	indexMap, ok := registry.GetIndexMap[T]()
	if !ok {
		return nil, fmt.Errorf("no index map found for type %T", *new(T))
	}
	
	// Build key condition expression
	keyConditions := []string{"GSI1PK = :pk"}
	
	// Expand the partition key using the index map pattern
	gsi1PKPattern, ok := indexMap["GSI1PK"]
	if !ok {
		return nil, fmt.Errorf("GSI1PK not found in index map")
	}
	
	// Simple expansion - replace macro with value
	expandedPK := strings.ReplaceAll(gsi1PKPattern, "{", "")
	expandedPK = strings.ReplaceAll(expandedPK, "}", "")
	
	// If pattern has a prefix (e.g., "EMAIL#{Email}"), extract it
	if strings.Contains(gsi1PKPattern, "#") {
		parts := strings.Split(gsi1PKPattern, "#")
		if len(parts) > 0 {
			expandedPK = parts[0] + "#" + q.pkValue
		}
	} else {
		expandedPK = q.pkValue
	}
	
	q.params.ExpressionAttributeValues[":pk"] = &types.AttributeValueMemberS{Value: expandedPK}
	
	// Handle sort key if provided
	if q.skValue != "" {
		gsi1SKPattern, hasSK := indexMap["GSI1SK"]
		if hasSK {
			// Expand sort key
			expandedSK := q.skValue
			if strings.Contains(gsi1SKPattern, "#") {
				parts := strings.Split(gsi1SKPattern, "#")
				if len(parts) > 0 && !strings.Contains(expandedSK, "#") {
					expandedSK = parts[0] + "#" + q.skValue
				}
			}
			
			switch q.skOperator {
			case "=":
				keyConditions = append(keyConditions, "GSI1SK = :sk")
				q.params.ExpressionAttributeValues[":sk"] = &types.AttributeValueMemberS{Value: expandedSK}
			case "begins_with":
				keyConditions = append(keyConditions, "begins_with(GSI1SK, :sk)")
				q.params.ExpressionAttributeValues[":sk"] = &types.AttributeValueMemberS{Value: expandedSK}
			case ">":
				keyConditions = append(keyConditions, "GSI1SK > :sk")
				q.params.ExpressionAttributeValues[":sk"] = &types.AttributeValueMemberS{Value: expandedSK}
			case "<":
				keyConditions = append(keyConditions, "GSI1SK < :sk")
				q.params.ExpressionAttributeValues[":sk"] = &types.AttributeValueMemberS{Value: expandedSK}
			case ">=":
				keyConditions = append(keyConditions, "GSI1SK >= :sk")
				q.params.ExpressionAttributeValues[":sk"] = &types.AttributeValueMemberS{Value: expandedSK}
			case "<=":
				keyConditions = append(keyConditions, "GSI1SK <= :sk")
				q.params.ExpressionAttributeValues[":sk"] = &types.AttributeValueMemberS{Value: expandedSK}
			case "BETWEEN":
				keyConditions = append(keyConditions, "GSI1SK BETWEEN :sk AND :sk2")
				q.params.ExpressionAttributeValues[":sk"] = &types.AttributeValueMemberS{Value: expandedSK}
				// :sk2 should already be set in WithSortKeyBetween
			}
		}
	}
	
	// Set key condition expression
	q.params.KeyConditionExpression = strings.Join(keyConditions, " AND ")
	
	// Set index name
	q.params.IndexName = aws.String(q.indexName)
	
	// Add filter expressions
	if len(q.filters) > 0 {
		filterExpr := strings.Join(q.filters, " AND ")
		q.params.FilterExpression = aws.String(filterExpr)
		
		// Merge filter values
		for k, v := range q.filterVals {
			q.params.ExpressionAttributeValues[k] = v
		}
	}
	
	return q.params, nil
}

// Execute runs the query and returns results
func (q *GSIQueryBuilder[T]) Execute(ctx context.Context) ([]T, error) {
	params, err := q.Build()
	if err != nil {
		return nil, err
	}
	
	// Use the store's Query method
	results, err := q.store.Query(ctx, params)
	if err != nil {
		return nil, err
	}
	
	// Convert results to typed slice
	typedResults := make([]T, 0, len(results))
	for _, r := range results {
		if typed, ok := r.(T); ok {
			typedResults = append(typedResults, typed)
		} else if typed, ok := r.(*T); ok {
			typedResults = append(typedResults, *typed)
		}
	}
	
	return typedResults, nil
}

// ExecuteWithPagination runs the query and returns results with pagination token
func (q *GSIQueryBuilder[T]) ExecuteWithPagination(ctx context.Context, exclusiveStartKey map[string]types.AttributeValue) ([]T, map[string]types.AttributeValue, error) {
	params, err := q.Build()
	if err != nil {
		return nil, nil, err
	}
	
	if exclusiveStartKey != nil {
		params.ExclusiveStartKey = exclusiveStartKey
	}
	
	// We need to expose the LastEvaluatedKey from the query
	// This would require modifying the Query method to return it
	// For now, we'll use Execute and note this as a future enhancement
	results, err := q.Execute(ctx)
	return results, nil, err
}

// Stream executes the query as a stream
func (q *GSIQueryBuilder[T]) Stream(ctx context.Context, opts ...storagemodels.StreamOption) <-chan storagemodels.StreamResult[T] {
	params, err := q.Build()
	if err != nil {
		// Return error channel
		ch := make(chan storagemodels.StreamResult[T], 1)
		ch <- storagemodels.StreamResult[T]{
			Error: fmt.Errorf("failed to build query: %w", err),
		}
		close(ch)
		return ch
	}
	
	return q.store.Stream(ctx, params, opts...)
}

// Common GSI query patterns as convenience methods

// QueryByGSI1PK queries using only the GSI1 partition key
func (d *DynamodbDataStore[T]) QueryByGSI1PK(ctx context.Context, pkValue string) ([]T, error) {
	return d.QueryGSI().
		WithPartitionKey(pkValue).
		Execute(ctx)
}

// QueryByGSI1PKAndSKPrefix queries using GSI1 partition key and sort key prefix
func (d *DynamodbDataStore[T]) QueryByGSI1PKAndSKPrefix(ctx context.Context, pkValue, skPrefix string) ([]T, error) {
	return d.QueryGSI().
		WithPartitionKey(pkValue).
		WithSortKeyPrefix(skPrefix).
		Execute(ctx)
}

// QueryByGSI1PKWithFilter queries using GSI1 partition key with additional filters
func (d *DynamodbDataStore[T]) QueryByGSI1PKWithFilter(ctx context.Context, pkValue string, filterExpr string, filterValues map[string]types.AttributeValue) ([]T, error) {
	return d.QueryGSI().
		WithPartitionKey(pkValue).
		WithFilter(filterExpr, filterValues).
		Execute(ctx)
}