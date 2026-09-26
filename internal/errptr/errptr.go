// Package errptr provides comparable identities for error traversal without
// requiring the dynamic error type to be comparable.
package errptr

import "unsafe"

// ID identifies an error by both words of its error interface header. Including
// the type distinguishes a wrapper from a first-field cause at the same address
// and distinguishes different zero-size error types that share a data address.
// The zero value means the error has no trackable identity.
type ID struct {
	typ  uintptr
	data uintptr
}

// IsZero reports whether id has no trackable identity.
func (id ID) IsZero() bool {
	return id == ID{}
}

// Get returns the identity of err, or a zero ID for nil and typed-nil errors
// with a nil interface data word.
//
// The type word is the method-table pointer of an error interface; the data word
// points to the concrete error or its boxed value. This does not compare error
// values, so even errors containing maps or slices can be tracked safely.
// Copies of an interface retain their identity. Separate conversions of values
// to interfaces may share storage, especially for zero-size values of the same
// type, so an ID is not guaranteed to distinguish every logical instance.
//
// IDs are only for comparisons during a traversal. Their uintptr fields do not
// keep errors alive and must never be converted back to pointers. This relies
// on Go's current interface representation and non-moving heap; it is not a
// portable object-identity guarantee from the language specification.
func Get(err error) ID {
	if err == nil {
		return ID{}
	}

	type iface struct {
		typ  unsafe.Pointer
		data unsafe.Pointer
	}
	p := (*iface)(unsafe.Pointer(&err))
	if p.data == nil {
		return ID{}
	}
	return ID{typ: uintptr(p.typ), data: uintptr(p.data)}
}
