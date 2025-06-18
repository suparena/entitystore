/*
 * Copyright © 2025 Suparena Software Inc., All rights reserved.
 */

package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestNotFoundError(t *testing.T) {
	err := NewNotFoundError("User", "123")
	
	// Test error message
	expected := `User with key "123" not found`
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
	
	// Test Is method
	if !errors.Is(err, ErrNotFound) {
		t.Error("NotFoundError should match ErrNotFound")
	}
	
	// Test helper function
	if !IsNotFound(err) {
		t.Error("IsNotFound should return true for NotFoundError")
	}
}

func TestAlreadyExistsError(t *testing.T) {
	err := NewAlreadyExistsError("Product", "ABC")
	
	// Test error message
	expected := `Product with key "ABC" already exists`
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
	
	// Test Is method
	if !errors.Is(err, ErrAlreadyExists) {
		t.Error("AlreadyExistsError should match ErrAlreadyExists")
	}
	
	// Test helper function
	if !IsAlreadyExists(err) {
		t.Error("IsAlreadyExists should return true for AlreadyExistsError")
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		message  string
		expected string
	}{
		{
			name:     "with field",
			field:    "email",
			message:  "invalid format",
			expected: `validation failed for field "email": invalid format`,
		},
		{
			name:     "without field",
			field:    "",
			message:  "missing required fields",
			expected: "validation failed: missing required fields",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidationError(tt.field, tt.message)
			
			if err.Error() != tt.expected {
				t.Errorf("Expected error message %q, got %q", tt.expected, err.Error())
			}
			
			if !errors.Is(err, ErrInvalidInput) {
				t.Error("ValidationError should match ErrInvalidInput")
			}
			
			if !IsValidationError(err) {
				t.Error("IsValidationError should return true for ValidationError")
			}
		})
	}
}

func TestConditionFailedError(t *testing.T) {
	err := NewConditionFailedError("update", "version = :oldVersion")
	
	// Test error message
	expected := "condition check failed for update operation: version = :oldVersion"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
	
	// Test Is method
	if !errors.Is(err, ErrConditionFailed) {
		t.Error("ConditionFailedError should match ErrConditionFailed")
	}
	
	// Test helper function
	if !IsConditionFailed(err) {
		t.Error("IsConditionFailed should return true for ConditionFailedError")
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test that wrapped errors still match
	original := NewNotFoundError("User", "123")
	wrapped := fmt.Errorf("database operation failed: %w", original)
	
	if !errors.Is(wrapped, ErrNotFound) {
		t.Error("Wrapped NotFoundError should still match ErrNotFound")
	}
	
	if !IsNotFound(wrapped) {
		t.Error("IsNotFound should work with wrapped errors")
	}
}

func TestSentinelErrors(t *testing.T) {
	// Ensure sentinel errors are distinct
	sentinels := []error{
		ErrNotFound,
		ErrAlreadyExists,
		ErrInvalidInput,
		ErrConditionFailed,
		ErrNoIndexMap,
	}
	
	for i, err1 := range sentinels {
		for j, err2 := range sentinels {
			if i != j && errors.Is(err1, err2) {
				t.Errorf("Sentinel errors should be distinct: %v matches %v", err1, err2)
			}
		}
	}
}