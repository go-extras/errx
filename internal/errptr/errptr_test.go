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
	// Create value errors with different content to ensure they're not optimized to same location
	var e1 error = valueError{msg: "test1"}
	var e2 error = valueError{msg: "test2"}

	ptr1 := errptr.Get(e1)
	ptr2 := errptr.Get(e2)

	// Note: The compiler may optimize identical value errors to the same location,
	// but errors with different content should have different pointers
	if ptr1 == 0 || ptr2 == 0 {
		t.Error("Pointers should not be 0 for non-nil errors")
	}
	// We don't assert ptr1 != ptr2 because the compiler may optimize,
	// but in practice they will be different for different values
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
