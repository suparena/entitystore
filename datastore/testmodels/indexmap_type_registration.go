// Code generated by postprocess tool; DO NOT EDIT.

package testmodels

import (
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/suparena/entitystore/registry"
)

func init() {
    // Register index map for model RatingSystem
    registry.RegisterIndexMap[RatingSystem](func() map[string]string {
        // Extract index map details from the vendor extension.
        return map[string]string{
            "PK": "{ID}",
            "SK": "{ID}",
        }
    }())
    // Register type registry for model RatingSystem.
    // The registry key is the model name (which is also injected as the EntityType when persisting).
    registry.RegisterType("RatingSystem", func(item map[string]types.AttributeValue) (interface{}, error) {
        obj := &RatingSystem{}
        err := attributevalue.UnmarshalMap(item, obj)
        return obj, err
    })
}
