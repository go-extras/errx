package errx

import (
	"errors"
)

// Ensure displayable implements Classified interface
var _ Classified = (*displayable)(nil)

// displayable is a specialized classification sentinel intended for user-facing error messages.
// Unlike regular errors or sentinels, displayable errors represent messages that are safe and
// appropriate to display directly to end users.
type displayable struct {
	*sentinel
}

// NewDisplayable creates a new displayable error with the given message.
// Displayable errors are intended for error messages that should be displayed to end users.
// These errors can be extracted from an error chain using DisplayText.
//
// Example:
//
//	err := NewDisplayable("Invalid email address")
//	wrapped := fmt.Errorf("validation failed: %w", err)
//	msg := DisplayText(wrapped)  // Returns: "Invalid email address"
func NewDisplayable(message string) Classified {
	return &displayable{
		sentinel: &sentinel{text: message},
	}
}

// IsClassified implements the Classified interface marker method.
// It always returns true to identify this as a Classified error.
func (*displayable) IsClassified() bool {
	return true
}

// IsDisplayable reports whether any error in err's chain is a displayable error.
// It traverses the error chain using errors.As to find a displayable error.
//
// This is useful for conditionally handling displayable errors differently
// from internal errors.
//
// Example:
//
//	if IsDisplayable(err) {
//	    // Safe to display to user
//	    return DisplayText(err)
//	}
//	// Internal error, log details but show generic message
//	log.Error(err)
//	return "An error occurred"
func IsDisplayable(err error) bool {
	if err == nil {
		return false
	}

	var dErr *displayable
	return errors.As(err, &dErr)
}

// DisplayText extracts the first displayable error message from an error chain.
// If a displayable error is found anywhere in the error chain (using errors.As),
// it returns just the displayable error's message without any wrapper context.
// If no displayable error is found, it returns the full error message.
//
// If multiple displayable errors exist in the chain, the message returned is the
// first one discovered via error traversal. This selection is based on the
// traversal order and does not imply any precedence semantics.
//
// This is useful for APIs that need to return user-friendly error messages
// while maintaining detailed error context internally.
//
// Example:
//
//	displayErr := NewDisplayable("Resource not found")
//	wrapped := Wrap("failed to fetch resource", displayErr, ErrNotFound)
//	deepWrapped := fmt.Errorf("operation failed: %w", wrapped)
//
//	// Returns: "Resource not found" (extracts just the displayable message)
//	msg := DisplayText(deepWrapped)
//
//	// For errors without displayable messages, returns full message
//	regularErr := errors.New("internal error")
//	msg := DisplayText(regularErr)  // Returns: "internal error"
func DisplayText(err error) string {
	if err == nil {
		return ""
	}

	var dErr *displayable
	if errors.As(err, &dErr) {
		return dErr.Error()
	}

	return err.Error()
}

// DisplayTextDefault extracts the first displayable error message from an error chain,
// or returns a default message if no displayable error is found.
//
// This function behaves like DisplayText, but instead of returning the full error message
// when no displayable error is found, it returns the provided default message.
// This is useful for providing consistent, user-friendly fallback messages.
//
// If err is nil, it returns an empty string (not the default message).
//
// Example:
//
//	// Error with displayable message
//	displayErr := NewDisplayable("Invalid email format")
//	wrapped := Wrap("validation failed", displayErr)
//	msg := DisplayTextDefault(wrapped, "An error occurred")
//	// Returns: "Invalid email format"
//
//	// Error without displayable message
//	regularErr := errors.New("database connection timeout")
//	msg := DisplayTextDefault(regularErr, "Service temporarily unavailable")
//	// Returns: "Service temporarily unavailable"
//
//	// Nil error
//	msg := DisplayTextDefault(nil, "An error occurred")
//	// Returns: ""
func DisplayTextDefault(err error, def string) string {
	if err == nil {
		return ""
	}

	if IsDisplayable(err) {
		return DisplayText(err)
	}

	return def
}
