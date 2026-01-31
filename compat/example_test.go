package compat_test

import (
	"errors"
	"fmt"

	"github.com/go-extras/errx"
	"github.com/go-extras/errx/compat"
)

// ExampleWrap demonstrates using compat.Wrap with standard errors
func ExampleWrap() {
	// Define classification errors
	var (
		ErrDatabase  = errors.New("database error")
		ErrRetryable = errors.New("retryable error")
	)

	// Simulate a database error
	dbErr := errors.New("connection timeout")

	// Wrap with context and classifications using standard errors
	err := compat.Wrap("failed to fetch user", dbErr, ErrDatabase, ErrRetryable)

	fmt.Println(err.Error())
	fmt.Println("Is database error:", errors.Is(err, ErrDatabase))
	fmt.Println("Is retryable:", errors.Is(err, ErrRetryable))

	// Output:
	// failed to fetch user: connection timeout
	// Is database error: true
	// Is retryable: true
}

// ExampleClassify demonstrates using compat.Classify with standard errors
func ExampleClassify() {
	// Define classification error
	var ErrValidation = errors.New("validation error")

	// Error with a clear message that doesn't need additional context
	err := errors.New("email format is invalid")

	// Classify without changing the message
	classified := compat.Classify(err, ErrValidation)

	fmt.Println(classified.Error())
	fmt.Println("Is validation error:", errors.Is(classified, ErrValidation))

	// Output:
	// email format is invalid
	// Is validation error: true
}

// ExampleClassifyNew demonstrates creating and classifying an error in one step
func ExampleClassifyNew() {
	// Define classification errors
	var (
		ErrDatabase  = errors.New("database error")
		ErrRetryable = errors.New("retryable error")
	)

	// Create a new error and classify it in one step
	err := compat.ClassifyNew("connection timeout", ErrDatabase, ErrRetryable)

	fmt.Println(err.Error())
	fmt.Println("Is database error:", errors.Is(err, ErrDatabase))
	fmt.Println("Is retryable:", errors.Is(err, ErrRetryable))

	// Output:
	// connection timeout
	// Is database error: true
	// Is retryable: true
}

// ExampleClassifyNew_withErrxTypes demonstrates ClassifyNew with errx types
func ExampleClassifyNew_withErrxTypes() {
	// Mix standard errors with errx types
	var ErrNotFound = errors.New("not found")

	displayable := errx.NewDisplayable("The requested user does not exist")
	attrErr := errx.Attrs("user_id", 12345, "table", "users")

	err := compat.ClassifyNew("user record missing from database", ErrNotFound, displayable, attrErr)

	fmt.Println("Error:", err.Error())
	fmt.Println("Display text:", errx.DisplayText(err))
	fmt.Println("Is not found:", errors.Is(err, ErrNotFound))
	fmt.Println("Has attributes:", errx.HasAttrs(err))

	// Output:
	// Error: user record missing from database
	// Display text: The requested user does not exist
	// Is not found: true
	// Has attributes: true
}

// ExampleWrap_withErrxTypes demonstrates mixing standard errors with errx types
func ExampleWrap_withErrxTypes() {
	// Define classification error
	var ErrNotFound = errors.New("not found")

	baseErr := errors.New("user not found in database")

	// Mix standard errors with errx types
	displayable := errx.NewDisplayable("The requested user does not exist")
	attrErr := errx.Attrs("user_id", 12345, "table", "users")

	err := compat.Wrap("lookup failed", baseErr, ErrNotFound, displayable, attrErr)

	fmt.Println(err.Error())
	fmt.Println("Is not found:", errors.Is(err, ErrNotFound))
	fmt.Println("Has attributes:", errx.HasAttrs(err))

	// Output:
	// lookup failed: user not found in database
	// Is not found: true
	// Has attributes: true
}

// ExampleClassify_multipleClassifications demonstrates multiple classifications
func ExampleClassify_multipleClassifications() {
	// Define classification errors
	var (
		ErrDatabase  = errors.New("database error")
		ErrRetryable = errors.New("retryable error")
	)

	dbErr := errors.New("deadlock detected")

	// Attach multiple classifications
	err := compat.Classify(dbErr, ErrDatabase, ErrRetryable)

	fmt.Println(err.Error())
	fmt.Println("Is database error:", errors.Is(err, ErrDatabase))
	fmt.Println("Is retryable:", errors.Is(err, ErrRetryable))

	// Output:
	// deadlock detected
	// Is database error: true
	// Is retryable: true
}

// ExampleWrap_chaining demonstrates chaining compat calls
func ExampleWrap_chaining() {
	// Define classification errors
	var (
		ErrDatabase  = errors.New("database error")
		ErrRetryable = errors.New("retryable error")
	)

	// Start with a base error
	baseErr := errors.New("disk full")

	// Layer 1: Classify as database error
	err1 := compat.Classify(baseErr, ErrDatabase)

	// Layer 2: Add context and mark as retryable
	err2 := compat.Wrap("failed to write transaction log", err1, ErrRetryable)

	// Layer 3: Add more context
	err3 := compat.Wrap("transaction commit failed", err2)

	fmt.Println(err3.Error())
	fmt.Println("Is database error:", errors.Is(err3, ErrDatabase))
	fmt.Println("Is retryable:", errors.Is(err3, ErrRetryable))

	// Output:
	// transaction commit failed: failed to write transaction log: disk full
	// Is database error: true
	// Is retryable: true
}

// ExampleWrap_withAttributes demonstrates using attributes with compat
func ExampleWrap_withAttributes() {
	// Define classification error
	var ErrDatabase = errors.New("database error")

	baseErr := errors.New("query timeout")

	// Create attributed error for structured logging
	attrErr := errx.Attrs(
		"query", "SELECT * FROM users WHERE id = ?",
		"timeout_ms", 5000,
		"retry_count", 3,
	)

	err := compat.Wrap("database query failed", baseErr, ErrDatabase, attrErr)

	// Extract attributes for logging
	if errx.HasAttrs(err) {
		attrs := errx.ExtractAttrs(err)
		fmt.Println("Error:", err.Error())
		fmt.Println("Attributes:", len(attrs))
		for _, attr := range attrs {
			fmt.Printf("  %s: %v\n", attr.Key, attr.Value)
		}
	}

	// Output:
	// Error: database query failed: query timeout
	// Attributes: 3
	//   query: SELECT * FROM users WHERE id = ?
	//   timeout_ms: 5000
	//   retry_count: 3
}

// ExampleWrap_nilCause demonstrates nil handling
func ExampleWrap_nilCause() {
	// Define classification error
	var ErrNotFound = errors.New("not found")

	// Wrap returns nil when cause is nil
	err := compat.Wrap("context", nil, ErrNotFound)

	fmt.Println("Error is nil:", err == nil)

	// Output:
	// Error is nil: true
}

// ExampleClassify_nilCause demonstrates nil handling for Classify
func ExampleClassify_nilCause() {
	// Define classification error
	var ErrValidation = errors.New("validation error")

	// Classify returns nil when cause is nil
	err := compat.Classify(nil, ErrValidation)

	fmt.Println("Error is nil:", err == nil)

	// Output:
	// Error is nil: true
}
