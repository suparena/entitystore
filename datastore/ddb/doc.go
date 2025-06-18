/*
Package ddb provides a DynamoDB implementation of the DataStore interface.

The DynamodbDataStore supports:
  - Single-table design patterns
  - Macro-based key expansion (e.g., "USER#{ID}")
  - Global Secondary Index (GSI) queries
  - Enhanced streaming with retry logic
  - Conditional updates for optimistic locking
  - Automatic EntityType injection for polymorphic storage

Key Features:

Macro Expansion:
Keys can use macros that are replaced with entity field values:

	indexMap := map[string]string{
	    "PK": "USER#{ID}",        // Becomes "USER#123"
	    "SK": "PROFILE",          // Static value
	    "GSI1PK": "{Email}",      // Direct field value
	}

Streaming:
The enhanced streaming API supports configurable options:

	results := store.Stream(ctx, params,
	    storagemodels.WithBufferSize(100),
	    storagemodels.WithPageSize(25),
	    storagemodels.WithMaxRetries(3),
	    storagemodels.WithProgressHandler(func(p StreamProgress) {
	        log.Printf("Processed %d items", p.ItemsProcessed)
	    }),
	)

For usage examples, see the integration tests and documentation.
*/
package ddb