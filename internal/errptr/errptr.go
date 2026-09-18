// Package errptr provides utilities for extracting pointer identities from error interfaces.
// This is used internally by errx for circular reference detection with both pointer-based
// and value-based errors.
package errptr

import "unsafe"

// ID uniquely identifies an error interface value for cycle detection.
// It is the (type, data) pair of the interface header, so two errors that
// share a data address but have different dynamic types stay distinct.
// That happens when a wrapper stores its cause as the first field and
// Unwrap returns &outer.cause (so &outer == &outer.cause), and when two
// zero-size values of different types both point at the runtime zerobase.
//
// A zero ID (IsZero) means the error is not trackable: nil or typed-nil.
// Callers treat a zero ID as "skip" for visited-set tracking.
type ID struct {
	typ  uintptr
	data uintptr
}

// IsZero reports whether id is the untrackable sentinel (nil or typed-nil).
func (id ID) IsZero() bool {
	return id.data == 0
}

// Get extracts a comparable identity from an error interface.
// This works for both pointer-based and value-based errors.
//
// For pointer errors, the data word is the pointer to the object.
// For value errors, the data word is the pointer to the copy stored in the interface.
//
// This function is safe to call on any error value (including nil).
// It returns a zero ID for nil and typed-nil errors.
//
// The returned ID uniquely identifies the error instance based on the
// interface header (dynamic type + data pointer), not value equality:
//   - The same error instance will always return the same ID
//   - Different instances with identical content will return different IDs
//   - For value-based errors, each assignment to an interface creates a new copy with a new data pointer
//   - Two errors of different types at the same address return different IDs
//
// # Safety Note on uintptr
//
// The ID fields are uintptr rather than unsafe.Pointer because:
//  1. The value is used only as a map key for identity comparison during a single operation
//  2. We never dereference the pointer or convert it back to unsafe.Pointer
//  3. The actual error values are kept alive by the call stack during traversal
//  4. uintptr is hashable and can be used as a map key, while unsafe.Pointer cannot
//
// While uintptr values are not guaranteed to remain stable across garbage collections
// in a hypothetical moving GC, this is safe for our use case because:
//   - The ID is only used for comparison within a single function call
//   - The errors being tracked are live on the stack and won't be moved during the operation
//   - We don't store the ID beyond the scope of the error traversal
//
// This is a standard pattern in Go for pointer identity tracking (similar to how
// reflect.Value.Pointer() is used) and is safe under the current and foreseeable
// Go memory model.
//
// Example:
//
//	err1 := &MyError{msg: "test"}
//	err2 := err1  // Same instance
//	ptr1 := errptr.Get(err1)
//	ptr2 := errptr.Get(err2)
//	// ptr1 == ptr2 (same instance)
//
//	err3 := &MyError{msg: "test"}  // Different instance, same content
//	ptr3 := errptr.Get(err3)
//	// ptr1 != ptr3 (different instances)
func Get(err error) ID {
	if err == nil {
		return ID{}
	}

	// An interface in Go is represented as two pointers:
	// - type pointer (points to type information)
	// - data pointer (points to the actual data)
	// Both words are required: the data pointer alone collides when a
	// wrapper's first field is the cause (same address, different type)
	// or when distinct zero-size types share the runtime zerobase.
	type iface struct {
		typ  unsafe.Pointer
		data unsafe.Pointer
	}
	p := (*iface)(unsafe.Pointer(&err))
	// Typed-nil errors (e.g. `var p *MyErr; var e error = p`) have a non-nil
	// type pointer but a nil data pointer. They do not have a trackable
	// identity, so we treat them the same as a nil error (zero ID). Callers
	// already special-case IsZero as "skip" for visited-set tracking, which
	// avoids collisions between multiple typed-nil values in the same error chain.
	if p.data == nil {
		return ID{}
	}
	return ID{typ: uintptr(p.typ), data: uintptr(p.data)}
}
