package errx_test

import (
	"errors"
	"testing"

	"github.com/go-extras/errx"
)

// TestJoinNilHandling verifies parity with stdlib errors.Join: nil arguments
// are dropped, and an all-nil (or empty) call returns nil.
func TestJoinNilHandling(t *testing.T) {
	if err := errx.Join(); err != nil {
		t.Errorf("Join() = %v, want nil", err)
	}
	if err := errx.Join(nil, nil); err != nil {
		t.Errorf("Join(nil, nil) = %v, want nil", err)
	}

	base := errors.New("boom")
	err := errx.Join(nil, base, nil)
	if err == nil {
		t.Fatal("Join(nil, base, nil) = nil, want non-nil")
	}
	if !errors.Is(err, base) {
		t.Error("expected joined error to match its single non-nil member")
	}
	if err.Error() != "boom" {
		t.Errorf("Error() = %q, want %q", err.Error(), "boom")
	}
}

// TestJoinErrorFormat verifies members render one per line, matching errors.Join.
func TestJoinErrorFormat(t *testing.T) {
	err := errx.Join(errors.New("first"), errors.New("second"))
	if got, want := err.Error(), "first\nsecond"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// TestJoinSingleElement pins the single-member contract: Error() has no trailing
// newline and the member's identity is preserved.
func TestJoinSingleElement(t *testing.T) {
	solo := errors.New("solo")
	err := errx.Join(solo)
	if got, want := err.Error(), "solo"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, solo) {
		t.Error("expected single-element join to match its member")
	}
}

// TestJoinOfJoin verifies nested joins traverse for errors.Is and render flat.
func TestJoinOfJoin(t *testing.T) {
	ErrA := errx.NewSentinel("a")
	inner := errx.Join(errx.ClassifyNew("x", ErrA), errors.New("y"))
	err := errx.Join(inner, errors.New("z"))

	if !errors.Is(err, ErrA) {
		t.Error("expected nested join to match inner sentinel ErrA")
	}
	if got, want := err.Error(), "x\ny\nz"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// TestJoinDoesNotAliasCallerSlice verifies the constructor copies its input, so
// mutating the caller's slice afterward does not change the aggregate.
func TestJoinDoesNotAliasCallerSlice(t *testing.T) {
	e1 := errors.New("one")
	e2 := errors.New("two")
	src := []error{e1, e2}
	err := errx.Join(src...)

	src[0] = errors.New("mutated")
	if !errors.Is(err, e1) {
		t.Error("join must not alias the caller's slice")
	}
}

// TestJoinIsTraversal verifies errors.Is descends into every joined branch.
func TestJoinIsTraversal(t *testing.T) {
	ErrA := errx.NewSentinel("a")
	ErrB := errx.NewSentinel("b")

	err := errx.Join(errx.ClassifyNew("first", ErrA), errx.ClassifyNew("second", ErrB))
	if !errors.Is(err, ErrA) {
		t.Error("expected joined error to match ErrA")
	}
	if !errors.Is(err, ErrB) {
		t.Error("expected joined error to match ErrB")
	}
}

// TestJoinClassifyComposition verifies the documented composition pattern:
// classifying an aggregate via the existing API, with attributes traversed
// across every joined branch.
func TestJoinClassifyComposition(t *testing.T) {
	ErrBatch := errx.NewSentinel("batch failed")

	inner1 := errx.Classify(errors.New("item 1 failed"), errx.Attrs("index", 0))
	inner2 := errx.Classify(errors.New("item 2 failed"), errx.Attrs("index", 1))

	err := errx.Classify(errx.Join(inner1, inner2), ErrBatch, errx.Attrs("count", 2))

	if !errors.Is(err, ErrBatch) {
		t.Error("expected aggregate to be classified as ErrBatch")
	}
	if !errx.HasAttrs(err) {
		t.Fatal("expected aggregate to report attributes")
	}

	attrs := errx.ExtractAttrs(err)
	got := make(map[string]any, len(attrs))
	for _, a := range attrs {
		got[a.Key] = a.Value
	}
	// Attributes from the carrier and from every joined branch should surface.
	if _, ok := got["count"]; !ok {
		t.Error("expected group-level attribute 'count'")
	}
	if _, ok := got["index"]; !ok {
		t.Error("expected per-branch attribute 'index' from joined members")
	}
}

// TestJoinUnwrapList verifies the result exposes Unwrap() []error so multi-error
// aware consumers (including the json subpackage) can enumerate members.
func TestJoinUnwrapList(t *testing.T) {
	e1 := errors.New("one")
	e2 := errors.New("two")
	err := errx.Join(e1, e2)

	u, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatal("joined error does not implement Unwrap() []error")
	}
	members := u.Unwrap()
	if len(members) != 2 {
		t.Fatalf("Unwrap() returned %d members, want 2", len(members))
	}
	if members[0] != e1 || members[1] != e2 {
		t.Error("Unwrap() did not preserve member order/identity")
	}
}
