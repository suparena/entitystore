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

	Stream(ctx context.Context, params *storagemodels.StreamQueryParams) (<-chan storagemodels.StreamItem, <-chan error)

	Delete(ctx context.Context, key string) error
}
