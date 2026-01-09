// Package errx provides error handling utilities with classification sentinels and displayable messages.
// It enables wrapping errors with classification sentinels that can be checked using errors.Is.
//
// # Core Concepts
//
// The package provides three main error categories:
//
// Classification Sentinels: For programmatic error checking using errors.Is. These sentinels
// are used to identify specific error conditions in code, such as "not found" or
// "access denied". The sentinel text is intentionally NOT visible in the error
// message chain to keep error messages clean.
//
// Displayable Errors: For user-facing error messages. These errors represent messages that
// are safe and appropriate to display directly to end users. They can be extracted
// from any error chain using DisplayText, which returns just the displayable message
// without internal context.
//
// # When to Use
//
// Use Classification sentinels (NewSentinel) when:
//   - You need to check for specific error conditions programmatically
//   - The error type is more important than the error message
//   - You want to attach error classifications without polluting error messages
//
// Use Displayable errors (NewDisplayable) when:
//   - You need to return user-friendly error messages from APIs
//   - The error should be safe to display to end users
//   - You want to separate internal error context from user messages
//
// Use Wrap when:
//   - You need to add context to an error
//   - You want to attach classification sentinels to existing errors
//   - You're propagating errors up the call stack
//
// Use Classify when:
//   - You want to attach classification sentinels WITHOUT adding context text
//   - You need to mark an error for programmatic checking but keep the original message
//   - You're at a layer where the error message is already sufficient
//
// # Example Usage
//
//	// Define classification sentinels
//	var ErrNotFound = errx.NewSentinel("resource not found")
//
//	// Create displayable error
//	func validateInput(email string) error {
//	    if !isValid(email) {
//	        return errx.NewDisplayable("Invalid email format")
//	    }
//	    return nil
//	}
//
//	// Wrap with context and sentinels
//	func fetchUser(id string) error {
//	    displayErr := errx.NewDisplayable("User not found")
//	    return errx.Wrap("failed to fetch user", displayErr, ErrNotFound)
//	}
//
//	// Classify without adding context
//	func processRecord(err error) error {
//	    return errx.Classify(err, ErrNotFound)  // Preserves original message
//	}
//
//	// Check for specific errors
//	if errors.Is(err, ErrNotFound) {
//	    // Handle not found case
//	}
//
//	// Extract displayable message
//	if errx.IsDisplayable(err) {
//	    return errx.DisplayText(err)  // Returns: "User not found"
//	}
package errx

import (
	"errors"
	"fmt"
)

// Classified is an interface for errors that can be classified.
// This interface can be implemented by external packages to extend the library.
// Internally, there are four categories of Classified implementations:
//
//  1. Sentinel errors (*sentinel): Pure markers for programmatic error
//     checking using errors.Is.
//
//  2. Displayable errors (*displayable): Errors with messages safe to display to
//     end users.
//
//  3. Attributed errors (*attributed): Errors that carry structured metadata (key-value pairs)
//     for logging and debugging.
//
//  4. Traced errors (stacktrace.*traced): Errors that capture stack traces (in stacktrace subpackage).
//
// The IsClassified() method serves as a type marker to distinguish Classified errors
// from regular Go errors. All implementations should return true.
type Classified interface {
	error
	// IsClassified is a marker method that identifies this error as a Classified error.
	// It should always return true for valid Classified implementations.
	// This method allows programmatic distinction between regular errors and errx Classified errors.
	IsClassified() bool
}

// Ensure sentinel implements Classified interface
var _ Classified = (*sentinel)(nil)

type sentinel struct {
	text    string
	parents []Classified
}

func (s *sentinel) Error() string {
	return s.text
}

func (s *sentinel) Unwrap() error {
	if len(s.parents) == 0 {
		return nil
	}
	// Return first parent for standard unwrapping
	return s.parents[0]
}

func (s *sentinel) Is(target error) bool {
	// Check if target is this sentinel
	if target == s {
		return true
	}

	// Check if target matches any parent
	for _, parent := range s.parents {
		if errors.Is(parent, target) {
			return true
		}
	}

	return false
}

// As checks if the target matches any parent errors.
func (s *sentinel) As(target any) bool {
	// Check parents via errors.As
	for _, parent := range s.parents {
		if errors.As(parent, target) {
			return true
		}
	}
	return false
}

// IsClassified implements the Classified interface marker method.
// It always returns true to identify this as a Classified error.
func (*sentinel) IsClassified() bool {
	return true
}

// NewSentinel creates a new classification sentinel with the given text.
// Classification sentinels are used for programmatic error checking with errors.Is.
// The sentinel text is intentionally not visible in error message chains.
//
// Optional parent sentinels can be provided to create a hierarchy. A sentinel with parents
// will match itself and all of its parents via errors.Is.
//
// # Circular References
//
// WARNING: Creating circular parent references will cause infinite loops when using errors.Is.
// It is the caller's responsibility to avoid circular hierarchies. For example:
//
//	// DON'T DO THIS - creates a circular reference
//	parent := errx.NewSentinel("parent")
//	child := errx.NewSentinel("child", parent)
//	// Then somehow making parent reference child would create a cycle
//
// The package does not detect or prevent circular references for performance reasons.
// Always ensure your sentinel hierarchies form a directed acyclic graph (DAG).
//
// Example:
//
//	// Simple sentinel
//	ErrDatabase := errx.NewSentinel("database error")
//
//	// Sentinel with parent (hierarchical)
//	ErrTimeout := errx.NewSentinel("timeout", ErrDatabase)
//	// Now ErrTimeout will match both itself and ErrDatabase
//
//	// Sentinel with multiple parents
//	ErrCritical := errx.NewSentinel("critical")
//	ErrDatabaseCritical := errx.NewSentinel("critical database error", ErrDatabase, ErrCritical)
//	// Matches itself, ErrDatabase, and ErrCritical
func NewSentinel(text string, parents ...Classified) Classified {
	if len(parents) == 0 {
		return &sentinel{text: text}
	}
	return &sentinel{text: text, parents: parents}
}

// Wrap wraps an error with additional context text and optional classification sentinels.
// The attached classification sentinels can be used later to identify the error using errors.Is,
// as well as add displayable errors.
// If err is nil, Wrap returns nil.
//
// If no classifications are provided, Wrap behaves like fmt.Errorf with %w,
// avoiding unnecessary carrier allocation.
func Wrap(text string, cause error, classifications ...Classified) error {
	if cause == nil {
		return nil
	}
	if len(classifications) == 0 {
		return fmt.Errorf("%s: %w", text, cause)
	}
	return fmt.Errorf("%s: %w", text, classify(cause, classifications...))
}

// Classify attaches one or more classification sentinels to an existing error.
// The attached classification sentinels can be used later to identify the error using errors.Is.
// If err is nil, Classify returns nil.
//
// Example:
//
//	var ErrNotFound = errx.NewSentinel("resource not found")
//
//	baseErr := errors.New("resource missing")
//	classifiedErr := errx.Classify(baseErr, ErrNotFound)
//
//	fmt.Println(errors.Is(classifiedErr, ErrNotFound)) // Output: true
func Classify(cause error, classifications ...Classified) error {
	return classify(cause, classifications...)
}

func classify(cause error, classifications ...Classified) error {
	if cause == nil {
		return nil
	}
	return &carrier{classifications: classifications, cause: cause}
}

type carrier struct {
	classifications []Classified
	cause           error
}

func (c *carrier) Error() string {
	// IMPORTANT: classification sentinel text is intentionally NOT shown here
	return c.cause.Error()
}

func (c *carrier) Unwrap() error {
	return c.cause
}

func (c *carrier) Is(target error) bool {
	if errors.Is(c.cause, target) {
		return true
	}

	for _, cls := range c.classifications {
		if errors.Is(cls, target) {
			return true
		}
	}

	return false
}

func (c *carrier) As(target any) bool {
	if errors.As(c.cause, target) {
		return true
	}

	for _, cls := range c.classifications {
		if errors.As(cls, target) {
			return true
		}
	}

	return false
}
