package errx_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-extras/errx"
)

func TestNewDisplayable(t *testing.T) {
	err := errx.NewDisplayable("user message")

	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if err.Error() != "user message" {
		t.Errorf("expected 'user message', got %q", err.Error())
	}
}

func TestDisplayText_WithDisplayable(t *testing.T) {
	displayErr := errx.NewDisplayable("visible to user")
	text := errx.DisplayText(displayErr)

	if text != "visible to user" {
		t.Errorf("expected 'visible to user', got %q", text)
	}
}

func TestDisplayText_WithRegularError(t *testing.T) {
	regularErr := errors.New("internal error")
	text := errx.DisplayText(regularErr)

	if text != "internal error" {
		t.Errorf("expected 'internal error', got %q", text)
	}
}

func TestDisplayText_WithNil(t *testing.T) {
	text := errx.DisplayText(nil)

	if text != "" {
		t.Errorf("expected empty string, got %q", text)
	}
}

func TestDisplayText_WithWrappedDisplayable(t *testing.T) {
	displayErr := errx.NewDisplayable("user error")
	wrapped := fmt.Errorf("context: %w", displayErr)

	text := errx.DisplayText(wrapped)

	if text != "user error" {
		t.Errorf("expected 'user error', got %q", text)
	}
}

func TestDisplayText_WithWrappedRegularError(t *testing.T) {
	regularErr := errors.New("internal error")
	wrapped := fmt.Errorf("context: %w", regularErr)

	text := errx.DisplayText(wrapped)

	if text != "context: internal error" {
		t.Errorf("expected 'context: internal error', got %q", text)
	}
}

func TestDisplayText_WithDeepChain(t *testing.T) {
	displayErr := errx.NewDisplayable("user error")
	level1 := fmt.Errorf("level1: %w", displayErr)
	level2 := fmt.Errorf("level2: %w", level1)
	level3 := fmt.Errorf("level3: %w", level2)

	text := errx.DisplayText(level3)

	if text != "user error" {
		t.Errorf("expected 'user error', got %q", text)
	}
}

func TestDisplayText_WithMultipleDisplayables(t *testing.T) {
	displayErr1 := errx.NewDisplayable("first user error")
	displayErr2 := errx.NewDisplayable("second user error")
	wrapped := fmt.Errorf("level1: %w", displayErr2)
	final := fmt.Errorf("level2: %w: also %w", wrapped, displayErr1)

	text := errx.DisplayText(final)

	// Should return the first one found in the chain
	if text == "" {
		t.Error("expected non-empty displayable message")
	}
}

func TestDisplayText_WithWrap(t *testing.T) {
	displayErr := errx.NewDisplayable("user error")
	ErrNotFound := errx.NewSentinel("not found")
	wrapped := errx.Wrap("operation failed", displayErr, ErrNotFound)

	text := errx.DisplayText(wrapped)

	if text != "user error" {
		t.Errorf("expected 'user error', got %q", text)
	}
}

func TestDisplayText_WithWrapAndRegularError(t *testing.T) {
	regularErr := errors.New("internal error")
	ErrNotFound := errx.NewSentinel("not found")
	wrapped := errx.Wrap("operation failed", regularErr, ErrNotFound)

	text := errx.DisplayText(wrapped)

	if text != "operation failed: internal error" {
		t.Errorf("expected 'operation failed: internal error', got %q", text)
	}
}

func TestIsDisplayable_WithDisplayable(t *testing.T) {
	displayErr := errx.NewDisplayable("user error")

	if !errx.IsDisplayable(displayErr) {
		t.Error("expected IsDisplayable to return true")
	}
}

func TestIsDisplayable_WithRegularError(t *testing.T) {
	regularErr := errors.New("regular error")

	if errx.IsDisplayable(regularErr) {
		t.Error("expected IsDisplayable to return false")
	}
}

func TestIsDisplayable_WithNil(t *testing.T) {
	if errx.IsDisplayable(nil) {
		t.Error("expected IsDisplayable to return false for nil")
	}
}

func TestIsDisplayable_WithWrappedDisplayable(t *testing.T) {
	displayErr := errx.NewDisplayable("user error")
	wrapped := fmt.Errorf("context: %w", displayErr)

	if !errx.IsDisplayable(wrapped) {
		t.Error("expected IsDisplayable to return true for wrapped displayable")
	}
}

func TestIsDisplayable_WithDeepChain(t *testing.T) {
	displayErr := errx.NewDisplayable("user error")
	level1 := fmt.Errorf("level1: %w", displayErr)
	level2 := fmt.Errorf("level2: %w", level1)
	level3 := fmt.Errorf("level3: %w", level2)

	if !errx.IsDisplayable(level3) {
		t.Error("expected IsDisplayable to return true for deep chain")
	}
}

func TestDisplayable_WithClassify(t *testing.T) {
	displayErr := errx.NewDisplayable("user message")
	ErrNotFound := errx.NewSentinel("not found")
	classified := errx.Classify(displayErr, ErrNotFound)

	// Should be displayable
	if !errx.IsDisplayable(classified) {
		t.Error("expected classified error to be displayable")
	}

	// Should match sentinel
	if !errors.Is(classified, ErrNotFound) {
		t.Error("expected error to match sentinel")
	}

	// Should return displayable text
	text := errx.DisplayText(classified)
	if text != "user message" {
		t.Errorf("expected 'user message', got %q", text)
	}
}

func TestDisplayable_PreservesErrorMessage(t *testing.T) {
	displayErr := errx.NewDisplayable("This is the user message")
	fullMsg := displayErr.Error()

	if fullMsg != "This is the user message" {
		t.Errorf("expected 'This is the user message', got %q", fullMsg)
	}
}

func TestDisplayText_ExtractsOnlyDisplayableMessage(t *testing.T) {
	// Create error with internal context
	displayErr := errx.NewDisplayable("File not found")
	wrapped := errx.Wrap("failed to open config", displayErr)
	deepWrapped := fmt.Errorf("startup failed: %w", wrapped)

	// Full error should have all context
	fullMsg := deepWrapped.Error()
	expected := "startup failed: failed to open config: File not found"
	if fullMsg != expected {
		t.Errorf("expected %q, got %q", expected, fullMsg)
	}

	// DisplayText should extract only the displayable message
	text := errx.DisplayText(deepWrapped)
	if text != "File not found" {
		t.Errorf("expected 'File not found', got %q", text)
	}
}

func TestDisplayTextDefault_WithDisplayable(t *testing.T) {
	displayErr := errx.NewDisplayable("User-facing message")
	defaultMsg := "Default error message"

	text := errx.DisplayTextDefault(displayErr, defaultMsg)

	if text != "User-facing message" {
		t.Errorf("expected 'User-facing message', got %q", text)
	}
}

func TestDisplayTextDefault_WithRegularError(t *testing.T) {
	regularErr := errors.New("internal error")
	defaultMsg := "Something went wrong"

	text := errx.DisplayTextDefault(regularErr, defaultMsg)

	if text != defaultMsg {
		t.Errorf("expected %q, got %q", defaultMsg, text)
	}
}

func TestDisplayTextDefault_WithNil(t *testing.T) {
	defaultMsg := "Default message"

	text := errx.DisplayTextDefault(nil, defaultMsg)

	if text != "" {
		t.Errorf("expected empty string for nil error, got %q", text)
	}
}

func TestDisplayTextDefault_WithWrappedDisplayable(t *testing.T) {
	displayErr := errx.NewDisplayable("User error")
	wrapped := fmt.Errorf("context: %w", displayErr)
	defaultMsg := "Generic error"

	text := errx.DisplayTextDefault(wrapped, defaultMsg)

	if text != "User error" {
		t.Errorf("expected 'User error', got %q", text)
	}
}

func TestDisplayTextDefault_WithWrappedRegularError(t *testing.T) {
	regularErr := errors.New("internal error")
	wrapped := fmt.Errorf("context: %w", regularErr)
	defaultMsg := "An error occurred"

	text := errx.DisplayTextDefault(wrapped, defaultMsg)

	if text != defaultMsg {
		t.Errorf("expected %q, got %q", defaultMsg, text)
	}
}

func TestDisplayTextDefault_WithDeepChain(t *testing.T) {
	displayErr := errx.NewDisplayable("User error")
	level1 := fmt.Errorf("level1: %w", displayErr)
	level2 := fmt.Errorf("level2: %w", level1)
	level3 := fmt.Errorf("level3: %w", level2)
	defaultMsg := "Fallback message"

	text := errx.DisplayTextDefault(level3, defaultMsg)

	if text != "User error" {
		t.Errorf("expected 'User error', got %q", text)
	}
}

func TestDisplayTextDefault_WithEmptyDefaultMessage(t *testing.T) {
	regularErr := errors.New("internal error")
	defaultMsg := ""

	text := errx.DisplayTextDefault(regularErr, defaultMsg)

	if text != "" {
		t.Errorf("expected empty string, got %q", text)
	}
}

func TestDisplayTextDefault_WithWrap(t *testing.T) {
	displayErr := errx.NewDisplayable("User message")
	ErrNotFound := errx.NewSentinel("not found")
	wrapped := errx.Wrap("operation failed", displayErr, ErrNotFound)
	defaultMsg := "Error occurred"

	text := errx.DisplayTextDefault(wrapped, defaultMsg)

	if text != "User message" {
		t.Errorf("expected 'User message', got %q", text)
	}
}

func TestDisplayTextDefault_WithWrapAndRegularError(t *testing.T) {
	regularErr := errors.New("internal error")
	ErrNotFound := errx.NewSentinel("not found")
	wrapped := errx.Wrap("operation failed", regularErr, ErrNotFound)
	defaultMsg := "Service unavailable"

	text := errx.DisplayTextDefault(wrapped, defaultMsg)

	if text != defaultMsg {
		t.Errorf("expected %q, got %q", defaultMsg, text)
	}
}
