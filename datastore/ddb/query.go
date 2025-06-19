/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package ddb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/suparena/entitystore/registry"
	"github.com/suparena/entitystore/storagemodels"
)

// Query performs a query against the DynamoDB table using the provided parameters.
// It uses the injected EntityType attribute (added at persist time) to select the correct
// unmarshal function from the type registry so that each item is unmarshaled to its proper type.
func (d *DynamodbDataStore[T]) Query(ctx context.Context, params *storagemodels.QueryParams) ([]interface{}, error) {
	input := &dynamodb.QueryInput{
		TableName:                 &params.TableName,
		KeyConditionExpression:    &params.KeyConditionExpression,
		ExpressionAttributeValues: params.ExpressionAttributeValues,
		FilterExpression:          params.FilterExpression,
		IndexName:                 params.IndexName,
		Limit:                     params.Limit,
		ScanIndexForward:          params.ScanIndexForward,
	}
	out, err := d.client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	var results []interface{}
	for _, item := range out.Items {
		// Retrieve the EntityType attribute.
		var entityType string
		if attr, ok := item["EntityType"]; ok {
			if err := attributevalue.Unmarshal(attr, &entityType); err != nil {
				return nil, fmt.Errorf("failed to unmarshal EntityType: %w", err)
			}
		} else {
			return nil, fmt.Errorf("missing EntityType attribute in item")
		}

		// Look up the unmarshal function from the type registry.
		unmarshalFn, err := registry.GetUnmarshalFunc(entityType)
		if err != nil {
			// Fallback: if no function is registered, unmarshal into a generic map.
			var generic map[string]interface{}
			if err := attributevalue.UnmarshalMap(item, &generic); err != nil {
				return nil, fmt.Errorf("failed to unmarshal generic item: %w", err)
			}
			results = append(results, generic)
			continue
		}

		// Use the unmarshal function to convert the raw item to a typed object.
		obj, err := unmarshalFn(item)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal item for EntityType %q: %w", entityType, err)
		}
		results = append(results, obj)
	}

	return results, nil
}
