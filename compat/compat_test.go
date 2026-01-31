package compat_test

import (
	"errors"
	"testing"

	"github.com/go-extras/errx"
	"github.com/go-extras/errx/compat"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrDatabase   = errors.New("database error")
	ErrValidation = errors.New("validation error")
)

func TestWrap_NilCause(t *testing.T) {
	err := compat.Wrap("context", nil, ErrNotFound)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestWrap_WithoutClassifications(t *testing.T) {
	baseErr := errors.New("base error")
	err := compat.Wrap("context", baseErr)

	if err == nil {
		t.Fatal("expected non-nil error")
	}

	expected := "context: base error"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestWrap_WithSingleClassification(t *testing.T) {
	baseErr := errors.New("base error")
	err := compat.Wrap("context", baseErr, ErrNotFound)

	if err == nil {
		t.Fatal("expected non-nil error")
	}

	// Check error message
	expected := "context: base error"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}

	// Check classification
	if !errors.Is(err, ErrNotFound) {
		t.Error("expected error to be classified as ErrNotFound")
	}

	// Check original error is preserved
	if !errors.Is(err, baseErr) {
		t.Error("expected error to wrap baseErr")
	}
}

func TestWrap_WithMultipleClassifications(t *testing.T) {
	baseErr := errors.New("base error")
	err := compat.Wrap("context", baseErr, ErrNotFound, ErrDatabase)

	if err == nil {
		t.Fatal("expected non-nil error")
	}

	// Check both classifications
	if !errors.Is(err, ErrNotFound) {
		t.Error("expected error to be classified as ErrNotFound")
	}
	if !errors.Is(err, ErrDatabase) {
		t.Error("expected error to be classified as ErrDatabase")
	}

	// Check original error is preserved
	if !errors.Is(err, baseErr) {
		t.Error("expected error to wrap baseErr")
	}
}

func TestWrap_WithErrxClassified(t *testing.T) {
	baseErr := errors.New("base error")
	sentinel := errx.NewSentinel("sentinel")
	displayable := errx.NewDisplayable("user message")

	err := compat.Wrap("context", baseErr, sentinel, displayable, ErrNotFound)

	if err == nil {
		t.Fatal("expected non-nil error")
	}

	// Check all classifications work
	if !errors.Is(err, sentinel) {
		t.Error("expected error to be classified as sentinel")
	}
	if !errors.Is(err, displayable) {
		t.Error("expected error to be classified as displayable")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Error("expected error to be classified as ErrNotFound")
	}
}

func TestClassify_NilCause(t *testing.T) {
	err := compat.Classify(nil, ErrNotFound)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestClassify_WithoutClassifications(t *testing.T) {
	baseErr := errors.New("base error")
	err := compat.Classify(baseErr)

	if err == nil {
		t.Fatal("expected non-nil error")
	}

	// Should preserve original message
	if err.Error() != baseErr.Error() {
		t.Errorf("expected %q, got %q", baseErr.Error(), err.Error())
	}
}

func TestClassify_WithSingleClassification(t *testing.T) {
	baseErr := errors.New("base error")
	err := compat.Classify(baseErr, ErrValidation)

	if err == nil {
		t.Fatal("expected non-nil error")
	}

	// Should preserve original message
	if err.Error() != baseErr.Error() {
		t.Errorf("expected %q, got %q", baseErr.Error(), err.Error())
	}

	// Check classification
	if !errors.Is(err, ErrValidation) {
		t.Error("expected error to be classified as ErrValidation")
	}

	// Check original error is preserved
	if !errors.Is(err, baseErr) {
		t.Error("expected error to wrap baseErr")
	}
}

func TestClassify_WithMultipleClassifications(t *testing.T) {
	baseErr := errors.New("base error")
	err := compat.Classify(baseErr, ErrValidation, ErrDatabase)

	if err == nil {
		t.Fatal("expected non-nil error")
	}

	// Check both classifications
	if !errors.Is(err, ErrValidation) {
		t.Error("expected error to be classified as ErrValidation")
	}
	if !errors.Is(err, ErrDatabase) {
		t.Error("expected error to be classified as ErrDatabase")
	}
}

func TestClassify_WithErrxClassified(t *testing.T) {
	baseErr := errors.New("base error")
	sentinel := errx.NewSentinel("sentinel")
	attrErr := errx.Attrs("key", "value")

	err := compat.Classify(baseErr, sentinel, attrErr, ErrValidation)

	if err == nil {
		t.Fatal("expected non-nil error")
	}

	// Check all classifications work
	if !errors.Is(err, sentinel) {
		t.Error("expected error to be classified as sentinel")
	}
	if !errors.Is(err, ErrValidation) {
		t.Error("expected error to be classified as ErrValidation")
	}

	// Check attributes are preserved
	if !errx.HasAttrs(err) {
		t.Error("expected error to have attributes")
	}
}

func TestWrap_WithAttributes(t *testing.T) {
	baseErr := errors.New("base error")
	attrErr := errx.Attrs("user_id", 123, "action", "delete")

	err := compat.Wrap("context", baseErr, attrErr, ErrDatabase)

	if err == nil {
		t.Fatal("expected non-nil error")
	}

	// Check attributes are preserved
	if !errx.HasAttrs(err) {
		t.Error("expected error to have attributes")
	}

	attrs := errx.ExtractAttrs(err)
	if len(attrs) != 2 {
		t.Errorf("expected 2 attributes, got %d", len(attrs))
	}

	// Check classification
	if !errors.Is(err, ErrDatabase) {
		t.Error("expected error to be classified as ErrDatabase")
	}
}

func TestWrap_WithDisplayable(t *testing.T) {
	baseErr := errors.New("internal error")
	displayable := errx.NewDisplayable("User-friendly message")

	err := compat.Wrap("context", baseErr, displayable)

	if err == nil {
		t.Fatal("expected non-nil error")
	}

	// Check displayable message can be extracted
	if !errors.Is(err, displayable) {
		t.Error("expected error to contain displayable")
	}
}

func TestErrorWrapper_PreservesIdentity(t *testing.T) {
	baseErr := errors.New("base error")
	err := compat.Wrap("context", baseErr, ErrNotFound)

	// The wrapped ErrNotFound should still be identifiable
	if !errors.Is(err, ErrNotFound) {
		t.Error("expected error to be classified as ErrNotFound")
	}

	// Should also work with errors.As for the base error
	var target *struct{ error }
	if errors.As(err, &target) {
		t.Error("expected errors.As to not match unrelated type")
	}

	// Verify we can unwrap to the base error
	if !errors.Is(err, baseErr) {
		t.Error("expected error to wrap baseErr")
	}
}

func TestWrap_ChainedCalls(t *testing.T) {
	baseErr := errors.New("base error")
	err1 := compat.Classify(baseErr, ErrDatabase)
	err2 := compat.Wrap("layer 2", err1, ErrValidation)
	err3 := compat.Wrap("layer 3", err2, ErrNotFound)

	// All classifications should be preserved
	if !errors.Is(err3, ErrDatabase) {
		t.Error("expected error to be classified as ErrDatabase")
	}
	if !errors.Is(err3, ErrValidation) {
		t.Error("expected error to be classified as ErrValidation")
	}
	if !errors.Is(err3, ErrNotFound) {
		t.Error("expected error to be classified as ErrNotFound")
	}

	// Original error should be preserved
	if !errors.Is(err3, baseErr) {
		t.Error("expected error to wrap baseErr")
	}

	// Check error message
	expected := "layer 3: layer 2: base error"
	if err3.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err3.Error())
	}
}

func TestClassify_PreservesMessage(t *testing.T) {
	baseErr := errors.New("very specific error message")
	err := compat.Classify(baseErr, ErrValidation)

	// Classify should not modify the error message
	if err.Error() != baseErr.Error() {
		t.Errorf("expected message %q to be preserved, got %q", baseErr.Error(), err.Error())
	}
}

func TestWrap_NilClassifications(t *testing.T) {
	baseErr := errors.New("base error")

	// Test with nil in classifications slice
	err := compat.Wrap("context", baseErr, nil, ErrNotFound, nil)

	if err == nil {
		t.Fatal("expected non-nil error")
	}

	// Should still work with non-nil classification
	if !errors.Is(err, ErrNotFound) {
		t.Error("expected error to be classified as ErrNotFound")
	}
}
