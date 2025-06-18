/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package entitystore

import (
	"fmt"
	"sync"
)

// Storage is a higher-level interface that manages a collection of DataStore instances.
// Note that its methods are not generic; they use the empty interface (any) to store and retrieve DataStores.
type Storage interface {
	// RegisterDataStore registers a DataStore under a given key (for example, "Player" or "RatingRecord").
	RegisterDataStore(key string, ds any) error
	// GetDataStore retrieves the registered DataStore for a given key.
	// The caller must type-assert the returned value to the appropriate DataStore type.
	GetDataStore(key string) (any, error)
}

// storageManager is a thread-safe implementation of the Storage interface.
type storageManager struct {
	mu     sync.RWMutex
	stores map[string]any
}

// NewStorageManager creates and returns a new Storage implementation.
func NewStorageManager() Storage {
	return &storageManager{
		stores: make(map[string]any),
	}
}

// RegisterDataStore stores the provided DataStore under the given key.
func (sm *storageManager) RegisterDataStore(key string, ds any) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if _, exists := sm.stores[key]; exists {
		return fmt.Errorf("datastore with key %q already registered", key)
	}
	sm.stores[key] = ds
	return nil
}

// GetDataStore retrieves the DataStore associated with the given key.
func (sm *storageManager) GetDataStore(key string) (any, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	ds, exists := sm.stores[key]
	if !exists {
		return nil, fmt.Errorf("datastore with key %q not found", key)
	}
	return ds, nil
}
