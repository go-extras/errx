package stacktrace_test

import (
	"errors"
	"fmt"

	"github.com/go-extras/errx"
	"github.com/go-extras/errx/stacktrace"
)

// ExampleHere demonstrates using Here() to capture stack traces per-error
func ExampleHere() {
	var ErrNotFound = errx.NewSentinel("not found")

	// Capture stack trace at this specific error site
	baseErr := errors.New("user record missing")
	err := errx.Wrap("failed to fetch user", baseErr, ErrNotFound, stacktrace.Here())

	// Extract and display the stack trace
	frames := stacktrace.Extract(err)
	if frames != nil {
		fmt.Printf("Stack trace captured: %d frames\n", len(frames))
		// In real code, you might log all frames
		if len(frames) > 0 {
			fmt.Printf("Top frame: %s\n", frames[0].Function)
		}
	}

	// Output:
	// Stack trace captured: 7 frames
	// Top frame: github.com/go-extras/errx/stacktrace_test.ExampleHere
}

// ExampleWrap demonstrates using stacktrace.Wrap for automatic trace capture
func ExampleWrap() {
	var ErrDatabase = errx.NewSentinel("database error")

	// stacktrace.Wrap automatically captures the stack trace
	baseErr := errors.New("connection timeout")
	err := stacktrace.Wrap("database query failed", baseErr, ErrDatabase)

	// The error works like a normal errx error
	fmt.Println(err.Error())
	fmt.Println("Is database error:", errors.Is(err, ErrDatabase))

	// But also has a stack trace
	frames := stacktrace.Extract(err)
	fmt.Println("Has stack trace:", frames != nil)

	// Output:
	// database query failed: connection timeout
	// Is database error: true
	// Has stack trace: true
}

// ExampleClassify demonstrates using stacktrace.Classify
func ExampleClassify() {
	var ErrRetryable = errx.NewSentinel("retryable error")

	// stacktrace.Classify adds classification and trace without changing the message
	baseErr := errors.New("temporary network failure")
	err := stacktrace.Classify(baseErr, ErrRetryable)

	// Original message is preserved
	fmt.Println(err.Error())

	// But classification and trace are added
	fmt.Println("Is retryable:", errors.Is(err, ErrRetryable))
	fmt.Println("Has trace:", stacktrace.Extract(err) != nil)

	// Output:
	// temporary network failure
	// Is retryable: true
	// Has trace: true
}

// ExampleClassifyNew demonstrates creating and classifying an error with automatic trace capture
func ExampleClassifyNew() {
	var (
		ErrDatabase  = errx.NewSentinel("database error")
		ErrRetryable = errx.NewSentinel("retryable error")
	)

	// Create a new error, classify it, and capture stack trace in one step
	err := stacktrace.ClassifyNew("connection timeout", ErrDatabase, ErrRetryable)

	fmt.Println(err.Error())
	fmt.Println("Is database error:", errors.Is(err, ErrDatabase))
	fmt.Println("Is retryable:", errors.Is(err, ErrRetryable))
	fmt.Println("Has trace:", stacktrace.Extract(err) != nil)

	// Output:
	// connection timeout
	// Is database error: true
	// Is retryable: true
	// Has trace: true
}

// ExampleClassifyNew_withDisplayable demonstrates ClassifyNew with displayable errors
func ExampleClassifyNew_withDisplayable() {
	var ErrNotFound = errx.NewSentinel("not found")
	displayErr := errx.NewDisplayable("The requested resource was not found")

	// Create error with classification, displayable message, and stack trace
	err := stacktrace.ClassifyNew("user record missing from database", ErrNotFound, displayErr)

	// Full error for logging
	fmt.Println("Full error:", err.Error())

	// User-facing message
	fmt.Println("Display text:", errx.DisplayText(err))

	// Classification check
	fmt.Println("Is not found:", errors.Is(err, ErrNotFound))

	// Stack trace available
	fmt.Println("Has trace:", stacktrace.Extract(err) != nil)

	// Output:
	// Full error: user record missing from database
	// Display text: The requested resource was not found
	// Is not found: true
	// Has trace: true
}

// ExampleExtract demonstrates extracting and formatting stack traces
func ExampleExtract() {
	// Create an error with a stack trace
	err := stacktrace.Wrap("operation failed", errors.New("base error"))

	// Extract the stack trace
	frames := stacktrace.Extract(err)
	if frames != nil {
		fmt.Printf("Stack trace (%d frames):\n", len(frames))
		for i, frame := range frames {
			if i >= 3 { // Limit output for example
				fmt.Println("  ...")
				break
			}
			fmt.Printf("  %s:%d\n", frame.Function, frame.Line)
		}
	}

	// Output:
	// Stack trace (7 frames):
	//   github.com/go-extras/errx/stacktrace_test.ExampleExtract:129
	//   testing.runExample:63
	//   testing.runExamples:41
	//   ...
}

// ExampleExtract_noTrace demonstrates Extract returning nil for errors without traces
func ExampleExtract_noTrace() {
	// Regular error without stack trace
	err := errors.New("simple error")

	frames := stacktrace.Extract(err)
	fmt.Printf("Stack trace: %v\n", frames)

	// Output:
	// Stack trace: []
}

// ExampleFrame_String demonstrates formatting a stack frame
func ExampleFrame_String() {
	frame := stacktrace.Frame{
		File:     "/home/user/project/main.go",
		Line:     42,
		Function: "main.processRequest",
	}

	fmt.Println(frame.String())

	// Output:
	// /home/user/project/main.go:42 main.processRequest
}

// Example_integration demonstrates combining stacktrace with other errx features
func Example_integration() {
	var ErrNotFound = errx.NewSentinel("not found")

	// Combine stack traces with displayable errors and attributes
	displayErr := errx.NewDisplayable("User not found")
	attrErr := errx.Attrs("user_id", "12345", "action", "fetch")

	err := stacktrace.Wrap("failed to get user profile",
		errx.Classify(displayErr, ErrNotFound, attrErr))

	// All features work together
	fmt.Println("Error:", err.Error())
	fmt.Println("Displayable:", errx.DisplayText(err))
	fmt.Println("Is not found:", errors.Is(err, ErrNotFound))
	fmt.Println("Has attributes:", errx.HasAttrs(err))
	fmt.Println("Has stack trace:", stacktrace.Extract(err) != nil)

	// Output:
	// Error: failed to get user profile: User not found
	// Displayable: User not found
	// Is not found: true
	// Has attributes: true
	// Has stack trace: true
}
