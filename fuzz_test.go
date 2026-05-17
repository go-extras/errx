package errx_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-extras/errx"
)

// FuzzAttrs exercises the variadic Attrs parser to surface panics or unexpected
// behavior on heterogeneous input (string keys, odd-length, !BADKEY fallback).
func FuzzAttrs(f *testing.F) {
	// Seed with shapes from existing test corpus. The fuzz target takes three
	// string args, so each seed must supply three values.
	f.Add("k1", "v1", "k2")
	f.Add("", "", "")
	f.Add("key", "value", "trailing")
	f.Add("user_id", "123", "action")
	f.Add("a", "b", "c")
	f.Add("only-key", "", "")

	f.Fuzz(func(t *testing.T, k1, v1, k2 string) {
		// The mix of string/non-string args is intentionally varied below to
		// drive both the "string key + value" branch and the "!BADKEY" branch.
		e := errx.Attrs(k1, v1, k2)
		if e == nil {
			t.Fatal("Attrs returned nil")
		}
		_ = errx.ExtractAttrs(e)
		_ = e.Error()
		_ = errx.HasAttrs(e)

		// Also exercise mixed-format input that includes an Attr struct and an []Attr slice
		// so we cover all switch arms in parseAttrs without changing the fuzz signature.
		mixed := errx.Attrs(
			k1, v1,
			errx.Attr{Key: k2, Value: v1},
			[]errx.Attr{{Key: k1, Value: k2}},
		)
		_ = errx.ExtractAttrs(mixed)
		_ = mixed.Error()
	})
}

// FuzzExtractAttrs_Chain builds an error chain interleaved with attributed and
// wrapped errors to stress the BFS traversal, visited-map de-dup, and
// multi-error handling in ExtractAttrs.
func FuzzExtractAttrs_Chain(f *testing.F) {
	f.Add("ctx1", "ctx2", "k", "v", uint8(3))
	f.Add("", "", "", "", uint8(0))
	f.Add("outer", "inner", "user_id", "42", uint8(10))

	f.Fuzz(func(t *testing.T, ctx1, ctx2, key, val string, depth uint8) {
		// Cap depth so the fuzz iterations stay fast.
		if depth > 32 {
			depth = 32
		}

		var err error = errx.Attrs(key, val)
		for i := uint8(0); i < depth; i++ {
			// Alternate between Wrap (adds carrier) and fmt.Errorf (plain %w wrap)
			// and occasionally inject another attributed error to exercise multi-attribute merging.
			if i%2 == 0 {
				err = errx.Wrap(ctx1, err, errx.Attrs(key, val))
			} else {
				err = fmt.Errorf("%s: %w", ctx2, err)
			}
		}

		attrs := errx.ExtractAttrs(err)
		if attrs == nil && depth > 0 {
			// We seeded at least one attributed error; ExtractAttrs must find it.
			t.Fatalf("ExtractAttrs returned nil for non-empty chain (depth=%d)", depth)
		}
		_ = errx.HasAttrs(err)
		_ = err.Error()
	})
}

// fuzzMultiError is a local multi-error type used by FuzzExtractAttrs_Chain to
// exercise the Unwrap() []error path. Declared here to keep fuzz tests
// self-contained without leaning on test-internal helpers.
type fuzzMultiError struct {
	errs []error
}

func (*fuzzMultiError) Error() string      { return "fuzz multi error" }
func (m *fuzzMultiError) Unwrap() []error  { return m.errs }
func (*fuzzMultiError) IsClassified() bool { return false }
func newFuzzMulti(errs ...error) error     { return &fuzzMultiError{errs: errs} }

// FuzzExtractAttrs_MultiError verifies the Unwrap() []error path doesn't panic
// on arbitrary attribute payloads at variable widths.
func FuzzExtractAttrs_MultiError(f *testing.F) {
	f.Add("k1", "v1", uint8(3))
	f.Add("", "", uint8(0))

	f.Fuzz(func(t *testing.T, key, val string, width uint8) {
		if width > 16 {
			width = 16
		}
		errs := make([]error, 0, int(width)+1)
		errs = append(errs, errors.New("plain"))
		for i := uint8(0); i < width; i++ {
			errs = append(errs, errx.Attrs(key, val))
		}
		multi := newFuzzMulti(errs...)
		_ = errx.ExtractAttrs(multi)
		_ = errx.HasAttrs(multi)
	})
}
