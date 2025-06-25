/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package ddb

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/suparena/entitystore/registry"
)

// GSIPutTestEntity for testing GSI key mapping
type GSIPutTestEntity struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Status      string    `json:"status"`
	Category    string    `json:"category"`
	Priority    int       `json:"priority"`
	CreatedAt   time.Time `json:"createdAt"`
}

func init() {
	// Register test entity
	registry.RegisterType("GSIPutTestEntity", func(item map[string]types.AttributeValue) (interface{}, error) {
		entity := &GSIPutTestEntity{}
		return entity, nil
	})

	// Register index map with GSI patterns
	indexMap := map[string]string{
		"PK":     "ENTITY#{ID}",
		"SK":     "ENTITY#{ID}",
		"GSI1PK": "EMAIL#{Email}",
		"GSI1SK": "STATUS#{Status}",
		"GSI2PK": "CATEGORY#{Category}",
		"GSI2SK": "PRIORITY#{Priority}",
	}
	registry.RegisterIndexMap[GSIPutTestEntity](indexMap)
}

func TestPutWithGSIKeyMapping(t *testing.T) {
	// These tests validate the GSI key mapping logic but require a real DynamoDB
	// client or a more sophisticated mock. For now, we'll focus on unit testing
	// the key mapping logic separately.
	t.Skip("Skipping GSI Put tests - requires mock infrastructure")

}

// TestGSIConfigIntegration tests the GSI configuration system
func TestGSIConfigIntegration(t *testing.T) {
	t.Run("DefaultGSIConfigsExist", func(t *testing.T) {
		// Test GSI1
		gsi1Config, ok := GetGSIConfig("GSI1")
		if !ok {
			t.Error("GSI1 config should exist")
		}
		if gsi1Config.PartitionKeyName != "PK1" {
			t.Errorf("Expected PK1, got %s", gsi1Config.PartitionKeyName)
		}
		if gsi1Config.SortKeyName != "SK1" {
			t.Errorf("Expected SK1, got %s", gsi1Config.SortKeyName)
		}

		// Test GSI2
		gsi2Config, ok := GetGSIConfig("GSI2")
		if !ok {
			t.Error("GSI2 config should exist")
		}
		if gsi2Config.PartitionKeyName != "PK2" {
			t.Errorf("Expected PK2, got %s", gsi2Config.PartitionKeyName)
		}

		// Test GSI3
		gsi3Config, ok := GetGSIConfig("GSI3")
		if !ok {
			t.Error("GSI3 config should exist")
		}
		if gsi3Config.PartitionKeyName != "PK3" {
			t.Errorf("Expected PK3, got %s", gsi3Config.PartitionKeyName)
		}
	})

	t.Run("NonExistentGSIConfig", func(t *testing.T) {
		_, ok := GetGSIConfig("GSI99")
		if ok {
			t.Error("GSI99 should not exist")
		}
	})
}
