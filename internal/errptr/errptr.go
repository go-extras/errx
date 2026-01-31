// Package errptr provides utilities for extracting pointer identities from error interfaces.
// This is used internally by errx for circular reference detection with both pointer-based
// and value-based errors.
package errptr

import "unsafe"

// Get extracts the data pointer from an error interface.
// This works for both pointer-based and value-based errors.
//
// For pointer errors, it returns the pointer to the object.
// For value errors, it returns the pointer to the copy stored in the interface.
//
// This function is safe to call on any error value (including nil).
// It returns 0 for nil errors.
//
// The returned pointer uniquely identifies the error instance based on pointer identity,
// not value equality. This means:
//   - The same error instance will always return the same pointer
//   - Different instances with identical content will return different pointers
//   - For value-based errors, each assignment to an interface creates a new copy with a new pointer
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
func Get(err error) uintptr {
	if err == nil {
		return 0
	}

	// An interface in Go is represented as two pointers:
	// - type pointer (points to type information)
	// - data pointer (points to the actual data)
	// We extract the data pointer which uniquely identifies the error instance.
	type iface struct {
		typ  unsafe.Pointer
		data unsafe.Pointer
	}
	return uintptr((*iface)(unsafe.Pointer(&err)).data)
}
