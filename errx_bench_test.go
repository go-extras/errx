package errx_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-extras/errx"
)

// Benchmark sentinel creation
func BenchmarkNewSentinel(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.NewSentinel("benchmark sentinel")
	}
}

func BenchmarkNewSentinel_WithParents(b *testing.B) {
	parent1 := errx.NewSentinel("parent1")
	parent2 := errx.NewSentinel("parent2")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.NewSentinel("child", parent1, parent2)
	}
}

// Benchmark classification operations
func BenchmarkClassify_Simple(b *testing.B) {
	err := errors.New("test error")
	sentinel := errx.NewSentinel("test sentinel")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.Classify(err, sentinel)
	}
}

func BenchmarkClassify_Multiple(b *testing.B) {
	err := errors.New("test error")
	s1 := errx.NewSentinel("sentinel1")
	s2 := errx.NewSentinel("sentinel2")
	s3 := errx.NewSentinel("sentinel3")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.Classify(err, s1, s2, s3)
	}
}

func BenchmarkClassify_WithHierarchy(b *testing.B) {
	err := errors.New("test error")
	parent := errx.NewSentinel("parent")
	child := errx.NewSentinel("child", parent)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.Classify(err, child)
	}
}

// Benchmark wrapping operations
func BenchmarkWrap_Simple(b *testing.B) {
	err := errors.New("test error")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.Wrap("context", err)
	}
}

func BenchmarkWrap_WithSentinel(b *testing.B) {
	err := errors.New("test error")
	sentinel := errx.NewSentinel("test sentinel")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.Wrap("context", err, sentinel)
	}
}

func BenchmarkWrap_WithMultipleSentinels(b *testing.B) {
	err := errors.New("test error")
	s1 := errx.NewSentinel("sentinel1")
	s2 := errx.NewSentinel("sentinel2")
	s3 := errx.NewSentinel("sentinel3")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.Wrap("context", err, s1, s2, s3)
	}
}

// Benchmark error checking with errors.Is
func BenchmarkErrorsIs_Shallow(b *testing.B) {
	sentinel := errx.NewSentinel("test sentinel")
	err := errx.Classify(errors.New("test"), sentinel)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errors.Is(err, sentinel)
	}
}

func BenchmarkErrorsIs_Deep(b *testing.B) {
	sentinel := errx.NewSentinel("test sentinel")
	err := errors.New("base error")
	err = errx.Classify(err, sentinel)
	err = fmt.Errorf("level1: %w", err)
	err = fmt.Errorf("level2: %w", err)
	err = fmt.Errorf("level3: %w", err)
	err = fmt.Errorf("level4: %w", err)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errors.Is(err, sentinel)
	}
}

func BenchmarkErrorsIs_WithHierarchy(b *testing.B) {
	parent := errx.NewSentinel("parent")
	child := errx.NewSentinel("child", parent)
	err := errx.Classify(errors.New("test"), child)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errors.Is(err, parent)
	}
}

// Benchmark displayable operations
func BenchmarkNewDisplayable(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.NewDisplayable("user message")
	}
}

func BenchmarkIsDisplayable_Shallow(b *testing.B) {
	err := errx.NewDisplayable("user message")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.IsDisplayable(err)
	}
}

func BenchmarkIsDisplayable_Deep(b *testing.B) {
	var err error
	err = errx.NewDisplayable("user message")
	err = fmt.Errorf("level1: %w", err)
	err = fmt.Errorf("level2: %w", err)
	err = fmt.Errorf("level3: %w", err)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.IsDisplayable(err)
	}
}

func BenchmarkDisplayText_Shallow(b *testing.B) {
	err := errx.NewDisplayable("user message")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.DisplayText(err)
	}
}

func BenchmarkDisplayText_Deep(b *testing.B) {
	var err error
	err = errx.NewDisplayable("user message")
	err = fmt.Errorf("level1: %w", err)
	err = fmt.Errorf("level2: %w", err)
	err = fmt.Errorf("level3: %w", err)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.DisplayText(err)
	}
}

func BenchmarkDisplayText_NoDisplayable(b *testing.B) {
	err := errors.New("regular error")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.DisplayText(err)
	}
}

// Benchmark attributed operations
func BenchmarkWithAttrs_KeyValuePairs(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.WithAttrs("key1", "value1", "key2", 123, "key3", true)
	}
}

func BenchmarkWithAttrs_AttrStructs(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.WithAttrs(
			errx.Attr{Key: "key1", Value: "value1"},
			errx.Attr{Key: "key2", Value: 123},
			errx.Attr{Key: "key3", Value: true},
		)
	}
}

func BenchmarkFromAttrMap_Small(b *testing.B) {
	attrs := map[string]any{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.FromAttrMap(attrs)
	}
}

func BenchmarkFromAttrMap_Large(b *testing.B) {
	attrs := map[string]any{
		"key1":  "value1",
		"key2":  123,
		"key3":  true,
		"key4":  "value4",
		"key5":  456,
		"key6":  false,
		"key7":  "value7",
		"key8":  789,
		"key9":  true,
		"key10": "value10",
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.FromAttrMap(attrs)
	}
}

func BenchmarkHasAttrs_Shallow(b *testing.B) {
	err := errx.WithAttrs("key", "value")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.HasAttrs(err)
	}
}

func BenchmarkHasAttrs_Deep(b *testing.B) {
	var err error
	err = errx.WithAttrs("key", "value")
	err = fmt.Errorf("level1: %w", err)
	err = fmt.Errorf("level2: %w", err)
	err = fmt.Errorf("level3: %w", err)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.HasAttrs(err)
	}
}

func BenchmarkExtractAttrs_Small(b *testing.B) {
	err := errx.WithAttrs("key1", "value1", "key2", 123)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.ExtractAttrs(err)
	}
}

func BenchmarkExtractAttrs_Large(b *testing.B) {
	err := errx.WithAttrs(
		"key1", "value1",
		"key2", 123,
		"key3", true,
		"key4", "value4",
		"key5", 456,
		"key6", false,
		"key7", "value7",
		"key8", 789,
	)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.ExtractAttrs(err)
	}
}

func BenchmarkExtractAttrs_Deep(b *testing.B) {
	var err error
	err = errx.WithAttrs("key1", "value1", "key2", 123)
	err = fmt.Errorf("level1: %w", err)
	err = fmt.Errorf("level2: %w", err)
	err = fmt.Errorf("level3: %w", err)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errx.ExtractAttrs(err)
	}
}

// Benchmark combined operations (realistic scenarios)
func BenchmarkCombined_SimpleClassification(b *testing.B) {
	ErrNotFound := errx.NewSentinel("not found")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err := errors.New("record not found")
		err = errx.Classify(err, ErrNotFound)
		_ = errors.Is(err, ErrNotFound)
	}
}

func BenchmarkCombined_WrapWithClassification(b *testing.B) {
	ErrDatabase := errx.NewSentinel("database")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err := errors.New("connection failed")
		err = errx.Wrap("query failed", err, ErrDatabase)
		_ = errors.Is(err, ErrDatabase)
	}
}

func BenchmarkCombined_RichError(b *testing.B) {
	ErrValidation := errx.NewSentinel("validation")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		baseErr := errors.New("invalid input")
		displayErr := errx.NewDisplayable("Please provide valid input")
		attrErr := errx.WithAttrs("field", "email", "value", "invalid@")
		err := errx.Classify(baseErr, displayErr, attrErr, ErrValidation)

		_ = errors.Is(err, ErrValidation)
		_ = errx.IsDisplayable(err)
		_ = errx.DisplayText(err)
		_ = errx.ExtractAttrs(err)
	}
}

func BenchmarkCombined_ErrorChain(b *testing.B) {
	ErrNotFound := errx.NewSentinel("not found")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err := errors.New("record missing")
		err = errx.Classify(err, ErrNotFound)
		err = errx.Wrap("database query failed", err)
		err = fmt.Errorf("handler error: %w", err)
		err = fmt.Errorf("request failed: %w", err)

		_ = errors.Is(err, ErrNotFound)
	}
}

func BenchmarkCombined_APIErrorHandling(b *testing.B) {
	ErrValidation := errx.NewSentinel("validation")
	ErrNotFound := errx.NewSentinel("not found")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Create service error
		var err error
		err = errx.NewDisplayable("Email is required")
		err = errx.Classify(err, ErrValidation)
		err = errx.Classify(err, errx.WithAttrs("field", "email"))

		// Handler logic
		statusCode := 500
		if errors.Is(err, ErrNotFound) {
			statusCode = 404
		} else if errors.Is(err, ErrValidation) {
			statusCode = 400
		}

		var message string
		if errx.IsDisplayable(err) {
			message = errx.DisplayText(err)
		}

		var attrs []errx.Attr
		if errx.HasAttrs(err) {
			attrs = errx.ExtractAttrs(err)
		}

		// Use the values to prevent optimization
		_ = statusCode
		_ = message
		_ = attrs
	}
}

// Benchmark comparison with standard library
func BenchmarkStdlib_ErrorsNew(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errors.New("test error")
	}
}

func BenchmarkStdlib_FmtErrorf(b *testing.B) {
	err := errors.New("base error")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fmt.Errorf("wrapped: %w", err)
	}
}

func BenchmarkStdlib_ErrorsIs(b *testing.B) {
	target := errors.New("target")
	err := fmt.Errorf("wrapped: %w", target)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = errors.Is(err, target)
	}
}
