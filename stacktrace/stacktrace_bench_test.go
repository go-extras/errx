package stacktrace_test

import (
	"errors"
	"testing"

	"github.com/go-extras/errx"
	"github.com/go-extras/errx/stacktrace"
)

// BenchmarkHere measures the cost of capturing the current stack trace via
// Here(). This is the most common entry point and is used inline by callers.
func BenchmarkHere(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stacktrace.Here()
	}
}

// BenchmarkWrap measures the cost of wrapping an error with a stack trace.
func BenchmarkWrap(b *testing.B) {
	cause := errors.New("base")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stacktrace.Wrap("ctx", cause)
	}
}

// BenchmarkExtract_Shallow measures Extract() on a single-trace shallow chain.
func BenchmarkExtract_Shallow(b *testing.B) {
	err := stacktrace.Wrap("ctx", errors.New("base"))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stacktrace.Extract(err)
	}
}

// BenchmarkExtract_Deep measures Extract() across a deeply nested chain so we
// also exercise the chain-walk + frame-resolution caching.
func BenchmarkExtract_Deep(b *testing.B) {
	err := error(errors.New("base"))
	for i := 0; i < 10; i++ {
		err = stacktrace.Wrap("layer", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stacktrace.Extract(err)
	}
}

// BenchmarkExtractAll_Deep measures ExtractAll() walking a 10-layer chain.
func BenchmarkExtractAll_Deep(b *testing.B) {
	err := error(errors.New("base"))
	for i := 0; i < 10; i++ {
		err = stacktrace.Wrap("layer", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stacktrace.ExtractAll(err)
	}
}

// BenchmarkHereDepth measures Here variants with custom depth.
func BenchmarkHereDepth_64(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stacktrace.HereDepth(64)
	}
}

// BenchmarkClassify measures Classify() with stack trace capture.
func BenchmarkClassify(b *testing.B) {
	var ErrSentinel = errx.NewSentinel("sentinel")
	cause := errors.New("base")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stacktrace.Classify(cause, ErrSentinel)
	}
}
