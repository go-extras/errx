package errx_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/go-extras/errx"
)

// Example demonstrates basic usage of classification tags
func Example() {
	// Define classification tags
	ErrNotFound := errx.NewSentinel("resource not found")

	// Create an error and classify it
	err := errors.New("record missing")
	classifiedErr := errx.Classify(err, ErrNotFound)

	// Check the classification
	if errors.Is(classifiedErr, ErrNotFound) {
		fmt.Println("Error is classified as not found")
	}

	// Output:
	// Error is classified as not found
}

// ExampleWrap demonstrates wrapping errors with context and tags
func ExampleWrap() {
	ErrDatabase := errx.NewSentinel("database error")

	err := errors.New("connection timeout")
	wrapped := errx.Wrap("failed to query database", err, ErrDatabase)

	fmt.Println(wrapped.Error())
	fmt.Println("Is database error:", errors.Is(wrapped, ErrDatabase))

	// Output:
	// failed to query database: connection timeout
	// Is database error: true
}

// ExampleClassify demonstrates classifying errors without adding context
func ExampleClassify() {
	ErrValidation := errx.NewSentinel("validation error")

	err := errors.New("invalid email format")
	classified := errx.Classify(err, ErrValidation)

	fmt.Println(classified.Error())
	fmt.Println("Is validation error:", errors.Is(classified, ErrValidation))

	// Output:
	// invalid email format
	// Is validation error: true
}

// ExampleNewSentinel_multipleParents demonstrates creating a sentinel with multiple parent sentinels
func ExampleNewSentinel_multipleParents() {
	// Create independent classification dimensions
	ErrRetryable := errx.NewSentinel("retryable")
	ErrDatabase := errx.NewSentinel("database")

	// Create a tag that inherits from both
	ErrDatabaseTimeout := errx.NewSentinel("database timeout", ErrDatabase, ErrRetryable)

	err := errx.Wrap("query failed", errors.New("connection timeout"), ErrDatabaseTimeout)

	// Can check for specific error
	if errors.Is(err, ErrDatabaseTimeout) {
		fmt.Println("Specific: database timeout")
	}

	// Can check for database errors
	if errors.Is(err, ErrDatabase) {
		fmt.Println("Category: database error")
	}

	// Can check for retryable errors
	if errors.Is(err, ErrRetryable) {
		fmt.Println("Behavior: retryable error")
	}

	// Output:
	// Specific: database timeout
	// Category: database error
	// Behavior: retryable error
}

// ExampleNewDisplayable demonstrates creating displayable errors
func ExampleNewDisplayable() {
	displayErr := errx.NewDisplayable("User not found")

	fmt.Println(displayErr.Error())
	fmt.Println("Is displayable:", errx.IsDisplayable(displayErr))

	// Output:
	// User not found
	// Is displayable: true
}

// ExampleDisplayText demonstrates extracting displayable messages
func ExampleDisplayText() {
	displayErr := errx.NewDisplayable("Invalid email address")
	wrapped := errx.Wrap("validation failed", displayErr)

	// Extract just the displayable message
	msg := errx.DisplayText(wrapped)
	fmt.Println("Display message:", msg)

	// Full error for logging
	fmt.Println("Full error:", wrapped.Error())

	// Output:
	// Display message: Invalid email address
	// Full error: validation failed: Invalid email address
}

// ExampleDisplayTextDefault demonstrates extracting displayable messages with fallback
func ExampleDisplayTextDefault() {
	// Error with displayable message - returns the displayable message
	displayErr := errx.NewDisplayable("Invalid email address")
	wrapped := errx.Wrap("validation failed", displayErr)
	msg1 := errx.DisplayTextDefault(wrapped, "An error occurred")
	fmt.Println("With displayable:", msg1)

	// Error without displayable message - returns the default
	regularErr := errors.New("database connection timeout")
	msg2 := errx.DisplayTextDefault(regularErr, "Service temporarily unavailable")
	fmt.Println("Without displayable:", msg2)

	// Nil error - returns empty string
	msg3 := errx.DisplayTextDefault(nil, "Default message")
	fmt.Printf("Nil error: %q\n", msg3)

	// Output:
	// With displayable: Invalid email address
	// Without displayable: Service temporarily unavailable
	// Nil error: ""
}

// ExampleWithAttrs demonstrates adding structured attributes to errors
func ExampleWithAttrs() {
	attrErr := errx.WithAttrs(
		"user_id", 12345,
		"action", "delete",
		"resource", "account",
	)

	attrs := errx.ExtractAttrs(attrErr)
	for _, attr := range attrs {
		fmt.Printf("%s=%v ", attr.Key, attr.Value)
	}

	// Output:
	// user_id=12345 action=delete resource=account
}

// Example_combinedUsage demonstrates combining all features
func Example_combinedUsage() {
	// Define tags
	ErrNotFound := errx.NewSentinel("not found")

	// Create error with attributes
	baseErr := errors.New("record not found in database")
	attrErr := errx.WithAttrs("table", "users", "id", 123)

	// Classify the error
	classifiedErr := errx.Classify(baseErr, attrErr, ErrNotFound)

	// Add displayable message
	displayErr := errx.NewDisplayable("User not found")
	finalErr := errx.Classify(classifiedErr, displayErr)

	// Check classification
	fmt.Println("Is not found:", errors.Is(finalErr, ErrNotFound))

	// Get displayable message
	fmt.Println("Display:", errx.DisplayText(finalErr))

	// Extract attributes
	attrs := errx.ExtractAttrs(finalErr)
	fmt.Printf("Attributes: %d found\n", len(attrs))

	// Output:
	// Is not found: true
	// Display: User not found
	// Attributes: 2 found
}

// ExampleIsDisplayable demonstrates checking if an error has a displayable message
func ExampleIsDisplayable() {
	displayErr := errx.NewDisplayable("Operation failed")
	regularErr := errors.New("internal error")

	fmt.Println("Display error is displayable:", errx.IsDisplayable(displayErr))
	fmt.Println("Regular error is displayable:", errx.IsDisplayable(regularErr))

	// Output:
	// Display error is displayable: true
	// Regular error is displayable: false
}

// ExampleHasAttrs demonstrates checking if an error has attributes
func ExampleHasAttrs() {
	attrErr := errx.WithAttrs("key", "value")
	regularErr := errors.New("no attributes")

	fmt.Println("Attr error has attrs:", errx.HasAttrs(attrErr))
	fmt.Println("Regular error has attrs:", errx.HasAttrs(regularErr))

	// Output:
	// Attr error has attrs: true
	// Regular error has attrs: false
}

// Example_apiHandler demonstrates a practical API error handling pattern
func Example_apiHandler() {
	// Define error tags
	ErrNotFound := errx.NewSentinel("not found")
	ErrValidation := errx.NewSentinel("validation")

	// Simulate an error from the service layer
	var serviceErr error
	serviceErr = errx.NewDisplayable("Email is required")
	serviceErr = errx.Classify(serviceErr, ErrValidation)
	serviceErr = errx.Classify(serviceErr, errx.WithAttrs("field", "email"))

	// API handler logic
	statusCode := 500
	if errors.Is(serviceErr, ErrNotFound) {
		statusCode = 404
	} else if errors.Is(serviceErr, ErrValidation) {
		statusCode = 400
	}

	message := "An error occurred"
	if errx.IsDisplayable(serviceErr) {
		message = errx.DisplayText(serviceErr)
	}

	fmt.Printf("HTTP %d: %s\n", statusCode, message)

	// Log with attributes
	if errx.HasAttrs(serviceErr) {
		attrs := errx.ExtractAttrs(serviceErr)
		fmt.Printf("Attributes: %v\n", attrs)
	}

	// Output:
	// HTTP 400: Email is required
	// Attributes: field=email
}

// ExampleFromAttrMap demonstrates creating attributes from a map
func ExampleFromAttrMap() {
	attrs := map[string]any{
		"user_id":  42,
		"ip":       "192.168.1.1",
		"endpoint": "/api/users",
	}

	attrErr := errx.FromAttrMap(attrs)
	extracted := errx.ExtractAttrs(attrErr)

	fmt.Printf("Total attributes: %d\n", len(extracted))
	fmt.Println("Has attrs:", errx.HasAttrs(attrErr))

	// Output:
	// Total attributes: 3
	// Has attrs: true
}

// ExampleExtractAttrs demonstrates extracting attributes from nested errors
func ExampleExtractAttrs() {
	// Create error with attributes
	baseErr := errors.New("database connection failed")
	attrErr := errx.WithAttrs("host", "localhost", "port", 5432)
	classified := errx.Classify(baseErr, attrErr)

	// Wrap it further
	wrapped := fmt.Errorf("startup failed: %w", classified)

	// Extract attributes from anywhere in the chain
	attrs := errx.ExtractAttrs(wrapped)
	fmt.Printf("Extracted %d attributes\n", len(attrs))
	for _, attr := range attrs {
		fmt.Printf("%s: %v\n", attr.Key, attr.Value)
	}

	// Output:
	// Extracted 2 attributes
	// host: localhost
	// port: 5432
}

// Example_richError demonstrates creating a fully-featured error
func Example_richError() {
	// Define classification hierarchy
	ErrDatabase := errx.NewSentinel("database")
	ErrRetryable := errx.NewSentinel("retryable")
	ErrDBTimeout := errx.NewSentinel("db timeout", ErrDatabase, ErrRetryable)

	// Create base error
	baseErr := errors.New("connection timeout after 30s")

	// Add user-facing message
	displayErr := errx.NewDisplayable("The service is temporarily unavailable")

	// Add structured context
	attrErr := errx.WithAttrs(
		"database", "users",
		"operation", "read",
		"timeout_seconds", 30,
	)

	// Combine everything
	finalErr := errx.Wrap("query execution failed", baseErr, displayErr, attrErr, ErrDBTimeout)

	// Use the error
	fmt.Println("Classification checks:")
	fmt.Println("  Is database error:", errors.Is(finalErr, ErrDatabase))
	fmt.Println("  Is retryable:", errors.Is(finalErr, ErrRetryable))

	fmt.Println("\nUser message:", errx.DisplayText(finalErr))

	fmt.Println("\nLogging context:")
	if errx.HasAttrs(finalErr) {
		attrs := errx.ExtractAttrs(finalErr)
		for _, attr := range attrs {
			fmt.Printf("  %s: %v\n", attr.Key, attr.Value)
		}
	}

	fmt.Println("\nFull error:", finalErr.Error())

	// Output:
	// Classification checks:
	//   Is database error: true
	//   Is retryable: true
	//
	// User message: The service is temporarily unavailable
	//
	// Logging context:
	//   database: users
	//   operation: read
	//   timeout_seconds: 30
	//
	// Full error: query execution failed: connection timeout after 30s
}

// Example_errorChain demonstrates working with error chains
func Example_errorChain() {
	ErrValidation := errx.NewSentinel("validation")

	// Build an error chain
	err1 := errors.New("field is empty")
	err2 := errx.Classify(err1, ErrValidation)
	err3 := errx.Wrap("validation failed", err2)
	err4 := fmt.Errorf("request processing: %w", err3)

	// Check classification through the chain
	fmt.Println("Is validation error:", errors.Is(err4, ErrValidation))

	// Add displayable at any level
	displayErr := errx.NewDisplayable("Please provide a valid value")
	err5 := errx.Classify(err4, displayErr)

	// Display message found through chain
	fmt.Println("Display text:", errx.DisplayText(err5))
	fmt.Println("Full error:", err5.Error())

	// Output:
	// Is validation error: true
	// Display text: Please provide a valid value
	// Full error: request processing: validation failed: field is empty
}

// Example_apiHandlerWithDefault demonstrates using DisplayTextDefault in an API handler
func Example_apiHandlerWithDefault() {
	// Define error sentinels
	ErrNotFound := errx.NewSentinel("not found")
	ErrDatabase := errx.NewSentinel("database")

	// Simulate different error scenarios
	type errorCase struct {
		name string
		err  error
	}

	cases := []errorCase{
		{
			name: "displayable error",
			err:  errx.Wrap("user lookup failed", errx.NewDisplayable("User not found"), ErrNotFound),
		},
		{
			name: "internal error",
			err:  errx.Wrap("query failed", errors.New("connection timeout"), ErrDatabase),
		},
	}

	for _, tc := range cases {
		// Using DisplayTextDefault provides consistent fallback behavior
		message := errx.DisplayTextDefault(tc.err, "An unexpected error occurred")
		fmt.Printf("%s: %s\n", tc.name, message)
	}

	// Output:
	// displayable error: User not found
	// internal error: An unexpected error occurred
}

// ExampleAttrs_ToSlogAttrs demonstrates converting errx.Attrs to slog.Attr for use with LogAttrs
func ExampleAttrs_ToSlogAttrs() {
	// Create an error with attributes
	err := errx.WithAttrs("user_id", 123, "action", "delete", "resource", "account")
	wrappedErr := errx.Wrap("operation failed", err)

	// Extract attributes from the error
	attrs := errx.ExtractAttrs(wrappedErr)

	// Convert to slog.Attr for use with LogAttrs (most efficient)
	slogAttrs := attrs.ToSlogAttrs()

	// Use with slog logger's LogAttrs method
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Remove time for consistent output
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))

	// LogAttrs accepts ...slog.Attr directly
	logger.LogAttrs(context.Background(), slog.LevelError, "operation failed", slogAttrs...)

	// Output:
	// level=ERROR msg="operation failed" user_id=123 action=delete resource=account
}

// ExampleAttrs_ToSlogArgs demonstrates converting errx.Attrs to []any for use with slog convenience methods
func ExampleAttrs_ToSlogArgs() {
	// Create an error with attributes
	err := errx.WithAttrs("user_id", 123, "action", "delete", "resource", "account")
	wrappedErr := errx.Wrap("operation failed", err)

	// Extract attributes from the error
	attrs := errx.ExtractAttrs(wrappedErr)

	// Convert to []any for use with Error/Info/Warn methods
	slogArgs := attrs.ToSlogArgs()

	// Use with slog logger's convenience methods
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Remove time for consistent output
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))

	// Error/Info/Warn methods accept ...any
	logger.Error("operation failed", slogArgs...)

	// Output:
	// level=ERROR msg="operation failed" user_id=123 action=delete resource=account
}
