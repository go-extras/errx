package compat_test

import (
	"errors"
	"testing"

	"github.com/go-extras/errx/compat"
)

// typedErr is a custom error type used to verify errors.As works through the compat wrapper.
type typedErr struct {
	Code int
	Msg  string
}

func (e *typedErr) Error() string { return e.Msg }

// TestErrorWrapper_IsTransparent verifies that a standard error passed as a
// classification still matches via errors.Is once the compat layer has wrapped it.
func TestErrorWrapper_IsTransparent(t *testing.T) {
	sentinel := errors.New("sentinel")
	base := errors.New("base")

	err := compat.Classify(base, sentinel)
	if !errors.Is(err, sentinel) {
		t.Error("expected errors.Is(err, sentinel) to be true via compat wrapper")
	}
}

// TestErrorWrapper_IsTransparent_WrappedSentinel verifies the Is method delegates
// to errors.Is on the inner error, so wrapped sentinels still match.
func TestErrorWrapper_IsTransparent_WrappedSentinel(t *testing.T) {
	sentinel := errors.New("sentinel")
	wrappedSentinel := errors.Join(errors.New("group"), sentinel)
	base := errors.New("base")

	err := compat.Classify(base, wrappedSentinel)
	if !errors.Is(err, sentinel) {
		t.Error("expected errors.Is to see through compat wrapper into the wrapped classification")
	}
}

// TestErrorWrapper_AsTransparent verifies that errors.As can extract a typed error
// that was originally provided as a classification (i.e. the wrapper does not
// hide custom error types from callers).
func TestErrorWrapper_AsTransparent(t *testing.T) {
	typedSentinel := &typedErr{Code: 7, Msg: "typed sentinel"}
	base := errors.New("base")

	err := compat.Classify(base, typedSentinel)

	var got *typedErr
	if !errors.As(err, &got) {
		t.Fatal("expected errors.As to extract *typedErr through compat wrapper")
	}
	if got != typedSentinel {
		t.Errorf("errors.As returned a different pointer than the original classification")
	}
	if got.Code != 7 {
		t.Errorf("typedErr.Code = %d, want 7", got.Code)
	}
}

// TestWrap_NoClassifications_DoesNotAllocateEmptySlice is a behavior-equivalent
// smoke test that the no-classification path still returns a working error.
// (The optimization is internal; we can't directly assert "no allocation" without
// benchmarks, but we can assert the function still behaves correctly.)
func TestWrap_NoClassifications_DoesNotAllocateEmptySlice(t *testing.T) {
	base := errors.New("base")
	err := compat.Wrap("ctx", base)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if err.Error() != "ctx: base" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

// TestClassify_AllNilClassifications verifies the all-nil short-circuit returns
// a working result (no classifications attached).
func TestClassify_AllNilClassifications(t *testing.T) {
	base := errors.New("base")
	err := compat.Classify(base, nil, nil, nil)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !errors.Is(err, base) {
		t.Error("expected errors.Is(err, base) to be true")
	}
}

// TestClassifyNew_AllNilClassifications verifies the all-nil short-circuit on ClassifyNew.
func TestClassifyNew_AllNilClassifications(t *testing.T) {
	err := compat.ClassifyNew("oops", nil, nil)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if err.Error() != "oops" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}
