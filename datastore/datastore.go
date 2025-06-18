/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package datastore

import (
	"context"
	"github.com/suparena/entitystore/storagemodels"
)

type DataStore[T any] interface {
	GetOne(ctx context.Context, key string) (*T, error)

	Put(ctx context.Context, entity T) error

	UpdateWithCondition(ctx context.Context, keyInput any, updates map[string]interface{}, condition string) error

	Query(ctx context.Context, params *storagemodels.QueryParams) ([]interface{}, error)

	// Stream returns a channel of StreamResult[T] for processing large result sets
	// The channel is closed when streaming completes or an error occurs
	// Use StreamOptions to configure buffering, retries, and progress tracking
	Stream(ctx context.Context, params *storagemodels.QueryParams, opts ...storagemodels.StreamOption) <-chan storagemodels.StreamResult[T]

	Delete(ctx context.Context, key string) error
}
