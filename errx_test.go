package errx_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-extras/errx"
)

// TestNewSentinel tests creating a new classification tag
func TestNewSentinel(t *testing.T) {
	tag := errx.NewSentinel("test error")

	if tag == nil {
		t.Fatal("expected non-nil tag")
	}
	if tag.Error() != "test error" {
		t.Errorf("expected 'test error', got %q", tag.Error())
	}
}

// TestWrapWithTag tests wrapping with a classification tag
func TestWrapWithTag(t *testing.T) {
	tag := errx.NewSentinel("tag error")
	baseErr := errors.New("base error")
	wrapped := errx.Wrap("context", baseErr, tag)

	if wrapped == nil {
		t.Fatal("expected non-nil wrapped error")
	}
	if wrapped.Error() != "context: base error" {
		t.Errorf("expected 'context: base error', got %q", wrapped.Error())
	}
	if !errors.Is(wrapped, tag) {
		t.Error("expected error to match tag")
	}
	if !errors.Is(wrapped, baseErr) {
		t.Error("expected error to match base error")
	}
}

// TestWrapWithMultipleTags tests wrapping with multiple classification tags
func TestWrapWithMultipleTags(t *testing.T) {
	tag1 := errx.NewSentinel("tag1")
	tag2 := errx.NewSentinel("tag2")
	baseErr := errors.New("base error")
	wrapped := errx.Wrap("context", baseErr, tag1, tag2)

	if wrapped == nil {
		t.Fatal("expected non-nil wrapped error")
	}
	if wrapped.Error() != "context: base error" {
		t.Errorf("expected 'context: base error', got %q", wrapped.Error())
	}
	if !errors.Is(wrapped, tag1) {
		t.Error("expected error to match tag1")
	}
	if !errors.Is(wrapped, tag2) {
		t.Error("expected error to match tag2")
	}
	if !errors.Is(wrapped, baseErr) {
		t.Error("expected error to match base error")
	}
}

// TestWrapWithoutTag tests wrapping without tags
func TestWrapWithoutTag(t *testing.T) {
	baseErr := errors.New("base error")
	wrapped := errx.Wrap("context", baseErr)

	if wrapped == nil {
		t.Fatal("expected non-nil wrapped error")
	}
	if wrapped.Error() != "context: base error" {
		t.Errorf("expected 'context: base error', got %q", wrapped.Error())
	}
	if !errors.Is(wrapped, baseErr) {
		t.Error("expected error to match base error")
	}
}

// TestWrapNilError tests wrapping nil error
func TestWrapNilError(t *testing.T) {
	tag := errx.NewSentinel("tag error")
	wrapped := errx.Wrap("context", nil, tag)

	if wrapped != nil {
		t.Error("expected nil when wrapping nil error")
	}
}

// TestWrapNilErrorWithoutTag tests wrapping nil error without tags
func TestWrapNilErrorWithoutTag(t *testing.T) {
	wrapped := errx.Wrap("context", nil)

	if wrapped != nil {
		t.Error("expected nil when wrapping nil error")
	}
}

// TestNestedWrapping tests nested wrapping
func TestNestedWrapping(t *testing.T) {
	tag1 := errx.NewSentinel("tag1")
	tag2 := errx.NewSentinel("tag2")
	baseErr := errors.New("base error")

	wrapped1 := errx.Wrap("level1", baseErr, tag1)
	wrapped2 := errx.Wrap("level2", wrapped1, tag2)

	if wrapped2 == nil {
		t.Fatal("expected non-nil wrapped error")
	}
	if wrapped2.Error() != "level2: level1: base error" {
		t.Errorf("expected 'level2: level1: base error', got %q", wrapped2.Error())
	}
	if !errors.Is(wrapped2, tag1) {
		t.Error("expected error to match tag1")
	}
	if !errors.Is(wrapped2, tag2) {
		t.Error("expected error to match tag2")
	}
	if !errors.Is(wrapped2, baseErr) {
		t.Error("expected error to match base error")
	}
}

// TestUnwrap tests unwrapping errors
func TestUnwrap(t *testing.T) {
	tag := errx.NewSentinel("tag error")
	baseErr := errors.New("base error")
	wrapped := errx.Wrap("context", baseErr, tag)

	unwrapped := errors.Unwrap(wrapped)

	if unwrapped == nil {
		t.Fatal("expected non-nil unwrapped error")
	}
	if !errors.Is(unwrapped, baseErr) {
		t.Error("expected unwrapped to match base error")
	}
	if !errors.Is(unwrapped, tag) {
		t.Error("expected unwrapped to match tag")
	}
}

// TestTagNotInError tests checking for tags not in error
func TestTagNotInError(t *testing.T) {
	tag1 := errx.NewSentinel("tag1")
	tag2 := errx.NewSentinel("tag2")
	baseErr := errors.New("base error")
	wrapped := errx.Wrap("context", baseErr, tag1)

	if !errors.Is(wrapped, tag1) {
		t.Error("expected error to match tag1")
	}
	if errors.Is(wrapped, tag2) {
		t.Error("expected error not to match tag2")
	}
}

// TestErrorMessagePreservation tests that error messages are preserved
func TestErrorMessagePreservation(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		baseErr  error
		expected string
	}{
		{
			name:     "simple context",
			text:     "operation failed",
			baseErr:  errors.New("connection timeout"),
			expected: "operation failed: connection timeout",
		},
		{
			name:     "empty context",
			text:     "",
			baseErr:  errors.New("some error"),
			expected: ": some error",
		},
		{
			name:     "nested error",
			text:     "outer",
			baseErr:  errors.New("inner error"),
			expected: "outer: inner error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := errx.Wrap(tt.text, tt.baseErr)
			if wrapped.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, wrapped.Error())
			}
		})
	}
}

// TestTagTextNotVisible tests that tag text is not visible in error messages
func TestTagTextNotVisible(t *testing.T) {
	tag := errx.NewSentinel("this is a tag")
	baseErr := errors.New("base error")
	wrapped := errx.Wrap("context", baseErr, tag)

	if wrapped.Error() != "context: base error" {
		t.Errorf("expected 'context: base error', got %q", wrapped.Error())
	}
	if fmt.Sprint(wrapped.Error()) != wrapped.Error() {
		t.Error("tag text should not appear in error message")
	}
}

// TestClassify tests the Classify function
func TestClassify(t *testing.T) {
	tag := errx.NewSentinel("tag error")
	baseErr := errors.New("base error")
	classified := errx.Classify(baseErr, tag)

	if classified == nil {
		t.Fatal("expected non-nil classified error")
	}
	if classified.Error() != "base error" {
		t.Errorf("expected 'base error', got %q", classified.Error())
	}
	if !errors.Is(classified, tag) {
		t.Error("expected error to match tag")
	}
	if !errors.Is(classified, baseErr) {
		t.Error("expected error to match base error")
	}
}

// TestClassifyMultipleTags tests classifying with multiple tags
func TestClassifyMultipleTags(t *testing.T) {
	tag1 := errx.NewSentinel("tag1")
	tag2 := errx.NewSentinel("tag2")
	baseErr := errors.New("base error")
	classified := errx.Classify(baseErr, tag1, tag2)

	if classified == nil {
		t.Fatal("expected non-nil classified error")
	}
	if classified.Error() != "base error" {
		t.Errorf("expected 'base error', got %q", classified.Error())
	}
	if !errors.Is(classified, tag1) {
		t.Error("expected error to match tag1")
	}
	if !errors.Is(classified, tag2) {
		t.Error("expected error to match tag2")
	}
	if !errors.Is(classified, baseErr) {
		t.Error("expected error to match base error")
	}
}

// TestClassifyNilError tests classifying nil error
func TestClassifyNilError(t *testing.T) {
	tag := errx.NewSentinel("tag error")
	classified := errx.Classify(nil, tag)

	if classified != nil {
		t.Error("expected nil when classifying nil error")
	}
}

// TestClassifyPreservesErrorMessage tests that Classify preserves error message
func TestClassifyPreservesErrorMessage(t *testing.T) {
	tag := errx.NewSentinel("this is a tag")
	baseErr := errors.New("original error message")
	classified := errx.Classify(baseErr, tag)

	if classified.Error() != "original error message" {
		t.Errorf("expected 'original error message', got %q", classified.Error())
	}
}

// TestClassifyToWrappedError tests classifying a wrapped error
func TestClassifyToWrappedError(t *testing.T) {
	tag1 := errx.NewSentinel("tag1")
	tag2 := errx.NewSentinel("tag2")
	baseErr := errors.New("base error")

	wrapped := errx.Wrap("context", baseErr, tag1)
	classified := errx.Classify(wrapped, tag2)

	if classified == nil {
		t.Fatal("expected non-nil classified error")
	}
	if classified.Error() != "context: base error" {
		t.Errorf("expected 'context: base error', got %q", classified.Error())
	}
	if !errors.Is(classified, tag1) {
		t.Error("expected error to match tag1")
	}
	if !errors.Is(classified, tag2) {
		t.Error("expected error to match tag2")
	}
	if !errors.Is(classified, baseErr) {
		t.Error("expected error to match base error")
	}
}

// TestClassifyVsWrapComparison tests the difference between Classify and Wrap
func TestClassifyVsWrapComparison(t *testing.T) {
	tag := errx.NewSentinel("tag error")
	baseErr := errors.New("base error")

	// Classify doesn't add context text
	classified := errx.Classify(baseErr, tag)
	if classified.Error() != "base error" {
		t.Errorf("expected 'base error', got %q", classified.Error())
	}

	// Wrap adds context text
	wrapped := errx.Wrap("context", baseErr, tag)
	if wrapped.Error() != "context: base error" {
		t.Errorf("expected 'context: base error', got %q", wrapped.Error())
	}

	// Both have the tag attached
	if !errors.Is(classified, tag) {
		t.Error("expected classified to match tag")
	}
	if !errors.Is(wrapped, tag) {
		t.Error("expected wrapped to match tag")
	}
}

// TestSentinelHierarchy_NilParent tests sentinel with nil parent
func TestSentinelHierarchy_NilParent(t *testing.T) {
	// Passing nil as parent should be handled gracefully
	sentinel := errx.NewSentinel("test", nil)

	if sentinel == nil {
		t.Fatal("expected non-nil sentinel")
	}
	if sentinel.Error() != "test" {
		t.Errorf("expected 'test', got %q", sentinel.Error())
	}

	// Should not match nil
	if errors.Is(sentinel, nil) {
		t.Error("sentinel should not match nil")
	}
}

// TestSentinelHierarchy_DeepHierarchy tests deep parent hierarchies (3+ levels)
func TestSentinelHierarchy_DeepHierarchy(t *testing.T) {
	// Create a 4-level hierarchy
	level1 := errx.NewSentinel("level1")
	level2 := errx.NewSentinel("level2", level1)
	level3 := errx.NewSentinel("level3", level2)
	level4 := errx.NewSentinel("level4", level3)

	err := errx.Classify(errors.New("test"), level4)

	// Should match all levels in the hierarchy
	if !errors.Is(err, level4) {
		t.Error("expected error to match level4")
	}
	if !errors.Is(err, level3) {
		t.Error("expected error to match level3")
	}
	if !errors.Is(err, level2) {
		t.Error("expected error to match level2")
	}
	if !errors.Is(err, level1) {
		t.Error("expected error to match level1")
	}

	// Should not match unrelated sentinel
	unrelated := errx.NewSentinel("unrelated")
	if errors.Is(err, unrelated) {
		t.Error("error should not match unrelated sentinel")
	}
}

// TestSentinelHierarchy_MultipleParents tests sentinel with multiple parents
func TestSentinelHierarchy_MultipleParents(t *testing.T) {
	parent1 := errx.NewSentinel("parent1")
	parent2 := errx.NewSentinel("parent2")
	child := errx.NewSentinel("child", parent1, parent2)

	err := errx.Classify(errors.New("test"), child)

	// Should match child and both parents
	if !errors.Is(err, child) {
		t.Error("expected error to match child")
	}
	if !errors.Is(err, parent1) {
		t.Error("expected error to match parent1")
	}
	if !errors.Is(err, parent2) {
		t.Error("expected error to match parent2")
	}
}

// TestCarrier_AsMethod tests the As() method on carrier type
func TestCarrier_AsMethod(t *testing.T) {
	// Create a custom error type
	type customError struct {
		code int
	}

	// Add Error method to make it an error
	customErr := &customError{code: 404}
	customErrWithMethod := fmt.Errorf("error code %d", customErr.code)

	// Wrap it to preserve the custom error
	type wrappedCustomError struct {
		error
		custom *customError
	}

	wce := &wrappedCustomError{
		error:  customErrWithMethod,
		custom: customErr,
	}

	tag := errx.NewSentinel("tag")

	// Wrap the custom error with classification
	wrapped := errx.Classify(wce, tag)

	// Should be able to extract the wrapped custom error using As
	var target *wrappedCustomError
	if !errors.As(wrapped, &target) {
		t.Fatal("expected As to find wrappedCustomError")
	}
	if target.custom.code != 404 {
		t.Errorf("expected code 404, got %d", target.custom.code)
	}
}

// TestCarrier_AsMethod_WithSentinel tests As() with sentinel types
func TestCarrier_AsMethod_WithSentinel(t *testing.T) {
	tag := errx.NewSentinel("tag")
	baseErr := errors.New("base error")
	wrapped := errx.Classify(baseErr, tag)

	// Should be able to extract the sentinel using As
	var target errx.Classified
	if !errors.As(wrapped, &target) {
		t.Fatal("expected As to find Classified")
	}
	if target.Error() != "tag" {
		t.Errorf("expected 'tag', got %q", target.Error())
	}
}
