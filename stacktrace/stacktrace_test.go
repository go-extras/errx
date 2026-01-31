package stacktrace_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/go-extras/errx"
	"github.com/go-extras/errx/stacktrace"
)

// TestHere verifies that Here() captures stack traces correctly
func TestHere(t *testing.T) {
	// Create an error with a stack trace
	baseErr := errors.New("base error")
	err := errx.Wrap("context", baseErr, stacktrace.Here())

	// Extract the stack trace
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Fatal("Expected stack trace, got nil")
	}

	if len(frames) == 0 {
		t.Fatal("Expected non-empty stack trace")
	}

	// Verify that the first frame contains this test function
	found := false
	for _, frame := range frames {
		if strings.Contains(frame.Function, "TestHere") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected stack trace to contain TestHere function")
	}
}

// TestExtractNil verifies that Extract returns nil for nil errors
func TestExtractNil(t *testing.T) {
	frames := stacktrace.Extract(nil)
	if frames != nil {
		t.Errorf("Expected nil for nil error, got %v", frames)
	}
}

// TestExtractNoTrace verifies that Extract returns nil for errors without traces
func TestExtractNoTrace(t *testing.T) {
	err := errors.New("no trace")
	frames := stacktrace.Extract(err)
	if frames != nil {
		t.Errorf("Expected nil for error without trace, got %v", frames)
	}
}

// TestExtractFromWrappedError verifies that Extract finds traces in wrapped errors
func TestExtractFromWrappedError(t *testing.T) {
	baseErr := errors.New("base")
	traced := errx.Classify(baseErr, stacktrace.Here())
	wrapped := errx.Wrap("outer", traced)

	frames := stacktrace.Extract(wrapped)
	if frames == nil {
		t.Fatal("Expected to find stack trace in wrapped error")
	}

	if len(frames) == 0 {
		t.Error("Expected non-empty stack trace")
	}
}

// TestWrap verifies that stacktrace.Wrap automatically captures traces
func TestWrap(t *testing.T) {
	baseErr := errors.New("base error")
	err := stacktrace.Wrap("operation failed", baseErr)

	// Verify the error message
	expected := "operation failed: base error"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}

	// Verify stack trace was captured
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Fatal("Expected stack trace from Wrap, got nil")
	}

	// Verify the trace contains this test function
	found := false
	for _, frame := range frames {
		if strings.Contains(frame.Function, "TestWrap") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected stack trace to contain TestWrap function")
	}
}

// TestWrapNil verifies that stacktrace.Wrap returns nil for nil errors
func TestWrapNil(t *testing.T) {
	err := stacktrace.Wrap("context", nil)
	if err != nil {
		t.Errorf("Expected nil for nil cause, got %v", err)
	}
}

// TestWrapWithClassifications verifies that Wrap works with additional classifications
func TestWrapWithClassifications(t *testing.T) {
	var ErrNotFound = errx.NewSentinel("not found")
	baseErr := errors.New("base")
	err := stacktrace.Wrap("failed", baseErr, ErrNotFound)

	// Verify classification
	if !errors.Is(err, ErrNotFound) {
		t.Error("Expected error to match ErrNotFound sentinel")
	}

	// Verify stack trace
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Error("Expected stack trace")
	}
}

// TestClassify verifies that stacktrace.Classify automatically captures traces
func TestClassify(t *testing.T) {
	var ErrDatabase = errx.NewSentinel("database error")
	baseErr := errors.New("connection failed")
	err := stacktrace.Classify(baseErr, ErrDatabase)

	// Verify classification
	if !errors.Is(err, ErrDatabase) {
		t.Error("Expected error to match ErrDatabase sentinel")
	}

	// Verify original message is preserved
	if err.Error() != "connection failed" {
		t.Errorf("Expected original message, got %q", err.Error())
	}

	// Verify stack trace
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Error("Expected stack trace from Classify")
	}
}

// TestClassifyNil verifies that stacktrace.Classify returns nil for nil errors
func TestClassifyNil(t *testing.T) {
	err := stacktrace.Classify(nil)
	if err != nil {
		t.Errorf("Expected nil for nil cause, got %v", err)
	}
}

// TestFrameString verifies the Frame.String() method
func TestFrameString(t *testing.T) {
	frame := stacktrace.Frame{
		File:     "/path/to/file.go",
		Line:     42,
		Function: "github.com/example/pkg.Function",
	}

	expected := "/path/to/file.go:42 github.com/example/pkg.Function"
	if frame.String() != expected {
		t.Errorf("Expected %q, got %q", expected, frame.String())
	}
}

// TestMultipleTraces verifies that only the first trace is extracted
func TestMultipleTraces(t *testing.T) {
	baseErr := errors.New("base")
	err1 := errx.Classify(baseErr, stacktrace.Here())
	err2 := errx.Wrap("outer", err1, stacktrace.Here())

	frames := stacktrace.Extract(err2)
	if frames == nil {
		t.Fatal("Expected stack trace")
	}

	// The first trace found should be from the outer wrap
	// (errors.As finds the first match in the chain)
	found := false
	for _, frame := range frames {
		if strings.Contains(frame.Function, "TestMultipleTraces") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected stack trace to contain TestMultipleTraces")
	}
}

// TestIntegrationWithDisplayable verifies stacktrace works with displayable errors
func TestIntegrationWithDisplayable(t *testing.T) {
	displayErr := errx.NewDisplayable("User not found")
	err := stacktrace.Wrap("fetch failed", displayErr)

	// Verify displayable message
	if !errx.IsDisplayable(err) {
		t.Error("Expected error to be displayable")
	}
	if errx.DisplayText(err) != "User not found" {
		t.Errorf("Expected displayable text 'User not found', got %q", errx.DisplayText(err))
	}

	// Verify stack trace
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Error("Expected stack trace")
	}
}

// TestIntegrationWithAttrs verifies stacktrace works with attributed errors
func TestIntegrationWithAttrs(t *testing.T) {
	attrErr := errx.Attrs("user_id", 123, "action", "delete")
	err := stacktrace.Wrap("operation failed", attrErr)

	// Verify attributes
	if !errx.HasAttrs(err) {
		t.Error("Expected error to have attributes")
	}
	attrs := errx.ExtractAttrs(err)
	if len(attrs) != 2 {
		t.Errorf("Expected 2 attributes, got %d", len(attrs))
	}

	// Verify stack trace
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Error("Expected stack trace")
	}
}

// TestComplexErrorChain verifies stacktrace works in complex error chains
func TestComplexErrorChain(t *testing.T) {
	var ErrNotFound = errx.NewSentinel("not found")

	// Build a complex error chain
	baseErr := errors.New("database error")
	attrErr := errx.Attrs("table", "users", "id", 42)
	displayErr := errx.NewDisplayable("Record not found")

	err := stacktrace.Wrap("query failed",
		errx.Wrap("fetch user",
			errx.Classify(baseErr, attrErr, displayErr, ErrNotFound)))

	// Verify all features work together
	if !errors.Is(err, ErrNotFound) {
		t.Error("Expected error to match ErrNotFound")
	}
	if !errx.IsDisplayable(err) {
		t.Error("Expected error to be displayable")
	}
	if !errx.HasAttrs(err) {
		t.Error("Expected error to have attributes")
	}

	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Error("Expected stack trace")
	}
}

// TestClassifyNew verifies that stacktrace.ClassifyNew creates and classifies errors with traces
func TestClassifyNew(t *testing.T) {
	var ErrDatabase = errx.NewSentinel("database error")
	err := stacktrace.ClassifyNew("connection timeout", ErrDatabase)

	// Verify error message
	if err.Error() != "connection timeout" {
		t.Errorf("Expected 'connection timeout', got %q", err.Error())
	}

	// Verify classification
	if !errors.Is(err, ErrDatabase) {
		t.Error("Expected error to match ErrDatabase sentinel")
	}

	// Verify stack trace was captured
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Fatal("Expected stack trace from ClassifyNew, got nil")
	}

	// Verify the trace contains this test function
	found := false
	for _, frame := range frames {
		if strings.Contains(frame.Function, "TestClassifyNew") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected stack trace to contain TestClassifyNew function")
	}
}

// TestClassifyNewMultipleClassifications verifies ClassifyNew with multiple classifications
func TestClassifyNewMultipleClassifications(t *testing.T) {
	var (
		ErrDatabase  = errx.NewSentinel("database error")
		ErrRetryable = errx.NewSentinel("retryable error")
	)

	err := stacktrace.ClassifyNew("temporary failure", ErrDatabase, ErrRetryable)

	// Verify both classifications
	if !errors.Is(err, ErrDatabase) {
		t.Error("Expected error to match ErrDatabase")
	}
	if !errors.Is(err, ErrRetryable) {
		t.Error("Expected error to match ErrRetryable")
	}

	// Verify stack trace
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Error("Expected stack trace")
	}
}

// TestClassifyNewWithDisplayable verifies ClassifyNew with displayable errors
func TestClassifyNewWithDisplayable(t *testing.T) {
	var ErrNotFound = errx.NewSentinel("not found")
	displayErr := errx.NewDisplayable("Resource not found")

	err := stacktrace.ClassifyNew("user record missing", ErrNotFound, displayErr)

	// Verify error message
	if err.Error() != "user record missing" {
		t.Errorf("Expected 'user record missing', got %q", err.Error())
	}

	// Verify classification
	if !errors.Is(err, ErrNotFound) {
		t.Error("Expected error to match ErrNotFound")
	}

	// Verify displayable
	if !errx.IsDisplayable(err) {
		t.Error("Expected error to be displayable")
	}
	if errx.DisplayText(err) != "Resource not found" {
		t.Errorf("Expected displayable text 'Resource not found', got %q", errx.DisplayText(err))
	}

	// Verify stack trace
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Error("Expected stack trace")
	}
}

// TestClassifyNewWithAttributes verifies ClassifyNew with attributes
func TestClassifyNewWithAttributes(t *testing.T) {
	var ErrDatabase = errx.NewSentinel("database error")
	attrErr := errx.Attrs("query", "SELECT * FROM users", "timeout_ms", 5000)

	err := stacktrace.ClassifyNew("query timeout", ErrDatabase, attrErr)

	// Verify classification
	if !errors.Is(err, ErrDatabase) {
		t.Error("Expected error to match ErrDatabase")
	}

	// Verify attributes
	if !errx.HasAttrs(err) {
		t.Error("Expected error to have attributes")
	}

	attrs := errx.ExtractAttrs(err)
	if len(attrs) != 2 {
		t.Errorf("Expected 2 attributes, got %d", len(attrs))
	}

	// Verify stack trace
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Error("Expected stack trace")
	}
}

// TestClassifyNewNoClassifications verifies ClassifyNew without classifications
func TestClassifyNewNoClassifications(t *testing.T) {
	err := stacktrace.ClassifyNew("simple error")

	// Verify error message
	if err.Error() != "simple error" {
		t.Errorf("Expected 'simple error', got %q", err.Error())
	}

	// Verify stack trace is still captured
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Error("Expected stack trace even without classifications")
	}
}
