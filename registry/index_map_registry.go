/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package registry

import (
	"reflect"
	"sync"
)

// IndexMapRegistry is a registry for Go types and their DynamoDB index maps.

var (
	indexMapRegistry = make(map[reflect.Type]map[string]string)
	mu               sync.RWMutex
)

// RegisterIndexMap associates a Go type T with a given DynamoDB index map (PK, SK, etc.).
func RegisterIndexMap[T any](idxMap map[string]string) {
	var zero T
	t := reflect.TypeOf(zero)

	mu.Lock()
	defer mu.Unlock()
	indexMapRegistry[t] = idxMap
}

// GetIndexMap retrieves the indexMap for type T, if any.
func GetIndexMap[T any]() (map[string]string, bool) {
	var zero T
	t := reflect.TypeOf(zero)

	mu.RLock()
	defer mu.RUnlock()
	m, ok := indexMapRegistry[t]
	return m, ok
}
