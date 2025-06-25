/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

// Package mock provides mock implementations of the DataStore interface for testing
package mock

import (
	"context"
	"fmt"
	"sync"
	
	"github.com/suparena/entitystore/errors"
	"github.com/suparena/entitystore/storagemodels"
)

// DataStore is a mock implementation of datastore.DataStore[T] for testing
type DataStore[T any] struct {
	mu           sync.RWMutex
	data         map[string]T
	queryFunc    func(ctx context.Context, params *storagemodels.QueryParams) ([]interface{}, error)
	streamFunc   func(ctx context.Context, params *storagemodels.QueryParams, opts ...storagemodels.StreamOption) <-chan storagemodels.StreamResult[T]
	getKeyFunc   func(entity T) string
	putError     error
	deleteError  error
	updateError  error
}

// New creates a new mock DataStore
func New[T any]() *DataStore[T] {
	return &DataStore[T]{
		data: make(map[string]T),
	}
}

// WithGetKeyFunc sets a custom function to extract keys from entities
func (m *DataStore[T]) WithGetKeyFunc(f func(T) string) *DataStore[T] {
	m.getKeyFunc = f
	return m
}

// WithQueryFunc sets a custom query function for testing
func (m *DataStore[T]) WithQueryFunc(f func(ctx context.Context, params *storagemodels.QueryParams) ([]interface{}, error)) *DataStore[T] {
	m.queryFunc = f
	return m
}

// WithStreamFunc sets a custom stream function for testing
func (m *DataStore[T]) WithStreamFunc(f func(ctx context.Context, params *storagemodels.QueryParams, opts ...storagemodels.StreamOption) <-chan storagemodels.StreamResult[T]) *DataStore[T] {
	m.streamFunc = f
	return m
}

// WithPutError makes Put operations return an error
func (m *DataStore[T]) WithPutError(err error) *DataStore[T] {
	m.putError = err
	return m
}

// WithDeleteError makes Delete operations return an error
func (m *DataStore[T]) WithDeleteError(err error) *DataStore[T] {
	m.deleteError = err
	return m
}

// WithUpdateError makes UpdateWithCondition operations return an error
func (m *DataStore[T]) WithUpdateError(err error) *DataStore[T] {
	m.updateError = err
	return m
}

// GetOne retrieves an entity by key
func (m *DataStore[T]) GetOne(ctx context.Context, key string) (*T, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if entity, exists := m.data[key]; exists {
		return &entity, nil
	}
	
	var zero T
	return nil, errors.NewNotFoundError(fmt.Sprintf("%T", zero), key)
}

// GetByKey retrieves an entity by explicit PK and SK values
func (m *DataStore[T]) GetByKey(ctx context.Context, pk, sk string) (*T, error) {
	// For mock, we'll use the composite key format
	key := fmt.Sprintf("%s|%s", pk, sk)
	return m.GetOne(ctx, key)
}

// Put stores an entity
func (m *DataStore[T]) Put(ctx context.Context, entity T) error {
	if m.putError != nil {
		return m.putError
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := m.extractKey(entity)
	if key == "" {
		return errors.NewValidationError("key", "unable to extract key from entity")
	}
	
	m.data[key] = entity
	return nil
}

// UpdateWithCondition updates an entity with a condition
func (m *DataStore[T]) UpdateWithCondition(ctx context.Context, keyInput any, updates map[string]interface{}, condition string) error {
	if m.updateError != nil {
		return m.updateError
	}
	
	// Simple mock implementation - just check if key exists
	key, ok := keyInput.(string)
	if !ok {
		return errors.NewValidationError("keyInput", "must be a string for mock")
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.data[key]; !exists {
		return errors.NewNotFoundError("entity", key)
	}
	
	// In a real implementation, we would apply the updates
	// For mock, we just verify the entity exists
	return nil
}

// Query executes a query
func (m *DataStore[T]) Query(ctx context.Context, params *storagemodels.QueryParams) ([]interface{}, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, params)
	}
	
	// Default implementation returns all data as interface{}
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	results := make([]interface{}, 0, len(m.data))
	for _, v := range m.data {
		results = append(results, v)
	}
	
	return results, nil
}

// Stream returns a channel of results
func (m *DataStore[T]) Stream(ctx context.Context, params *storagemodels.QueryParams, opts ...storagemodels.StreamOption) <-chan storagemodels.StreamResult[T] {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, params, opts...)
	}
	
	// Default implementation streams all data
	resultChan := make(chan storagemodels.StreamResult[T], 10)
	
	go func() {
		defer close(resultChan)
		
		m.mu.RLock()
		defer m.mu.RUnlock()
		
		index := int64(0)
		for _, v := range m.data {
			select {
			case <-ctx.Done():
				return
			case resultChan <- storagemodels.StreamResult[T]{
				Item: v,
				Meta: storagemodels.StreamMeta{
					Index:      index,
					PageNumber: 1,
				},
			}:
				index++
			}
		}
	}()
	
	return resultChan
}

// Delete removes an entity by key
func (m *DataStore[T]) Delete(ctx context.Context, key string) error {
	if m.deleteError != nil {
		return m.deleteError
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.data[key]; !exists {
		var zero T
		return errors.NewNotFoundError(fmt.Sprintf("%T", zero), key)
	}
	
	delete(m.data, key)
	return nil
}

// Helper methods for testing

// SetData directly sets the internal data map (for testing)
func (m *DataStore[T]) SetData(data map[string]T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = data
}

// GetData returns a copy of the internal data map (for testing)
func (m *DataStore[T]) GetData() map[string]T {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[string]T, len(m.data))
	for k, v := range m.data {
		result[k] = v
	}
	return result
}

// Count returns the number of stored entities
func (m *DataStore[T]) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}

// Clear removes all data
func (m *DataStore[T]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]T)
}

// extractKey attempts to extract a key from an entity
func (m *DataStore[T]) extractKey(entity T) string {
	if m.getKeyFunc != nil {
		return m.getKeyFunc(entity)
	}
	
	// Default: try to use ID field via reflection
	// This is a simplified version for testing
	return fmt.Sprintf("key_%v", entity)
}