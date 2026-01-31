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
// # Safety Note on uintptr
//
// This function returns uintptr instead of unsafe.Pointer because:
//  1. The value is used only as a map key for identity comparison during a single operation
//  2. We never dereference the pointer or convert it back to unsafe.Pointer
//  3. The actual error values are kept alive by the call stack during traversal
//  4. uintptr is hashable and can be used as a map key, while unsafe.Pointer cannot
//
// While uintptr values are not guaranteed to remain stable across garbage collections
// in a hypothetical moving GC, this is safe for our use case because:
//   - The uintptr is only used for comparison within a single function call
//   - The errors being tracked are live on the stack and won't be moved during the operation
//   - We don't store the uintptr beyond the scope of the error traversal
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
