/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package ddb

// GSIConfig holds the configuration for GSI key mappings
type GSIConfig struct {
	// IndexName is the actual GSI name in DynamoDB (e.g., "GSI1")
	IndexName string
	// PartitionKeyName is the actual partition key attribute name in the GSI (e.g., "PK1")
	PartitionKeyName string
	// SortKeyName is the actual sort key attribute name in the GSI (e.g., "SK1")
	SortKeyName string
}

// DefaultGSIConfigs holds the default GSI configurations
var DefaultGSIConfigs = map[string]GSIConfig{
	"GSI1": {
		IndexName:        "GSI1",
		PartitionKeyName: "PK1",
		SortKeyName:      "SK1",
	},
}

// GetGSIConfig returns the GSI configuration for a given index name
func GetGSIConfig(indexName string) (GSIConfig, bool) {
	config, ok := DefaultGSIConfigs[indexName]
	return config, ok
}