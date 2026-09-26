package errptr_test

import (
	"errors"
	"testing"

	"github.com/go-extras/errx/internal/errptr"
)

// pointerError is an error with a pointer receiver
type pointerError struct {
	msg string
}

func (e *pointerError) Error() string {
	return e.msg
}

// valueError is an error with a value receiver
type valueError struct {
	msg string
}

func (e valueError) Error() string {
	return e.msg
}

// unhashableError is an error with unhashable fields
type unhashableError struct {
	msg  string
	data map[string]any
}

func (e *unhashableError) Error() string {
	return e.msg
}

func TestGet_Nil(t *testing.T) {
	id := errptr.Get(nil)
	if !id.IsZero() {
		t.Errorf("Get(nil) = %v, want zero ID", id)
	}
}

// makeTypedNil returns an error interface holding a typed-nil *pointerError.
// It exists in a helper so that the typed-nil-vs-interface comparison logic
// is opaque to staticcheck's SA4023 (which would otherwise flag the in-test
// comparison as "never true").
func makeTypedNil() error {
	var pErr *pointerError
	return pErr
}

// TestGet_TypedNil verifies that typed-nil errors (an interface with a
// non-nil type pointer but a nil data pointer) return a zero ID, so callers
// can skip recording untrackable errors in their visited sets.
func TestGet_TypedNil(t *testing.T) {
	e := makeTypedNil() // interface holding typed-nil

	if e == nil {
		t.Fatal("test precondition failed: typed-nil should compare != nil as interface")
	}

	id := errptr.Get(e)
	if !id.IsZero() {
		t.Errorf("Get(typed-nil) = %v, want zero ID", id)
	}
}

// TestGet_TypedNil_DAGCollision verifies that two distinct typed-nil errors
// of the same underlying type both yield a zero ID and therefore do not falsely
// appear as the same identity for visited-set tracking (callers use a zero ID as a
// sentinel meaning "do not record"). Prior to the fix, both returned the
// same non-zero key, causing spurious "(circular reference)" reports.
func TestGet_TypedNil_DAGCollision(t *testing.T) {
	e1 := makeTypedNil()
	e2 := makeTypedNil()

	id1 := errptr.Get(e1)
	id2 := errptr.Get(e2)

	if !id1.IsZero() || !id2.IsZero() {
		t.Errorf("typed-nil errors should return zero IDs, got id1=%v id2=%v", id1, id2)
	}
}

func TestGet_PointerError_SameInstance(t *testing.T) {
	err := &pointerError{msg: "test"}
	var e1 error = err
	var e2 error = err

	id1 := errptr.Get(e1)
	id2 := errptr.Get(e2)

	if id1 != id2 {
		t.Errorf("Same instance should have same identity: %v != %v", id1, id2)
	}
	if id1.IsZero() {
		t.Error("Identity should not be zero for non-nil error")
	}
}

func TestGet_PointerError_DifferentInstances(t *testing.T) {
	err1 := &pointerError{msg: "test"}
	err2 := &pointerError{msg: "test"}

	id1 := errptr.Get(err1)
	id2 := errptr.Get(err2)

	if id1 == id2 {
		t.Errorf("Different instances should have different identities: %v == %v", id1, id2)
	}
}

func TestGet_ValueError_SameVariable(t *testing.T) {
	// Value errors are boxed when converted to interfaces. The compiler
	// may reuse storage, so distinct conversions need not have distinct IDs.
	valErr := valueError{msg: "test"}
	var e1 error = valErr
	var e2 error = valErr

	id1 := errptr.Get(e1)
	id2 := errptr.Get(e2)

	// Both conversions must have trackable identities.
	if id1.IsZero() || id2.IsZero() {
		t.Error("Identities should not be zero for non-nil errors")
	}
}

func TestGet_ValueError_DifferentValues(t *testing.T) {
	// Create value errors with different content
	var e1 error = valueError{msg: "test1"}
	var e2 error = valueError{msg: "test2"}

	id1 := errptr.Get(e1)
	id2 := errptr.Get(e2)

	if id1.IsZero() || id2.IsZero() {
		t.Error("Identities should not be zero for non-nil errors")
	}

	// Different values should have different identities
	if id1 == id2 {
		t.Errorf("Different value errors should have different identities, got id1=%v id2=%v", id1, id2)
	}
}

func TestGet_UnhashableError(t *testing.T) {
	// This should not panic even though the error has unhashable fields
	err := &unhashableError{
		msg:  "test",
		data: map[string]any{"key": "value"},
	}

	id := errptr.Get(err)
	if id.IsZero() {
		t.Error("Identity should not be zero for non-nil error")
	}
}

func TestGet_StandardError(t *testing.T) {
	err := errors.New("standard error")
	id := errptr.Get(err)

	if id.IsZero() {
		t.Error("Identity should not be zero for non-nil error")
	}
}

func TestGet_Consistency(t *testing.T) {
	// Calling Get multiple times on the same error should return the same identity
	err := &pointerError{msg: "test"}

	id1 := errptr.Get(err)
	id2 := errptr.Get(err)
	id3 := errptr.Get(err)

	if id1 != id2 || id2 != id3 {
		t.Errorf("Multiple calls should return same identity: %v, %v, %v", id1, id2, id3)
	}
}

func TestGet_WrappedError(t *testing.T) {
	inner := &pointerError{msg: "inner"}
	outer := &pointerError{msg: "outer"}

	idInner := errptr.Get(inner)
	idOuter := errptr.Get(outer)

	if idInner == idOuter {
		t.Error("Different errors should have different identities")
	}
}

type firstFieldError struct{ inner pointerError }

func (e *firstFieldError) Error() string { return e.inner.Error() }

// TestGet_FirstField distinguishes different types at the same address.
func TestGet_FirstField(t *testing.T) {
	err := &firstFieldError{inner: pointerError{msg: "test"}}
	if errptr.Get(err) == errptr.Get(&err.inner) {
		t.Fatal("wrapper and first-field cause must have different identities")
	}
}

type zeroErrorA struct{}
type zeroErrorB struct{}

func (zeroErrorA) Error() string { return "a" }
func (zeroErrorB) Error() string { return "b" }

// TestGet_ZeroSizeDistinctTypes checks values and pointers to zero-size errors.
func TestGet_ZeroSizeDistinctTypes(t *testing.T) {
	if errptr.Get(zeroErrorA{}) == errptr.Get(zeroErrorB{}) {
		t.Fatal("different zero-size value types must have different identities")
	}
	a := &zeroErrorA{}
	b := (*zeroErrorB)(a) // Guarantee the same address for the pointer case.
	if errptr.Get(a) == errptr.Get(b) {
		t.Fatal("different zero-size pointer types must have different identities")
	}
}

type unhashableValueError []string

func (unhashableValueError) Error() string { return "unhashable" }

// TestGet_UnhashableValue verifies that IDs do not hash the dynamic error value.
func TestGet_UnhashableValue(t *testing.T) {
	var err error = unhashableValueError{"test"}
	id := errptr.Get(err)
	seen := map[errptr.ID]bool{id: true}
	if !seen[errptr.Get(err)] {
		t.Fatal("the same unhashable error must retain its identity")
	}
}
