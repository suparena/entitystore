package registry

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// UnmarshalFunc defines a function that takes a raw DynamoDB item and returns the unmarshaled object.
type UnmarshalFunc func(item map[string]types.AttributeValue) (interface{}, error)

// typeRegistry holds the mapping from a type prefix (like "PL", "DR", etc.) to its unmarshal function.
var typeRegistry = make(map[string]UnmarshalFunc)

// RegisterType registers an unmarshal function for a given type prefix.
// If a type is already registered for the given prefix, it panics to prevent accidental overrides.
func RegisterType(prefix string, fn UnmarshalFunc) {
	if _, exists := typeRegistry[prefix]; exists {
		panic(fmt.Sprintf("type registry: type with prefix %q already registered", prefix))
	}
	typeRegistry[prefix] = fn
}

// GetUnmarshalFunc returns the registered unmarshal function for the given type prefix.
// If no function is registered, it returns an error.
func GetUnmarshalFunc(prefix string) (UnmarshalFunc, error) {
	fn, ok := typeRegistry[prefix]
	if !ok {
		return nil, fmt.Errorf("type registry: no type registered for prefix %q", prefix)
	}
	return fn, nil
}
