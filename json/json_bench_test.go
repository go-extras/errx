package json_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-extras/errx"
	errxjson "github.com/go-extras/errx/json"
	"github.com/go-extras/errx/stacktrace"
)

// BenchmarkMarshal_Simple measures Marshal() on a basic wrapped error without
// stack traces or attributes.
func BenchmarkMarshal_Simple(b *testing.B) {
	err := errx.Wrap("context", errors.New("base"))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = errxjson.Marshal(err)
	}
}

// BenchmarkMarshal_DeepChain measures Marshal() across a deep wrap chain.
func BenchmarkMarshal_DeepChain(b *testing.B) {
	err := error(errors.New("base"))
	for i := 0; i < 10; i++ {
		err = errx.Wrap("layer", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = errxjson.Marshal(err)
	}
}

// BenchmarkMarshal_MultiError measures Marshal() on a multi-error tree.
func BenchmarkMarshal_MultiError(b *testing.B) {
	multi := errors.Join(
		errx.Wrap("op-1", errors.New("e1")),
		errx.Wrap("op-2", errors.New("e2")),
		errx.Wrap("op-3", errors.New("e3")),
	)
	err := errx.Wrap("aggregate", multi)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = errxjson.Marshal(err)
	}
}

// BenchmarkMarshal_WithStackTrace measures Marshal() on an error with a
// captured stack trace. Frame caching means subsequent marshals reuse the
// resolved frames.
func BenchmarkMarshal_WithStackTrace(b *testing.B) {
	err := stacktrace.Wrap("ctx", errors.New("base"))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = errxjson.Marshal(err)
	}
}

// BenchmarkMarshal_FreshStackTrace measures Marshal() when each iteration uses
// a freshly-captured stack trace, so the frame cache miss is included.
func BenchmarkMarshal_FreshStackTrace(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := stacktrace.Wrap("ctx", fmt.Errorf("base-%d", i))
		_, _ = errxjson.Marshal(err)
	}
}

// BenchmarkMarshal_AttrsScalar measures Marshal() on an attribute-heavy error
// whose values are cheap scalars. This is the worst case for the optimisation:
// per-value encode cost is tiny, so any framing overhead is most visible here.
func BenchmarkMarshal_AttrsScalar(b *testing.B) {
	attrs := errx.Attrs(
		"user_id", 12345,
		"request_id", "abc-123-def-456-789",
		"status", 200,
		"method", "POST",
		"path", "/api/v1/resource/sub",
		"latency_ms", 42,
		"retries", 3,
		"ok", true,
	)
	err := errx.Wrap("operation failed", errors.New("base"), attrs)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = errxjson.Marshal(err)
	}
}

// BenchmarkMarshal_AttrsComplex measures Marshal() on an attribute-heavy error
// whose values are nested maps/structs/slices. Here per-value encode cost
// dominates, so eliminating the duplicate encode should show the biggest win.
func BenchmarkMarshal_AttrsComplex(b *testing.B) {
	type address struct {
		Street string `json:"street"`
		City   string `json:"city"`
		Zip    string `json:"zip"`
	}
	attrs := errx.Attrs(
		"user", map[string]any{
			"id":    12345,
			"name":  "Alice Example",
			"roles": []string{"admin", "user", "auditor"},
			"prefs": map[string]bool{"dark": true, "beta": false, "email": true},
		},
		"address", address{Street: "123 Example Ave", City: "Springfield", Zip: "01234"},
		"tags", []string{"alpha", "beta", "gamma", "delta", "epsilon"},
		"scores", map[string]int{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5},
		"nested", map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"items": []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
				},
			},
		},
	)
	err := errx.Wrap("operation failed", errors.New("base"), attrs)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = errxjson.Marshal(err)
	}
}
