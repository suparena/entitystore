/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package errors

import (
	"errors"
	"fmt"
)

// Common sentinel errors
var (
	// ErrNotFound is returned when an entity is not found
	ErrNotFound = errors.New("entity not found")
	
	// ErrAlreadyExists is returned when attempting to create an entity that already exists
	ErrAlreadyExists = errors.New("entity already exists")
	
	// ErrInvalidInput is returned when input validation fails
	ErrInvalidInput = errors.New("invalid input")
	
	// ErrConditionFailed is returned when a conditional update fails
	ErrConditionFailed = errors.New("condition check failed")
	
	// ErrNoIndexMap is returned when no index map is found for a type
	ErrNoIndexMap = errors.New("no index map found for type")
)

// NotFoundError represents an error when an entity is not found
type NotFoundError struct {
	Type string
	Key  string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with key %q not found", e.Type, e.Key)
}

func (e *NotFoundError) Is(target error) bool {
	return target == ErrNotFound
}

// AlreadyExistsError represents an error when an entity already exists
type AlreadyExistsError struct {
	Type string
	Key  string
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("%s with key %q already exists", e.Type, e.Key)
}

func (e *AlreadyExistsError) Is(target error) bool {
	return target == ErrAlreadyExists
}

// ValidationError represents an input validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation failed for field %q: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation failed: %s", e.Message)
}

func (e *ValidationError) Is(target error) bool {
	return target == ErrInvalidInput
}

// ConditionFailedError represents a failed conditional operation
type ConditionFailedError struct {
	Operation string
	Condition string
}

func (e *ConditionFailedError) Error() string {
	return fmt.Sprintf("condition check failed for %s operation: %s", e.Operation, e.Condition)
}

func (e *ConditionFailedError) Is(target error) bool {
	return target == ErrConditionFailed
}

// Helper functions for creating errors

// NewNotFoundError creates a new NotFoundError
func NewNotFoundError(entityType, key string) error {
	return &NotFoundError{Type: entityType, Key: key}
}

// NewAlreadyExistsError creates a new AlreadyExistsError
func NewAlreadyExistsError(entityType, key string) error {
	return &AlreadyExistsError{Type: entityType, Key: key}
}

// NewValidationError creates a new ValidationError
func NewValidationError(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}

// NewConditionFailedError creates a new ConditionFailedError
func NewConditionFailedError(operation, condition string) error {
	return &ConditionFailedError{Operation: operation, Condition: condition}
}

// IsNotFound checks if an error is a not found error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsAlreadyExists checks if an error is an already exists error
func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrAlreadyExists)
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

// IsConditionFailed checks if an error is a condition failed error
func IsConditionFailed(err error) bool {
	return errors.Is(err, ErrConditionFailed)
}