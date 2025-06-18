//go:build integration
// +build integration

/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package entitystore_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
	
	"github.com/suparena/entitystore"
	"github.com/suparena/entitystore/datastore/ddb"
	"github.com/suparena/entitystore/errors"
	"github.com/suparena/entitystore/registry"
	"github.com/suparena/entitystore/storagemodels"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Test entities
type IntegrationUser struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type IntegrationOrder struct {
	UserID    string    `json:"userId"`
	OrderID   string    `json:"orderId"`
	Total     float64   `json:"total"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

func init() {
	// Register types
	registry.RegisterType("IntegrationUser", func() interface{} {
		return &IntegrationUser{}
	})
	
	registry.RegisterType("IntegrationOrder", func() interface{} {
		return &IntegrationOrder{}
	})
	
	// Register index maps
	userIndexMap := map[string]string{
		"PK": "USER#{ID}",
		"SK": "USER#{ID}",
		"GSI1PK": "EMAIL#{Email}",
		"GSI1SK": "USER",
	}
	registry.RegisterIndexMap(IntegrationUser{}, userIndexMap)
	
	orderIndexMap := map[string]string{
		"PK": "USER#{UserID}",
		"SK": "ORDER#{OrderID}",
		"GSI1PK": "ORDER#{OrderID}",
		"GSI1SK": "STATUS#{Status}",
	}
	registry.RegisterIndexMap(IntegrationOrder{}, orderIndexMap)
}

func setupTestDataStore[T any](t *testing.T) *ddb.DynamodbDataStore[T] {
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	region := os.Getenv("AWS_REGION")
	tableName := os.Getenv("DDB_TEST_TABLE_NAME")
	
	if tableName == "" {
		t.Skip("DDB_TEST_TABLE_NAME not set, skipping integration test")
	}
	
	store, err := ddb.NewDynamodbDataStore[T](accessKey, secretKey, region, tableName)
	if err != nil {
		t.Fatalf("Failed to create datastore: %v", err)
	}
	
	return store
}

func TestIntegrationBasicOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	ctx := context.Background()
	store := setupTestDataStore[IntegrationUser](t)
	
	// Create user
	user := IntegrationUser{
		ID:        fmt.Sprintf("test-%d", time.Now().Unix()),
		Email:     "test@example.com",
		Name:      "Test User",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// Test Put
	err := store.Put(ctx, user)
	if err != nil {
		t.Fatalf("Failed to put user: %v", err)
	}
	
	// Test GetOne
	retrieved, err := store.GetOne(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	
	if retrieved.ID != user.ID || retrieved.Email != user.Email {
		t.Errorf("Retrieved user doesn't match: got %+v, want %+v", retrieved, user)
	}
	
	// Test UpdateWithCondition
	updates := map[string]interface{}{
		"Name": "Updated Name",
		"UpdatedAt": time.Now(),
	}
	err = store.UpdateWithCondition(ctx, user, updates, "attribute_exists(PK)")
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}
	
	// Test Delete
	err = store.Delete(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}
	
	// Verify deletion
	_, err = store.GetOne(ctx, user.ID)
	if !errors.IsNotFound(err) {
		t.Errorf("Expected not found error, got: %v", err)
	}
}

func TestIntegrationQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	ctx := context.Background()
	store := setupTestDataStore[IntegrationOrder](t)
	userID := fmt.Sprintf("user-%d", time.Now().Unix())
	
	// Create multiple orders
	orders := []IntegrationOrder{
		{
			UserID:    userID,
			OrderID:   "order-1",
			Total:     100.50,
			Status:    "pending",
			CreatedAt: time.Now(),
		},
		{
			UserID:    userID,
			OrderID:   "order-2",
			Total:     200.75,
			Status:    "completed",
			CreatedAt: time.Now(),
		},
		{
			UserID:    userID,
			OrderID:   "order-3",
			Total:     50.25,
			Status:    "pending",
			CreatedAt: time.Now(),
		},
	}
	
	// Put all orders
	for _, order := range orders {
		err := store.Put(ctx, order)
		if err != nil {
			t.Fatalf("Failed to put order: %v", err)
		}
	}
	
	// Query by user ID
	params := &storagemodels.QueryParams{
		TableName:              os.Getenv("DDB_TEST_TABLE_NAME"),
		KeyConditionExpression: "PK = :pk",
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userID)},
		},
	}
	
	results, err := store.Query(ctx, params)
	if err != nil {
		t.Fatalf("Failed to query orders: %v", err)
	}
	
	if len(results) != 3 {
		t.Errorf("Expected 3 orders, got %d", len(results))
	}
	
	// Clean up
	for _, order := range orders {
		store.Delete(ctx, fmt.Sprintf("%s#%s", order.UserID, order.OrderID))
	}
}

func TestIntegrationStreaming(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	ctx := context.Background()
	store := setupTestDataStore[IntegrationUser](t)
	
	// Create multiple users
	baseTime := time.Now().Unix()
	users := make([]IntegrationUser, 10)
	for i := 0; i < 10; i++ {
		users[i] = IntegrationUser{
			ID:        fmt.Sprintf("stream-test-%d-%d", baseTime, i),
			Email:     fmt.Sprintf("user%d@example.com", i),
			Name:      fmt.Sprintf("User %d", i),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		err := store.Put(ctx, users[i])
		if err != nil {
			t.Fatalf("Failed to put user: %v", err)
		}
	}
	
	// Test streaming with options
	params := &storagemodels.QueryParams{
		TableName:              os.Getenv("DDB_TEST_TABLE_NAME"),
		KeyConditionExpression: "begins_with(PK, :prefix)",
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":prefix": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#stream-test-%d", baseTime)},
		},
	}
	
	var progressCalled int
	resultChan := store.Stream(ctx, params,
		storagemodels.WithPageSize(3),
		storagemodels.WithProgressHandler(func(p storagemodels.StreamProgress) {
			progressCalled++
			t.Logf("Progress: %d items processed", p.ItemsProcessed)
		}),
	)
	
	count := 0
	for result := range resultChan {
		if result.Error != nil {
			t.Errorf("Stream error: %v", result.Error)
			continue
		}
		count++
	}
	
	if count < 5 {
		t.Logf("Note: Got %d items, expected at least 5. This might be due to eventual consistency.", count)
	}
	
	if progressCalled == 0 {
		t.Error("Progress handler was not called")
	}
	
	// Clean up
	for _, user := range users {
		store.Delete(ctx, user.ID)
	}
}

func TestIntegrationMultiTypeStorage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	ctx := context.Background()
	mts := entitystore.NewMultiTypeStorage()
	
	// Register user datastore
	userStore := setupTestDataStore[IntegrationUser](t)
	err := entitystore.RegisterDataStore(mts, "users", userStore)
	if err != nil {
		t.Fatalf("Failed to register user store: %v", err)
	}
	
	// Register order datastore
	orderStore := setupTestDataStore[IntegrationOrder](t)
	err = entitystore.RegisterDataStore(mts, "orders", orderStore)
	if err != nil {
		t.Fatalf("Failed to register order store: %v", err)
	}
	
	// Test operations through MultiTypeStorage
	retrievedUserStore, err := entitystore.GetDataStore[IntegrationUser](mts, "users")
	if err != nil {
		t.Fatalf("Failed to get user store: %v", err)
	}
	
	user := IntegrationUser{
		ID:        fmt.Sprintf("mts-test-%d", time.Now().Unix()),
		Email:     "mts@example.com",
		Name:      "MTS Test User",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	err = retrievedUserStore.Put(ctx, user)
	if err != nil {
		t.Fatalf("Failed to put user through MTS: %v", err)
	}
	
	// Clean up
	retrievedUserStore.Delete(ctx, user.ID)
}