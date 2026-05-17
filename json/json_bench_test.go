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
