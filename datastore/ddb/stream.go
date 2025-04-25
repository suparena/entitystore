/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package ddb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/suparena/entitystore/registry"
	"github.com/suparena/entitystore/storagemodels"
)

// Stream performs a streaming query against DynamoDB. It sends each unmarshaled item through a channel as soon as it’s available.
// It leverages the injected EntityType attribute (added when persisting objects) to look up the proper unmarshal function
// from the type registry.
func (d *DynamodbDataStore[T]) Stream(ctx context.Context, params *storagemodels.StreamQueryParams) (<-chan storagemodels.StreamItem, <-chan error) {
	itemCh := make(chan storagemodels.StreamItem)
	errCh := make(chan error, 1)

	go func() {
		defer close(itemCh)
		defer close(errCh)

		input := &dynamodb.QueryInput{
			TableName:                 &params.TableName,
			KeyConditionExpression:    &params.KeyConditionExpression,
			ExpressionAttributeValues: params.ExpressionAttributeValues,
			FilterExpression:          params.FilterExpression,
			IndexName:                 params.IndexName,
			Limit:                     params.Limit,
		}

		var lastEvaluatedKey map[string]types.AttributeValue
		for {
			if lastEvaluatedKey != nil {
				input.ExclusiveStartKey = lastEvaluatedKey
			}

			out, err := d.client.Query(ctx, input)
			if err != nil {
				errCh <- fmt.Errorf("query error: %w", err)
				return
			}

			for _, item := range out.Items {
				// Retrieve the EntityType attribute.
				var entityType string
				if attr, ok := item["EntityType"]; ok {
					if err := attributevalue.Unmarshal(attr, &entityType); err != nil {
						errCh <- fmt.Errorf("failed to unmarshal EntityType: %w", err)
						return
					}
					delete(item, "EntityType")
				} else {
					errCh <- fmt.Errorf("missing EntityType attribute in item")
					return
				}

				// Look up the unmarshal function from the type registry.
				unmarshalFn, err := registry.GetUnmarshalFunc(entityType)
				if err != nil {
					// Fallback: unmarshal into a generic map.
					var generic map[string]interface{}
					if err := attributevalue.UnmarshalMap(item, &generic); err != nil {
						errCh <- fmt.Errorf("failed to unmarshal generic item: %w", err)
						return
					}
					select {
					case <-ctx.Done():
						return
					case itemCh <- storagemodels.StreamItem{Item: generic, Raw: item}:
					}
					continue
				}

				// Use the unmarshal function to convert the raw item to a typed object.
				obj, err := unmarshalFn(item)
				if err != nil {
					errCh <- fmt.Errorf("failed to unmarshal item for EntityType %q: %w", entityType, err)
					return
				}

				select {
				case <-ctx.Done():
					return
				case itemCh <- storagemodels.StreamItem{Item: obj, Raw: item}:
				}
			}

			// Check for pagination.
			if out.LastEvaluatedKey == nil || len(out.LastEvaluatedKey) == 0 {
				break
			}
			lastEvaluatedKey = out.LastEvaluatedKey
		}
	}()

	return itemCh, errCh
}
