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
	ptr := errptr.Get(nil)
	if ptr != 0 {
		t.Errorf("Get(nil) = %v, want 0", ptr)
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
// non-nil type pointer but a nil data pointer) return 0. Returning 0 lets
// callers' "skip if 0" sentinel handling work correctly, instead of having
// multiple typed-nil errors collide on the same non-zero key.
func TestGet_TypedNil(t *testing.T) {
	e := makeTypedNil() // interface holding typed-nil

	if e == nil {
		t.Fatal("test precondition failed: typed-nil should compare != nil as interface")
	}

	ptr := errptr.Get(e)
	if ptr != 0 {
		t.Errorf("Get(typed-nil) = %v, want 0", ptr)
	}
}

// TestGet_TypedNil_DAGCollision verifies that two distinct typed-nil errors
// of the same underlying type both yield 0 and therefore do not falsely
// appear as the same identity for visited-set tracking (callers use 0 as a
// sentinel meaning "do not record"). Prior to the fix, both returned the
// same non-zero key, causing spurious "(circular reference)" reports.
func TestGet_TypedNil_DAGCollision(t *testing.T) {
	e1 := makeTypedNil()
	e2 := makeTypedNil()

	ptr1 := errptr.Get(e1)
	ptr2 := errptr.Get(e2)

	if ptr1 != 0 || ptr2 != 0 {
		t.Errorf("typed-nil pointers should return 0, got ptr1=%v ptr2=%v", ptr1, ptr2)
	}
}

func TestGet_PointerError_SameInstance(t *testing.T) {
	err := &pointerError{msg: "test"}
	var e1 error = err
	var e2 error = err

	ptr1 := errptr.Get(e1)
	ptr2 := errptr.Get(e2)

	if ptr1 != ptr2 {
		t.Errorf("Same instance should have same pointer: %v != %v", ptr1, ptr2)
	}
	if ptr1 == 0 {
		t.Error("Pointer should not be 0 for non-nil error")
	}
}

func TestGet_PointerError_DifferentInstances(t *testing.T) {
	err1 := &pointerError{msg: "test"}
	err2 := &pointerError{msg: "test"}

	ptr1 := errptr.Get(err1)
	ptr2 := errptr.Get(err2)

	if ptr1 == ptr2 {
		t.Errorf("Different instances should have different pointers: %v == %v", ptr1, ptr2)
	}
}

func TestGet_ValueError_SameVariable(t *testing.T) {
	// Note: When a value error is assigned to an interface, the interface
	// stores a copy of the value. Each assignment creates a new copy.
	valErr := valueError{msg: "test"}
	var e1 error = valErr
	var e2 error = valErr

	ptr1 := errptr.Get(e1)
	ptr2 := errptr.Get(e2)

	// These will be different because each assignment to interface creates a new copy
	// This is expected behavior - we're testing pointer identity, not value equality
	if ptr1 == 0 || ptr2 == 0 {
		t.Error("Pointers should not be 0 for non-nil errors")
	}
}

func TestGet_ValueError_DifferentValues(t *testing.T) {
	// Create value errors with different content
	var e1 error = valueError{msg: "test1"}
	var e2 error = valueError{msg: "test2"}

	ptr1 := errptr.Get(e1)
	ptr2 := errptr.Get(e2)

	if ptr1 == 0 || ptr2 == 0 {
		t.Error("Pointers should not be 0 for non-nil errors")
	}

	// Different values should have different pointers
	if ptr1 == ptr2 {
		t.Errorf("Different value errors should have different pointers, got ptr1=%v ptr2=%v", ptr1, ptr2)
	}
}

func TestGet_UnhashableError(t *testing.T) {
	// This should not panic even though the error has unhashable fields
	err := &unhashableError{
		msg:  "test",
		data: map[string]any{"key": "value"},
	}

	ptr := errptr.Get(err)
	if ptr == 0 {
		t.Error("Pointer should not be 0 for non-nil error")
	}
}

func TestGet_StandardError(t *testing.T) {
	err := errors.New("standard error")
	ptr := errptr.Get(err)

	if ptr == 0 {
		t.Error("Pointer should not be 0 for non-nil error")
	}
}

func TestGet_Consistency(t *testing.T) {
	// Calling Get multiple times on the same error should return the same pointer
	err := &pointerError{msg: "test"}

	ptr1 := errptr.Get(err)
	ptr2 := errptr.Get(err)
	ptr3 := errptr.Get(err)

	if ptr1 != ptr2 || ptr2 != ptr3 {
		t.Errorf("Multiple calls should return same pointer: %v, %v, %v", ptr1, ptr2, ptr3)
	}
}

func TestGet_WrappedError(t *testing.T) {
	inner := &pointerError{msg: "inner"}
	outer := &pointerError{msg: "outer"}

	ptrInner := errptr.Get(inner)
	ptrOuter := errptr.Get(outer)

	if ptrInner == ptrOuter {
		t.Error("Different errors should have different pointers")
	}
}
