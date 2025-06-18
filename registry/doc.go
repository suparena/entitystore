/*
Package registry manages type registration and index mapping for EntityStore.

The registry system enables:
  - Polymorphic entity storage in a single DynamoDB table
  - Dynamic type resolution based on EntityType attributes
  - Flexible key patterns through index maps

Type Registry:
Maps entity type names to unmarshal functions:

	registry.RegisterType("User", func() interface{} {
	    return &User{}
	})

Index Map Registry:
Associates Go types with DynamoDB key patterns:

	indexMap := map[string]string{
	    "PK": "USER#{ID}",
	    "SK": "USER#{ID}",
	    "GSI1PK": "EMAIL#{Email}",
	    "GSI1SK": "USER",
	}
	registry.RegisterIndexMap(User{}, indexMap)

The registry is thread-safe and should be populated during initialization,
typically in init() functions or through generated code.
*/
package registry