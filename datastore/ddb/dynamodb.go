/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package ddb

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/suparena/entitystore/registry"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	sdk "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamodbDataStore implements storage.DataStore[T] by using AWS DynamoDB as the underlying data store.
type DynamodbDataStore[T any] struct {
	client    *sdk.Client
	tableName string
}

var macroPattern = regexp.MustCompile(`{([^}]+)}`)

func expandMacros(indexMap map[string]string, keysInput any) (map[string]string, error) {
	// Convert keysInput to a map of attribute values
	av, err := attributevalue.MarshalMap(keysInput)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal keysInput: %w", err)
	}

	res := make(map[string]string, len(indexMap))

	for fieldName, template := range indexMap {
		expanded := macroPattern.ReplaceAllStringFunc(template, func(macro string) string {
			// macro is something like "{ID}"
			key := strings.Trim(macro, "{}")

			val, ok := av[key]
			if !ok {
				return ""
			}

			// Convert 'val' (types.AttributeValue) into a string.
			switch tv := val.(type) {
			case *types.AttributeValueMemberS:
				// e.g. S="abc123"
				return tv.Value

			case *types.AttributeValueMemberN:
				// e.g. N="42"
				return tv.Value

			case *types.AttributeValueMemberBOOL:
				// e.g. BOOL=true
				return fmt.Sprintf("%v", tv.Value)

			case *types.AttributeValueMemberNULL:
				// e.g. NULL=true
				return ""

			case *types.AttributeValueMemberB:
				// Binary data in tv.Value
				// You might base64-encode or return empty
				return ""

			case *types.AttributeValueMemberBS,
				*types.AttributeValueMemberNS,
				*types.AttributeValueMemberSS:
				// sets of strings/numbers/binaries
				// Typically you’d convert to CSV or something
				return ""

			default:
				// fallback if an unknown type
				return ""
			}
		})
		res[fieldName] = expanded
	}

	return res, nil
}

// NewDynamoDBClient initializes a DynamoDB client using AWS credentials.
func NewDynamoDBClient(awsAccessKey, awsSecretKey, awsRegion, tableName string) (*sdk.Client, error) {
	// Load the custom AWS configuration using static credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(awsRegion),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(awsAccessKey, awsSecretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// Create a DynamoDB client
	client := sdk.NewFromConfig(cfg)

	fmt.Printf("DynamoDB client initialized for table: %s in region: %s\n", tableName, awsRegion)
	return client, nil
}

// NewDynamodbDataStore constructs a new DynamodbDataStore for type T.
func NewDynamodbDataStore[T any](awsAccessKey, awsSecretKey, awsRegion, awsDDBTableName string) (*DynamodbDataStore[T], error) {
	// Create a new DynamoDB client
	client, err := NewDynamoDBClient(awsAccessKey, awsSecretKey, awsRegion, awsDDBTableName)
	if err != nil {
		return nil, fmt.Errorf("failed to create DynamoDB client: %w", err)
	}

	return &DynamodbDataStore[T]{
		client:    client,
		tableName: awsDDBTableName,
	}, nil
}

// GetOne retrieves a single item from DynamoDB using a string key.
// It returns a pointer to the item of type T, or nil if no item is found.
func (d *DynamodbDataStore[T]) GetOne(ctx context.Context, key string) (*T, error) {
	indexMap, ok := registry.GetIndexMap[T]()
	if !ok {
		return nil, errors.New("no index map found for entity type")
	}

	// Expand the string key using the index map configuration.
	expanded, err := expandStringKey(indexMap, key)
	if err != nil {
		return nil, fmt.Errorf("failed to expand string key: %w", err)
	}

	// Build the DynamoDB key.
	keyMap, err := buildKeyFromExpanded(expanded)
	if err != nil {
		return nil, fmt.Errorf("failed to build key: %w", err)
	}

	// Perform the GetItem call.
	out, err := d.client.GetItem(ctx, &sdk.GetItemInput{
		TableName: &d.tableName,
		Key:       keyMap,
	})
	if err != nil {
		return nil, fmt.Errorf("GetItem error: %w", err)
	}
	if out.Item == nil {
		// Not found: return nil, nil
		return nil, nil
	}

	// Create a new instance of T and unmarshal the item into it.
	result := new(T)
	if err := attributevalue.UnmarshalMap(out.Item, result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal item: %w", err)
	}
	return result, nil
}

// queryOne is a helper used by GetOne() when we don't have a full PK+SK to do GetItem.
// We can do a small Query. If you store PK1, SK1, etc. for a GSI, you can detect that
// here and set up QueryInput accordingly.
func (d *DynamodbDataStore[T]) queryOne(ctx context.Context, expanded map[string]string) ([]map[string]types.AttributeValue, error) {
	// Check if we have PK or PK1, etc. For example:
	pk, ok := expanded["PK"]
	if !ok || pk == "" {
		// For a real design, you might handle GSI or return an error
		return nil, errors.New("no PK found in indexMap for partial key lookup")
	}
	keyCond := "PK = :pkVal"
	exprVals := map[string]types.AttributeValue{
		":pkVal": &types.AttributeValueMemberS{Value: pk},
	}
	if sk, skOK := expanded["SK"]; skOK && sk != "" {
		// If we want an equality condition on SK, we do:
		keyCond += " AND SK = :skVal"
		exprVals[":skVal"] = &types.AttributeValueMemberS{Value: sk}
	}

	out, err := d.client.Query(ctx, &sdk.QueryInput{
		TableName:                 &d.tableName,
		KeyConditionExpression:    &keyCond,
		ExpressionAttributeValues: exprVals,
		Limit:                     aws.Int32(1), // only need one item
	})
	if err != nil {
		return nil, fmt.Errorf("queryOne - Query error: %w", err)
	}
	return out.Items, nil
}

// Put stores the given 'entity' in the underlying data store using macros in 'indexMap'
// to populate partition/sort keys (and possibly GSIs).
func (d *DynamodbDataStore[T]) Put(ctx context.Context, entity T) error {
	indexMap, ok := registry.GetIndexMap[T]()
	if !ok {
		return errors.New("no index map found for entity type")
	}

	av, err := attributevalue.MarshalMap(entity)
	if err != nil {
		return fmt.Errorf("failed to marshal entity: %w", err)
	}

	// Expand macros using the entity itself (assuming the entity has ID, etc.)
	expanded, err := expandMacros(indexMap, entity)
	if err != nil {
		return err
	}

	// Insert the expanded fields as PK, SK, etc.
	for k, v := range expanded {
		av[k] = &types.AttributeValueMemberS{Value: v}
	}

	_, err = d.client.PutItem(ctx, &sdk.PutItemInput{
		TableName: &d.tableName,
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("PutItem failed: %w", err)
	}
	return nil
}

// Delete removes an item from DynamoDB using a string key.
func (d *DynamodbDataStore[T]) Delete(ctx context.Context, key string) error {
	indexMap, ok := registry.GetIndexMap[T]()
	if !ok {
		return errors.New("no index map found for entity type")
	}

	// Expand the string key.
	expanded, err := expandStringKey(indexMap, key)
	if err != nil {
		return fmt.Errorf("failed to expand string key: %w", err)
	}

	// Build the DynamoDB key.
	keyMap, err := buildKeyFromExpanded(expanded)
	if err != nil {
		return fmt.Errorf("failed to build key for Delete: %w", err)
	}

	// Call DeleteItem.
	_, err = d.client.DeleteItem(ctx, &sdk.DeleteItemInput{
		TableName: &d.tableName,
		Key:       keyMap,
	})
	if err != nil {
		var cfe *types.ConditionalCheckFailedException
		if errors.As(err, &cfe) {
			return fmt.Errorf("delete condition failed: %w", err)
		}
		return fmt.Errorf("failed to delete item in DynamoDB: %w", err)
	}
	return nil
}

func (d *DynamodbDataStore[T]) getKey(keyInput any, indexMap map[string]string) (map[string]types.AttributeValue, error) {
	expanded, err := expandMacros(indexMap, keyInput)
	if err != nil {
		return nil, err
	}

	// Check for a single object key scenario.
	if key, ok, err := buildSingleKey(expanded); err != nil {
		return nil, err
	} else if ok {
		return key, nil
	}

	// Otherwise, fall back to the standard approach.
	pk, hasPK := expanded["PK"]
	sk, hasSK := expanded["SK"]

	if hasPK && hasSK && pk != "" && sk != "" {
		return map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		}, nil
	}

	return nil, errors.New("missing PK or SK in expanded indexMap")
}

// buildUpdateExpression transforms a map of field->value into:
//   - an "update expression" (e.g., "SET #f1 = :v1, #f2 = :v2")
//   - a corresponding map of expression attribute names
//   - a corresponding map of expression attribute values
func buildUpdateExpression(updates map[string]interface{}) (string,
	map[string]string,
	map[string]types.AttributeValue,
	error) {

	if len(updates) == 0 {
		return "", nil, nil, errors.New("no updates provided")
	}

	setClauses := make([]string, 0, len(updates))
	exprAttrNames := make(map[string]string)
	exprAttrValues := make(map[string]types.AttributeValue)

	i := 0
	for field, val := range updates {
		placeholderName := fmt.Sprintf("#f%d", i)
		placeholderValue := fmt.Sprintf(":v%d", i)

		setClauses = append(setClauses, fmt.Sprintf("%s = %s", placeholderName, placeholderValue))
		exprAttrNames[placeholderName] = field

		// Convert val -> AttributeValue; this is a naive approach for demonstration.
		// In real code, handle various types (string, number, bool, etc.).
		// We'll assume everything is string for simplicity here:
		switch typedVal := val.(type) {
		case string:
			exprAttrValues[placeholderValue] = &types.AttributeValueMemberS{Value: typedVal}
		case int, int64, float64:
			// Convert numeric to string for AttributeValueMemberN
			exprAttrValues[placeholderValue] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%v", typedVal)}
		default:
			// Could marshal to JSON or handle other data types
			return "", nil, nil, fmt.Errorf("unhandled update value type for field '%s'", field)
		}

		i++
	}

	updateExpr := "SET " + joinClauses(setClauses)
	return updateExpr, exprAttrNames, exprAttrValues, nil
}

// joinClauses is a tiny helper. You could just use strings.Join(setClauses, ", ") directly,
// but it's shown as a separate function for clarity.
func joinClauses(clauses []string) string {
	joined := ""
	for i, c := range clauses {
		if i > 0 {
			joined += ", "
		}
		joined += c
	}
	return joined
}

func (d *DynamodbDataStore[T]) UpdateWithCondition(ctx context.Context, keyInput any, updates map[string]interface{}, condition string) error {
	indexMap, ok := registry.GetIndexMap[T]()
	if !ok {
		return errors.New("no index map found for entity type")
	}

	key, err := d.getKey(keyInput, indexMap)
	if err != nil {
		return fmt.Errorf("failed to build key: %w", err)
	}

	updateExpr, exprAttrNames, exprAttrValues, err := buildUpdateExpression(updates)
	if err != nil {
		return fmt.Errorf("failed to build update expression: %w", err)
	}

	input := &sdk.UpdateItemInput{
		TableName:                 &d.tableName,
		Key:                       key,
		UpdateExpression:          &updateExpr,
		ExpressionAttributeNames:  exprAttrNames,
		ExpressionAttributeValues: exprAttrValues,
		ConditionExpression:       &condition,
		ReturnValues:              types.ReturnValueAllNew, // or ALL_OLD, NONE, etc.
	}

	_, err = d.client.UpdateItem(ctx, input)
	if err != nil {
		// If the condition fails, DynamoDB returns a ConditionalCheckFailedException
		var cfe *types.ConditionalCheckFailedException
		if errors.As(err, &cfe) {
			return fmt.Errorf("condition failed: %w", err)
		}
		// Other possible errors: ProvisionedThroughputExceeded, etc.
		return fmt.Errorf("UpdateWithCondition failed: %w", err)
	}

	return nil
}

// buildKeyFromExpanded builds a DynamoDB key from the expanded index map.
// It assumes that the expanded map has valid non-empty values for "PK" and "SK".
func buildKeyFromExpanded(expanded map[string]string) (map[string]types.AttributeValue, error) {
	pk, okPK := expanded["PK"]
	sk, okSK := expanded["SK"]

	if !okPK || !okSK || pk == "" || sk == "" {
		return nil, fmt.Errorf("expanded index map missing valid PK or SK")
	}

	return map[string]types.AttributeValue{
		"PK": &types.AttributeValueMemberS{Value: pk},
		"SK": &types.AttributeValueMemberS{Value: sk},
	}, nil
}

// expandStringKey replaces macro patterns in the indexMap values with the provided key.
// It assumes that each value in the index map contains a macro pattern (e.g., "{id}").
// In a more robust implementation, you might want to validate that each value contains exactly one macro.
func expandStringKey(indexMap map[string]string, key string) (map[string]string, error) {
	expanded := make(map[string]string, len(indexMap))
	for field, template := range indexMap {
		// Replace all macro occurrences in the template with the provided key.
		// If the template contains multiple macros or unexpected content, you might need more advanced logic.
		expanded[field] = macroPattern.ReplaceAllString(template, key)
	}
	return expanded, nil
}

func buildSingleKey(expanded map[string]string) (map[string]types.AttributeValue, bool, error) {
	pk, hasPK := expanded["PK"]
	sk, hasSK := expanded["SK"]

	// If both exist and are identical, we treat them as a single object key.
	if hasPK && hasSK && pk != "" && sk != "" && pk == sk {
		return map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		}, true, nil
	}

	// Not a single key scenario.
	return nil, false, nil
}
