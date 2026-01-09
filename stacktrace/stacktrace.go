// Package stacktrace provides optional stack trace support for errx errors.
//
// This package extends errx with stack trace capabilities while keeping the core
// errx package minimal and zero-dependency. It offers two usage patterns:
//
//  1. Per-error opt-in using Here() as a Classified:
//     err := errx.Wrap("context", cause, ErrNotFound, stacktrace.Here())
//
//  2. Convenience functions that automatically capture traces:
//     err := stacktrace.Wrap("context", cause, ErrNotFound)
//
// Stack traces can be extracted from any error in the chain using Extract():
//
//	frames := stacktrace.Extract(err)
//	for _, frame := range frames {
//	    fmt.Printf("%s:%d %s\n", frame.File, frame.Line, frame.Function)
//	}
package stacktrace

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/go-extras/errx"
)

// Frame represents a single stack frame with file, line, and function information.
type Frame struct {
	File     string // Full path to the source file
	Line     int    // Line number in the source file
	Function string // Fully qualified function name
}

// String returns a formatted representation of the frame.
func (f Frame) String() string {
	return fmt.Sprintf("%s:%d %s", f.File, f.Line, f.Function)
}

// traced is an internal type that implements errx.Classified and captures stack trace.
type traced struct {
	pcs []uintptr // Program counters captured from the stack
}

// Error returns a string representation of the traced error.
// This is primarily for debugging; the trace itself is accessed via Extract().
func (t *traced) Error() string {
	frames := t.frames()
	if len(frames) == 0 {
		return "(empty stack trace)"
	}
	return fmt.Sprintf("stack trace: %d frames", len(frames))
}

// frames converts the stored program counters into Frame structs.
// This is done lazily to avoid the cost of frame resolution unless needed.
func (t *traced) frames() []Frame {
	if len(t.pcs) == 0 {
		return nil
	}

	frames := runtime.CallersFrames(t.pcs)
	var result []Frame
	for {
		frame, more := frames.Next()
		result = append(result, Frame{
			File:     frame.File,
			Line:     frame.Line,
			Function: frame.Function,
		})
		if !more {
			break
		}
	}
	return result
}

// IsClassified implements the errx.Classified interface marker method.
// It always returns true to identify this as a Classified error.
func (*traced) IsClassified() bool {
	return true
}

// Here captures the current stack trace and returns it as an errx.Classified.
// It can be used with errx.Wrap() or errx.Classify() to attach stack traces to errors.
//
// The stack trace is captured starting from the caller of Here(), skipping the
// Here() function itself and the runtime.Callers call.
//
// Example:
//
//	err := errx.Wrap("operation failed", cause, ErrNotFound, stacktrace.Here())
//
// The captured stack trace can later be extracted using Extract().
func Here() errx.Classified {
	return captureStack(2) // Skip Here() and runtime.Callers
}

// captureStack captures the current stack trace with the specified skip count.
// skip indicates how many stack frames to skip (0 = captureStack itself).
func captureStack(skip int) *traced {
	const maxDepth = 32 // Reasonable default depth limit
	pcs := make([]uintptr, maxDepth)
	n := runtime.Callers(skip+1, pcs) // +1 to skip captureStack itself
	return &traced{pcs: pcs[:n]}
}

// Extract returns stack frames from the first traced error found in the error chain.
// It traverses the entire error chain looking for a traced error and returns its frames.
//
// Returns nil if the error is nil or does not contain any stack trace.
//
// Example:
//
//	frames := stacktrace.Extract(err)
//	if frames != nil {
//	    for _, frame := range frames {
//	        fmt.Printf("%s:%d %s\n", frame.File, frame.Line, frame.Function)
//	    }
//	}
func Extract(err error) []Frame {
	if err == nil {
		return nil
	}

	// Use errors.As to find the first traced error in the chain
	var t *traced
	if errors.As(err, &t) {
		return t.frames()
	}

	return nil
}

// Wrap wraps an error with additional context text and optional classifications,
// automatically capturing a stack trace at the call site.
//
// This is a convenience function equivalent to:
//
//	errx.Wrap(text, cause, append(classifications, stacktrace.Here())...)
//
// If cause is nil, Wrap returns nil.
//
// Example:
//
//	err := stacktrace.Wrap("failed to process order", cause, ErrNotFound)
func Wrap(text string, cause error, classifications ...errx.Classified) error {
	if cause == nil {
		return nil
	}
	// Capture stack with skip=2 to skip Wrap() and runtime.Callers
	trace := captureStack(2)
	classifications = append(classifications, trace)
	return errx.Wrap(text, cause, classifications...)
}

// Classify attaches one or more classifications to an error, automatically
// capturing a stack trace at the call site.
//
// This is a convenience function equivalent to:
//
//	errx.Classify(cause, append(classifications, stacktrace.Here())...)
//
// If cause is nil, Classify returns nil.
//
// Example:
//
//	err := stacktrace.Classify(cause, ErrNotFound)
func Classify(cause error, classifications ...errx.Classified) error {
	if cause == nil {
		return nil
	}
	// Capture stack with skip=2 to skip Classify() and runtime.Callers
	trace := captureStack(2)
	classifications = append(classifications, trace)
	return errx.Classify(cause, classifications...)
}
