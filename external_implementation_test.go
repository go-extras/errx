package errx_test

import (
	"errors"
	"testing"

	"github.com/go-extras/errx"
)

// customClassified is an example of an external implementation of the Classified interface.
// This demonstrates that external packages can now extend the library by implementing Classified.
type customClassified struct {
	message string
	code    int
}

func (c *customClassified) Error() string {
	return c.message
}

// IsClassified implements the Classified interface marker method.
// This allows the error to be recognized as a Classified error.
func (*customClassified) IsClassified() bool {
	return true
}

// TestExternalClassifiedImplementation verifies that external packages can implement Classified
func TestExternalClassifiedImplementation(t *testing.T) {
	// Create a custom classified error
	customErr := &customClassified{
		message: "custom error",
		code:    404,
	}

	// Verify it implements Classified
	var classified errx.Classified = customErr
	if classified.Error() != "custom error" {
		t.Errorf("expected 'custom error', got %q", classified.Error())
	}

	// Verify IsClassified returns true
	if !customErr.IsClassified() {
		t.Error("expected IsClassified to return true")
	}

	// Verify it can be used with errx.Wrap
	wrapped := errx.Wrap("operation failed", errors.New("base error"), customErr)
	if wrapped == nil {
		t.Fatal("expected non-nil wrapped error")
	}

	// Verify it can be extracted using errors.As
	var target errx.Classified
	if !errors.As(wrapped, &target) {
		t.Fatal("expected As to find Classified")
	}

	// Verify we can extract our custom type
	var customTarget *customClassified
	if !errors.As(wrapped, &customTarget) {
		t.Fatal("expected As to find customClassified")
	}
	if customTarget.code != 404 {
		t.Errorf("expected code 404, got %d", customTarget.code)
	}
}

// TestExternalClassifiedWithClassify verifies external Classified works with Classify
func TestExternalClassifiedWithClassify(t *testing.T) {
	customErr := &customClassified{
		message: "custom classified error",
		code:    500,
	}

	baseErr := errors.New("base error")
	classified := errx.Classify(baseErr, customErr)

	// Verify the error chain
	if classified.Error() != "base error" {
		t.Errorf("expected 'base error', got %q", classified.Error())
	}

	// Verify we can extract the custom classified error
	var target *customClassified
	if !errors.As(classified, &target) {
		t.Fatal("expected As to find customClassified")
	}
	if target.code != 500 {
		t.Errorf("expected code 500, got %d", target.code)
	}
}

// TestExternalClassifiedMarkerMethod verifies the IsClassified marker method works
func TestExternalClassifiedMarkerMethod(t *testing.T) {
	customErr := &customClassified{
		message: "test error",
		code:    400,
	}

	// Verify the marker method
	if !customErr.IsClassified() {
		t.Error("expected IsClassified to return true")
	}

	// Verify it can be used as a Classified interface
	var classified errx.Classified = customErr
	if !classified.IsClassified() {
		t.Error("expected IsClassified to return true through interface")
	}
}
