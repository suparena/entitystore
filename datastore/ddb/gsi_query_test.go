/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package ddb

import (
	"context"
	"testing"
	
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/suparena/entitystore/registry"
)

// TestEntity for GSI testing
type GSITestEntity struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Status   string `json:"status"`
	Country  string `json:"country"`
	Score    int    `json:"score"`
}

func init() {
	// Register test entity
	registry.RegisterType("GSITestEntity", func(item map[string]types.AttributeValue) (interface{}, error) {
		entity := &GSITestEntity{}
		// In real code, you would unmarshal the item into entity
		// For testing, we'll just return an empty entity
		return entity, nil
	})
	
	// Register index map with common GSI patterns
	indexMap := map[string]string{
		"PK":     "ENTITY#{ID}",
		"SK":     "ENTITY#{ID}",
		"GSI1PK": "EMAIL#{Email}",
		"GSI1SK": "STATUS#{Status}",
	}
	registry.RegisterIndexMap[GSITestEntity](indexMap)
}

func TestGSIQueryBuilder(t *testing.T) {
	// This would need a mock or test DynamoDB instance
	// For now, we'll test the query building logic
	
	t.Run("BuildBasicGSIQuery", func(t *testing.T) {
		store := &DynamodbDataStore[GSITestEntity]{
			tableName: "test-table",
		}
		
		builder := store.QueryGSI().
			WithPartitionKey("test@example.com")
		
		params, err := builder.Build()
		if err != nil {
			t.Fatalf("Failed to build query: %v", err)
		}
		
		// Check that GSI1 is used
		if params.IndexName == nil || *params.IndexName != "GSI1" {
			t.Errorf("Expected IndexName to be GSI1")
		}
		
		// Check key condition
		expectedKey := "PK1 = :pk"
		if params.KeyConditionExpression != expectedKey {
			t.Errorf("Expected key condition %s, got %s", expectedKey, params.KeyConditionExpression)
		}
		
		// Check that email prefix was added
		pkVal := params.ExpressionAttributeValues[":pk"].(*types.AttributeValueMemberS).Value
		if pkVal != "EMAIL#test@example.com" {
			t.Errorf("Expected PK value EMAIL#test@example.com, got %s", pkVal)
		}
	})
	
	t.Run("BuildGSIQueryWithSortKey", func(t *testing.T) {
		store := &DynamodbDataStore[GSITestEntity]{
			tableName: "test-table",
		}
		
		builder := store.QueryGSI().
			WithPartitionKey("test@example.com").
			WithSortKey("active")
		
		params, err := builder.Build()
		if err != nil {
			t.Fatalf("Failed to build query: %v", err)
		}
		
		// Check key condition includes sort key
		expectedKey := "PK1 = :pk AND SK1 = :sk"
		if params.KeyConditionExpression != expectedKey {
			t.Errorf("Expected key condition %s, got %s", expectedKey, params.KeyConditionExpression)
		}
		
		// Check sort key value has prefix
		skVal := params.ExpressionAttributeValues[":sk"].(*types.AttributeValueMemberS).Value
		if skVal != "STATUS#active" {
			t.Errorf("Expected SK value STATUS#active, got %s", skVal)
		}
	})
	
	t.Run("BuildGSIQueryWithSortKeyPrefix", func(t *testing.T) {
		store := &DynamodbDataStore[GSITestEntity]{
			tableName: "test-table",
		}
		
		builder := store.QueryGSI().
			WithPartitionKey("test@example.com").
			WithSortKeyPrefix("STATUS#act")
		
		params, err := builder.Build()
		if err != nil {
			t.Fatalf("Failed to build query: %v", err)
		}
		
		// Check key condition uses begins_with
		expectedKey := "PK1 = :pk AND begins_with(SK1, :sk)"
		if params.KeyConditionExpression != expectedKey {
			t.Errorf("Expected key condition %s, got %s", expectedKey, params.KeyConditionExpression)
		}
	})
	
	t.Run("BuildGSIQueryWithFilter", func(t *testing.T) {
		store := &DynamodbDataStore[GSITestEntity]{
			tableName: "test-table",
		}
		
		filterValues := map[string]types.AttributeValue{
			":country": &types.AttributeValueMemberS{Value: "USA"},
		}
		
		builder := store.QueryGSI().
			WithPartitionKey("test@example.com").
			WithFilter("Country = :country", filterValues).
			WithLimit(10)
		
		params, err := builder.Build()
		if err != nil {
			t.Fatalf("Failed to build query: %v", err)
		}
		
		// Check filter expression
		if params.FilterExpression == nil || *params.FilterExpression != "Country = :country" {
			t.Errorf("Expected filter expression 'Country = :country'")
		}
		
		// Check filter values
		if params.ExpressionAttributeValues[":country"] == nil {
			t.Error("Expected :country in expression attribute values")
		}
		
		// Check limit
		if params.Limit == nil || *params.Limit != 10 {
			t.Errorf("Expected limit 10")
		}
	})
	
	t.Run("BuildGSIQueryWithSortKeyRange", func(t *testing.T) {
		store := &DynamodbDataStore[GSITestEntity]{
			tableName: "test-table",
		}
		
		builder := store.QueryGSI().
			WithPartitionKey("test@example.com").
			WithSortKeyBetween("STATUS#active", "STATUS#inactive")
		
		params, err := builder.Build()
		if err != nil {
			t.Fatalf("Failed to build query: %v", err)
		}
		
		// Check key condition uses BETWEEN
		expectedKey := "PK1 = :pk AND SK1 BETWEEN :sk AND :sk2"
		if params.KeyConditionExpression != expectedKey {
			t.Errorf("Expected key condition %s, got %s", expectedKey, params.KeyConditionExpression)
		}
		
		// Check both sort key values
		sk1Val := params.ExpressionAttributeValues[":sk"].(*types.AttributeValueMemberS).Value
		if sk1Val != "STATUS#active" {
			t.Errorf("Expected SK1 value STATUS#active, got %s", sk1Val)
		}
		
		sk2Val := params.ExpressionAttributeValues[":sk2"].(*types.AttributeValueMemberS).Value
		if sk2Val != "STATUS#inactive" {
			t.Errorf("Expected SK2 value STATUS#inactive, got %s", sk2Val)
		}
	})
	
	t.Run("QueryBuilderValidation", func(t *testing.T) {
		store := &DynamodbDataStore[GSITestEntity]{
			tableName: "test-table",
		}
		
		// Test missing partition key
		builder := store.QueryGSI()
		_, err := builder.Build()
		if err == nil {
			t.Error("Expected error for missing partition key")
		}
	})
}

func TestGSIQueryConvenienceMethods(t *testing.T) {
	// These tests would require mocking the DynamoDB client
	// They demonstrate the usage patterns
	
	t.Run("ConvenienceMethodsExist", func(t *testing.T) {
		store := &DynamodbDataStore[GSITestEntity]{
			tableName: "test-table",
		}
		
		// Verify methods exist and can be called
		ctx := context.Background()
		
		// These would fail without a real DynamoDB connection
		// but we're testing that the methods exist and compile
		
		_ = func() {
			_, _ = store.QueryByGSI1PK(ctx, "test@example.com")
		}
		
		_ = func() {
			_, _ = store.QueryByGSI1PKAndSKPrefix(ctx, "test@example.com", "STATUS")
		}
		
		_ = func() {
			filters := map[string]types.AttributeValue{
				":score": &types.AttributeValueMemberN{Value: "100"},
			}
			_, _ = store.QueryByGSI1PKWithFilter(ctx, "test@example.com", "Score > :score", filters)
		}
	})
}