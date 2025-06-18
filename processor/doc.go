/*
Package processor provides code generation functionality for EntityStore.

The processor reads OpenAPI specifications with vendor extensions and generates
Go code for automatic type registration and index mapping.

OpenAPI Extension:
The processor looks for the x-dynamodb-indexmap vendor extension:

	UserProfile:
	  type: object
	  x-dynamodb-indexmap:
	    PK: "USER#{UserId}"
	    SK: "PROFILE"
	    GSI1PK: "EMAIL#{Email}"
	    GSI1SK: "USER"
	  properties:
	    userId:
	      type: string
	    email:
	      type: string

Generated Code:
The processor generates registration code:

	func init() {
	    // Register type
	    registry.RegisterType("UserProfile", func() interface{} {
	        return &UserProfile{}
	    })
	    
	    // Register index map
	    registry.RegisterIndexMap(UserProfile{}, map[string]string{
	        "PK": "USER#{UserId}",
	        "SK": "PROFILE",
	        "GSI1PK": "EMAIL#{Email}",
	        "GSI1SK": "USER",
	    })
	}

This automation reduces boilerplate and ensures consistency between
the API specification and storage configuration.
*/
package processor