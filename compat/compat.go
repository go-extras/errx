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
	"errors"

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

// Is reports whether the wrapped error matches target via errors.Is. This makes
// errorWrapper transparent to standard error matching: for example, if a caller
// passes a sentinel `var ErrNotFound = errors.New("not found")` as a classification,
// downstream errors.Is(returned, ErrNotFound) still works after the value has been
// wrapped to satisfy errx.Classified.
func (w *errorWrapper) Is(target error) bool {
	return errors.Is(w.err, target)
}

// As delegates to errors.As on the wrapped error, so callers can extract typed
// errors that were originally provided as classifications without having to know
// they were silently wrapped by the compat layer.
func (w *errorWrapper) As(target any) bool {
	return errors.As(w.err, target)
}

func (*errorWrapper) IsClassified() bool {
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

// toClassifiedSlice converts a variadic []error into []errx.Classified,
// skipping nil entries. It returns nil (not a zero-length slice) when the
// input is empty or contains only nil entries, so callers avoid an
// unnecessary allocation in the common no-classification path.
func toClassifiedSlice(in []error) []errx.Classified {
	if len(in) == 0 {
		return nil
	}
	var out []errx.Classified
	for _, cls := range in {
		if c := toClassified(cls); c != nil {
			if out == nil {
				out = make([]errx.Classified, 0, len(in))
			}
			out = append(out, c)
		}
	}
	return out
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

	return errx.Wrap(text, cause, toClassifiedSlice(classifications)...)
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

	return errx.Classify(cause, toClassifiedSlice(classifications)...)
}

// ClassifyNew creates a new error with the given text and immediately classifies it
// with one or more classifications. This is a convenience function equivalent to
// calling compat.Classify(errors.New(text), classifications...).
//
// This function accepts standard Go error interface for classifications instead of
// requiring errx.Classified types, making it compatible with any error type including
// third-party errors and standard library errors.
//
// This function is useful when you want to create a new error and classify it in a
// single step, reducing verbosity compared to the two-step approach.
//
// Example:
//
//	var ErrNotFound = errors.New("not found")
//	var ErrDatabase = errors.New("database error")
//
//	// Instead of:
//	// err := compat.Classify(errors.New("user record missing"), ErrNotFound, ErrDatabase)
//
//	// You can write:
//	err := compat.ClassifyNew("user record missing", ErrNotFound, ErrDatabase)
//
//	fmt.Println(err.Error())                        // Output: user record missing
//	fmt.Println(errors.Is(err, ErrNotFound))        // Output: true
//	fmt.Println(errors.Is(err, ErrDatabase))        // Output: true
func ClassifyNew(text string, classifications ...error) error {
	return errx.ClassifyNew(text, toClassifiedSlice(classifications)...)
}
