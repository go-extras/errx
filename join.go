package errx

// joinError aggregates multiple errors into a single error value. It mirrors
// the type returned by the standard library's errors.Join: its Error method
// renders each member on its own line, and its Unwrap method exposes the full
// list so errors.Is/errors.As (and errx's own ExtractAttrs/HasAttrs traversal)
// can walk every branch.
type joinError struct {
	errs []error
}

// Error renders each joined error on its own line, matching the format used by
// the standard library's errors.Join. Each member's Error() is called exactly
// once (single pass), mirroring the stdlib implementation.
func (e *joinError) Error() string {
	if len(e.errs) == 1 {
		return e.errs[0].Error()
	}
	b := []byte(e.errs[0].Error())
	for _, err := range e.errs[1:] {
		b = append(b, '\n')
		b = append(b, err.Error()...)
	}
	return string(b)
}

// Unwrap exposes the joined errors so errors.Is, errors.As, and errx's
// attribute/displayable traversal can descend into every branch.
func (e *joinError) Unwrap() []error {
	return e.errs
}

// Join combines multiple errors into a single error that reports every non-nil
// member. It mirrors the standard library's errors.Join: nil arguments are
// dropped, and if every argument is nil, Join returns nil.
//
// The result implements Unwrap() []error, so it works transparently with
// errors.Is and errors.As, and errx helpers such as ExtractAttrs, HasAttrs, and
// DisplayText traverse every joined branch. The json subpackage serializes the
// members into the causes array (by default).
//
// Join is intentionally orthogonal to classification: to classify an aggregate,
// compose it with the existing API rather than threading classifications through
// Join itself:
//
//	var ErrBatch = errx.NewSentinel("batch failed")
//
//	err := errx.Classify(errx.Join(err1, err2), ErrBatch, errx.Attrs("count", 2))
//	errors.Is(err, ErrBatch)        // true
//	attrs := errx.ExtractAttrs(err) // [{count 2}]
func Join(errs ...error) error {
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	e := &joinError{errs: make([]error, 0, n)}
	for _, err := range errs {
		if err != nil {
			e.errs = append(e.errs, err)
		}
	}
	return e
}
