/*
Package errors provides semantic error types for the EntityStore library.

The package defines common error scenarios with specific types that can be
checked using the standard errors.Is() function or the provided helper functions.

Common Errors:

	var (
	    ErrNotFound        = errors.New("entity not found")
	    ErrAlreadyExists   = errors.New("entity already exists")
	    ErrInvalidInput    = errors.New("invalid input")
	    ErrConditionFailed = errors.New("condition check failed")
	    ErrNoIndexMap      = errors.New("no index map found for type")
	)

Usage:

	// Check error type
	user, err := store.GetOne(ctx, "123")
	if err != nil {
	    if errors.IsNotFound(err) {
	        // Handle not found case
	        return nil, fmt.Errorf("user %s does not exist", "123")
	    }
	    return nil, err
	}

	// Create typed errors
	err := errors.NewNotFoundError("User", "123")
	err := errors.NewValidationError("email", "invalid format")
	err := errors.NewConditionFailedError("update", "version mismatch")

The error types implement the error interface and support wrapping,
making them compatible with Go's standard error handling patterns.
*/
package errors