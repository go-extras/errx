// Package compat provides compatibility functions that accept standard Go error interface
// instead of requiring errx.Classified types. This package is designed for users who
// prefer working with the standard error interface while still benefiting from errx's
// classification and wrapping capabilities.
//
// # Why the Parent Package Uses errx.Classified
//
// The parent errx package uses the Classified interface for several important reasons:
//
//  1. Type Safety: The Classified interface ensures that only valid classification types
//     (sentinels, displayable errors, attributed errors) can be attached to errors.
//     This prevents accidental misuse and provides compile-time guarantees.
//
//  2. Sealed Interface Pattern: The Classified interface uses a marker method (IsClassified)
//     that allows the library to maintain controlled extensibility. External packages can
//     implement Classified, but the library can identify and validate these implementations.
//
//  3. API Stability: By requiring Classified types, the library can evolve its internal
//     implementation without breaking existing code that depends on the classification
//     behavior.
//
//  4. Clear Intent: Using Classified makes it explicit that you're attaching metadata
//     (classifications, displayable messages, attributes) rather than wrapping arbitrary
//     errors in the classification chain.
//
// # How This Package Provides Flexibility
//
// This compat package provides mirror functions that accept standard Go error interface:
//
//   - compat.Wrap(text, cause, classifications...) accepts error classifications
//   - compat.Classify(cause, classifications...) accepts error classifications
//
// These functions internally convert the provided error values to errx.Classified types
// before calling the parent package functions. This conversion is done by wrapping each
// error in an errx.Classified wrapper that preserves the error's identity for errors.Is
// and errors.As checks.
//
// # Tradeoffs
//
// Using this package involves some tradeoffs:
//
// **Advantages:**
//   - Works with any error type, including third-party errors
//   - More flexible for codebases that heavily use standard error interface
//   - Easier migration path from existing error handling code
//
// **Disadvantages:**
//   - Less type safety - you can accidentally pass non-classification errors
//   - Slightly more overhead due to additional wrapping layer
//   - Less clear intent - harder to distinguish classification metadata from regular errors
//
// # Stacktrace Integration
//
// Since stacktrace functionality requires errx.Classified types, this package does NOT
// provide mirror functions for the stacktrace package. This is an intentional design
// decision. If you need stack traces, you have two options:
//
//  1. Use stacktrace.Here() explicitly in your compat calls:
//     err := compat.Wrap("failed", cause, stacktrace.Here())
//
//  2. Use the stacktrace package functions directly:
//     err := stacktrace.Wrap("failed", cause, classification)
//
// # Example Usage
//
//	// Define classification errors (can be any error type)
//	var ErrNotFound = errors.New("not found")
//	var ErrInvalid = errors.New("invalid input")
//
//	// Use compat functions with standard errors
//	func fetchUser(id string) error {
//	    err := db.Query(id)
//	    if err != nil {
//	        return compat.Wrap("failed to fetch user", err, ErrNotFound)
//	    }
//	    return nil
//	}
//
//	// Check classifications using standard errors.Is
//	if errors.Is(err, ErrNotFound) {
//	    // Handle not found case
//	}
package compat

import (
	"github.com/go-extras/errx"
)

// errorWrapper wraps a standard error to make it implement errx.Classified.
// This allows standard errors to be used as classifications in the compat package.
type errorWrapper struct {
	err error
}

func (w *errorWrapper) Error() string {
	return w.err.Error()
}

func (w *errorWrapper) Unwrap() error {
	return w.err
}

func (w *errorWrapper) IsClassified() bool {
	return true
}

// toClassified converts a standard error to errx.Classified.
// If the error is already a Classified, it returns it as-is.
// Otherwise, it wraps the error in an errorWrapper.
func toClassified(err error) errx.Classified {
	if err == nil {
		return nil
	}
	
	// If it's already Classified, return as-is
	if classified, ok := err.(errx.Classified); ok {
		return classified
	}
	
	// Wrap standard error to make it Classified
	return &errorWrapper{err: err}
}

// Wrap wraps an error with additional context text and optional classifications.
// This is a compatibility function that accepts standard Go error interface for
// classifications instead of requiring errx.Classified types.
//
// The function internally converts the provided error classifications to errx.Classified
// types before calling errx.Wrap. This allows you to use any error type as a
// classification, including third-party errors and standard library errors.
//
// If cause is nil, Wrap returns nil.
//
// Example:
//
//	var ErrNotFound = errors.New("not found")
//	var ErrDatabase = errors.New("database error")
//
//	err := db.Query(id)
//	return compat.Wrap("failed to fetch user", err, ErrNotFound, ErrDatabase)
//
//	// Later, check with errors.Is
//	if errors.Is(err, ErrNotFound) {
//	    // Handle not found case
//	}
func Wrap(text string, cause error, classifications ...error) error {
	if cause == nil {
		return nil
	}
	
	// Convert error classifications to Classified
	classified := make([]errx.Classified, 0, len(classifications))
	for _, cls := range classifications {
		if c := toClassified(cls); c != nil {
			classified = append(classified, c)
		}
	}
	
	return errx.Wrap(text, cause, classified...)
}

// Classify attaches one or more classifications to an existing error without adding
// context text. This is a compatibility function that accepts standard Go error
// interface for classifications instead of requiring errx.Classified types.
//
// The function internally converts the provided error classifications to errx.Classified
// types before calling errx.Classify. This allows you to use any error type as a
// classification, including third-party errors and standard library errors.
//
// If cause is nil, Classify returns nil.
//
// Example:
//
//	var ErrValidation = errors.New("validation error")
//
//	err := validateInput(data)
//	return compat.Classify(err, ErrValidation)
//
//	// Later, check with errors.Is
//	if errors.Is(err, ErrValidation) {
//	    // Handle validation error
//	}
func Classify(cause error, classifications ...error) error {
	if cause == nil {
		return nil
	}
	
	// Convert error classifications to Classified
	classified := make([]errx.Classified, 0, len(classifications))
	for _, cls := range classifications {
		if c := toClassified(cls); c != nil {
			classified = append(classified, c)
		}
	}
	
	return errx.Classify(cause, classified...)
}

