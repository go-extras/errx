package stacktrace_test

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/go-extras/errx"
	"github.com/go-extras/errx/stacktrace"
)

// TestHere verifies that Here() captures stack traces correctly
func TestHere(t *testing.T) {
	// Create an error with a stack trace
	baseErr := errors.New("base error")
	err := errx.Wrap("context", baseErr, stacktrace.Here())

	// Extract the stack trace
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Fatal("Expected stack trace, got nil")
	}

	if len(frames) == 0 {
		t.Fatal("Expected non-empty stack trace")
	}

	// Verify that the first frame contains this test function
	found := false
	for _, frame := range frames {
		if strings.Contains(frame.Function, "TestHere") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected stack trace to contain TestHere function")
	}
}

// TestExtractNil verifies that Extract returns nil for nil errors
func TestExtractNil(t *testing.T) {
	frames := stacktrace.Extract(nil)
	if frames != nil {
		t.Errorf("Expected nil for nil error, got %v", frames)
	}
}

// TestExtractNoTrace verifies that Extract returns nil for errors without traces
func TestExtractNoTrace(t *testing.T) {
	err := errors.New("no trace")
	frames := stacktrace.Extract(err)
	if frames != nil {
		t.Errorf("Expected nil for error without trace, got %v", frames)
	}
}

// TestExtractFromWrappedError verifies that Extract finds traces in wrapped errors
func TestExtractFromWrappedError(t *testing.T) {
	baseErr := errors.New("base")
	traced := errx.Classify(baseErr, stacktrace.Here())
	wrapped := errx.Wrap("outer", traced)

	frames := stacktrace.Extract(wrapped)
	if frames == nil {
		t.Fatal("Expected to find stack trace in wrapped error")
	}

	if len(frames) == 0 {
		t.Error("Expected non-empty stack trace")
	}
}

// TestWrap verifies that stacktrace.Wrap automatically captures traces
func TestWrap(t *testing.T) {
	baseErr := errors.New("base error")
	err := stacktrace.Wrap("operation failed", baseErr)

	// Verify the error message
	expected := "operation failed: base error"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}

	// Verify stack trace was captured
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Fatal("Expected stack trace from Wrap, got nil")
	}

	// Verify the trace contains this test function
	found := false
	for _, frame := range frames {
		if strings.Contains(frame.Function, "TestWrap") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected stack trace to contain TestWrap function")
	}
}

// TestWrapNil verifies that stacktrace.Wrap returns nil for nil errors
func TestWrapNil(t *testing.T) {
	err := stacktrace.Wrap("context", nil)
	if err != nil {
		t.Errorf("Expected nil for nil cause, got %v", err)
	}
}

// TestWrapWithClassifications verifies that Wrap works with additional classifications
func TestWrapWithClassifications(t *testing.T) {
	var ErrNotFound = errx.NewSentinel("not found")
	baseErr := errors.New("base")
	err := stacktrace.Wrap("failed", baseErr, ErrNotFound)

	// Verify classification
	if !errors.Is(err, ErrNotFound) {
		t.Error("Expected error to match ErrNotFound sentinel")
	}

	// Verify stack trace
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Error("Expected stack trace")
	}
}

// TestClassify verifies that stacktrace.Classify automatically captures traces
func TestClassify(t *testing.T) {
	var ErrDatabase = errx.NewSentinel("database error")
	baseErr := errors.New("connection failed")
	err := stacktrace.Classify(baseErr, ErrDatabase)

	// Verify classification
	if !errors.Is(err, ErrDatabase) {
		t.Error("Expected error to match ErrDatabase sentinel")
	}

	// Verify original message is preserved
	if err.Error() != "connection failed" {
		t.Errorf("Expected original message, got %q", err.Error())
	}

	// Verify stack trace
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Error("Expected stack trace from Classify")
	}
}

// TestClassifyNil verifies that stacktrace.Classify returns nil for nil errors
func TestClassifyNil(t *testing.T) {
	err := stacktrace.Classify(nil)
	if err != nil {
		t.Errorf("Expected nil for nil cause, got %v", err)
	}
}

// TestFrameString verifies the Frame.String() method
func TestFrameString(t *testing.T) {
	frame := stacktrace.Frame{
		File:     "/path/to/file.go",
		Line:     42,
		Function: "github.com/example/pkg.Function",
	}

	expected := "/path/to/file.go:42 github.com/example/pkg.Function"
	if frame.String() != expected {
		t.Errorf("Expected %q, got %q", expected, frame.String())
	}
}

// TestMultipleTraces verifies that only the first trace is extracted
func TestMultipleTraces(t *testing.T) {
	baseErr := errors.New("base")
	err1 := errx.Classify(baseErr, stacktrace.Here())
	err2 := errx.Wrap("outer", err1, stacktrace.Here())

	frames := stacktrace.Extract(err2)
	if frames == nil {
		t.Fatal("Expected stack trace")
	}

	// The first trace found should be from the outer wrap
	// (errors.As finds the first match in the chain)
	found := false
	for _, frame := range frames {
		if strings.Contains(frame.Function, "TestMultipleTraces") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected stack trace to contain TestMultipleTraces")
	}
}

// TestIntegrationWithDisplayable verifies stacktrace works with displayable errors
func TestIntegrationWithDisplayable(t *testing.T) {
	displayErr := errx.NewDisplayable("User not found")
	err := stacktrace.Wrap("fetch failed", displayErr)

	// Verify displayable message
	if !errx.IsDisplayable(err) {
		t.Error("Expected error to be displayable")
	}
	if errx.DisplayText(err) != "User not found" {
		t.Errorf("Expected displayable text 'User not found', got %q", errx.DisplayText(err))
	}

	// Verify stack trace
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Error("Expected stack trace")
	}
}

// TestIntegrationWithAttrs verifies stacktrace works with attributed errors
func TestIntegrationWithAttrs(t *testing.T) {
	attrErr := errx.Attrs("user_id", 123, "action", "delete")
	err := stacktrace.Wrap("operation failed", attrErr)

	// Verify attributes
	if !errx.HasAttrs(err) {
		t.Error("Expected error to have attributes")
	}
	attrs := errx.ExtractAttrs(err)
	if len(attrs) != 2 {
		t.Errorf("Expected 2 attributes, got %d", len(attrs))
	}

	// Verify stack trace
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Error("Expected stack trace")
	}
}

// TestComplexErrorChain verifies stacktrace works in complex error chains
func TestComplexErrorChain(t *testing.T) {
	var ErrNotFound = errx.NewSentinel("not found")

	// Build a complex error chain
	baseErr := errors.New("database error")
	attrErr := errx.Attrs("table", "users", "id", 42)
	displayErr := errx.NewDisplayable("Record not found")

	err := stacktrace.Wrap("query failed",
		errx.Wrap("fetch user",
			errx.Classify(baseErr, attrErr, displayErr, ErrNotFound)))

	// Verify all features work together
	if !errors.Is(err, ErrNotFound) {
		t.Error("Expected error to match ErrNotFound")
	}
	if !errx.IsDisplayable(err) {
		t.Error("Expected error to be displayable")
	}
	if !errx.HasAttrs(err) {
		t.Error("Expected error to have attributes")
	}

	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Error("Expected stack trace")
	}
}

// TestClassifyNew verifies that stacktrace.ClassifyNew creates and classifies errors with traces
func TestClassifyNew(t *testing.T) {
	var ErrDatabase = errx.NewSentinel("database error")
	err := stacktrace.ClassifyNew("connection timeout", ErrDatabase)

	// Verify error message
	if err.Error() != "connection timeout" {
		t.Errorf("Expected 'connection timeout', got %q", err.Error())
	}

	// Verify classification
	if !errors.Is(err, ErrDatabase) {
		t.Error("Expected error to match ErrDatabase sentinel")
	}

	// Verify stack trace was captured
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Fatal("Expected stack trace from ClassifyNew, got nil")
	}

	// Verify the trace contains this test function
	found := false
	for _, frame := range frames {
		if strings.Contains(frame.Function, "TestClassifyNew") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected stack trace to contain TestClassifyNew function")
	}
}

// TestClassifyNewMultipleClassifications verifies ClassifyNew with multiple classifications
func TestClassifyNewMultipleClassifications(t *testing.T) {
	var (
		ErrDatabase  = errx.NewSentinel("database error")
		ErrRetryable = errx.NewSentinel("retryable error")
	)

	err := stacktrace.ClassifyNew("temporary failure", ErrDatabase, ErrRetryable)

	// Verify both classifications
	if !errors.Is(err, ErrDatabase) {
		t.Error("Expected error to match ErrDatabase")
	}
	if !errors.Is(err, ErrRetryable) {
		t.Error("Expected error to match ErrRetryable")
	}

	// Verify stack trace
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Error("Expected stack trace")
	}
}

// TestClassifyNewWithDisplayable verifies ClassifyNew with displayable errors
func TestClassifyNewWithDisplayable(t *testing.T) {
	var ErrNotFound = errx.NewSentinel("not found")
	displayErr := errx.NewDisplayable("Resource not found")

	err := stacktrace.ClassifyNew("user record missing", ErrNotFound, displayErr)

	// Verify error message
	if err.Error() != "user record missing" {
		t.Errorf("Expected 'user record missing', got %q", err.Error())
	}

	// Verify classification
	if !errors.Is(err, ErrNotFound) {
		t.Error("Expected error to match ErrNotFound")
	}

	// Verify displayable
	if !errx.IsDisplayable(err) {
		t.Error("Expected error to be displayable")
	}
	if errx.DisplayText(err) != "Resource not found" {
		t.Errorf("Expected displayable text 'Resource not found', got %q", errx.DisplayText(err))
	}

	// Verify stack trace
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Error("Expected stack trace")
	}
}

// TestClassifyNewWithAttributes verifies ClassifyNew with attributes
func TestClassifyNewWithAttributes(t *testing.T) {
	var ErrDatabase = errx.NewSentinel("database error")
	attrErr := errx.Attrs("query", "SELECT * FROM users", "timeout_ms", 5000)

	err := stacktrace.ClassifyNew("query timeout", ErrDatabase, attrErr)

	// Verify classification
	if !errors.Is(err, ErrDatabase) {
		t.Error("Expected error to match ErrDatabase")
	}

	// Verify attributes
	if !errx.HasAttrs(err) {
		t.Error("Expected error to have attributes")
	}

	attrs := errx.ExtractAttrs(err)
	if len(attrs) != 2 {
		t.Errorf("Expected 2 attributes, got %d", len(attrs))
	}

	// Verify stack trace
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Error("Expected stack trace")
	}
}

// TestClassifyNewNoClassifications verifies ClassifyNew without classifications
func TestClassifyNewNoClassifications(t *testing.T) {
	err := stacktrace.ClassifyNew("simple error")

	// Verify error message
	if err.Error() != "simple error" {
		t.Errorf("Expected 'simple error', got %q", err.Error())
	}

	// Verify stack trace is still captured
	frames := stacktrace.Extract(err)
	if frames == nil {
		t.Error("Expected stack trace even without classifications")
	}
}

// fakeTracer is an external Tracer implementation used to verify that Extract
// and ExtractAll discover third-party tracers in an error chain.
type fakeTracer struct {
	msg    string
	frames []stacktrace.Frame
	cause  error
}

func (f *fakeTracer) Error() string { return f.msg }
func (f *fakeTracer) Unwrap() error { return f.cause }
func (f *fakeTracer) Frames() []stacktrace.Frame {
	return f.frames
}

// TestFormatPlusV verifies that %+v renders the captured stack frames.
func TestFormatPlusV(t *testing.T) {
	tr := stacktrace.Here()

	plus := fmt.Sprintf("%+v", tr)
	if !strings.Contains(plus, "TestFormatPlusV") {
		t.Errorf("Expected %%+v output to contain TestFormatPlusV, got:\n%s", plus)
	}
	// pkg/errors style: "\n<function>\n\t<file>:<line>"
	if !strings.Contains(plus, ":") || !strings.Contains(plus, "\n\t") {
		t.Errorf("Expected %%+v output to look like pkg/errors stack lines, got:\n%s", plus)
	}
}

// TestFormatPlainV verifies that %v and %s still render the underlying error
// message (backwards-compatible behaviour).
func TestFormatPlainV(t *testing.T) {
	tr := stacktrace.Here()

	plain := fmt.Sprintf("%v", tr)
	if strings.Contains(plain, "TestFormatPlainV") {
		t.Errorf("Expected %%v output to NOT contain stack frames, got:\n%s", plain)
	}
	if !strings.HasPrefix(plain, "stack trace:") {
		t.Errorf("Expected %%v output to start with 'stack trace:', got %q", plain)
	}

	s := fmt.Sprintf("%s", tr)
	if s != plain {
		t.Errorf("Expected %%s output to match %%v output, got %q vs %q", s, plain)
	}
}

// TestFramesCachedSingleSymbolResolution verifies that frames are resolved only
// once and concurrent callers all observe the same slice header.
func TestFramesCachedSingleSymbolResolution(t *testing.T) {
	tr := stacktrace.Here()

	// First call resolves; capture pointer/length to compare against subsequent
	// calls. The cache invariant guarantees identical slice contents.
	first := stacktrace.Extract(errx.Classify(errors.New("e"), tr))
	if first == nil {
		t.Fatal("expected frames")
	}

	const concurrent = 32
	results := make([][]stacktrace.Frame, concurrent)
	var wg sync.WaitGroup
	wg.Add(concurrent)
	for i := 0; i < concurrent; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx] = stacktrace.Extract(errx.Classify(errors.New("e"), tr))
		}(i)
	}
	wg.Wait()

	for i, r := range results {
		if len(r) != len(first) {
			t.Fatalf("result %d: expected %d frames, got %d", i, len(first), len(r))
		}
		for j := range r {
			if r[j] != first[j] {
				t.Fatalf("result %d frame %d differs: %+v vs %+v", i, j, r[j], first[j])
			}
		}
	}
}

// TestExtractAllMultipleTraces verifies that ExtractAll collects every captured
// trace in outermost-first order.
func TestExtractAllMultipleTraces(t *testing.T) {
	base := errors.New("base")
	inner := errx.Classify(base, stacktrace.Here())
	middle := errx.Wrap("middle", inner, stacktrace.Here())
	outer := errx.Wrap("outer", middle, stacktrace.Here())

	all := stacktrace.ExtractAll(outer)
	if len(all) != 3 {
		t.Fatalf("expected 3 traces, got %d", len(all))
	}

	// All traces should contain this test function.
	for i, frames := range all {
		found := false
		for _, f := range frames {
			if strings.Contains(f.Function, "TestExtractAllMultipleTraces") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("trace %d did not contain TestExtractAllMultipleTraces", i)
		}
	}
}

// TestExtractAllNilAndSingle verifies edge cases for ExtractAll.
func TestExtractAllNilAndSingle(t *testing.T) {
	if got := stacktrace.ExtractAll(nil); got != nil {
		t.Errorf("expected nil for nil error, got %v", got)
	}

	if got := stacktrace.ExtractAll(errors.New("plain")); got != nil {
		t.Errorf("expected nil for error without trace, got %v", got)
	}

	single := stacktrace.Classify(errors.New("base"))
	all := stacktrace.ExtractAll(single)
	if len(all) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(all))
	}
}

// TestHereDepthCapturesDeeperStack verifies that HereDepth honours the supplied
// depth. We build a sufficiently deep recursion (>32 frames) and capture with
// depth=64, verifying we observed more than the default 32 frames.
func TestHereDepthCapturesDeeperStack(t *testing.T) {
	const recursionDepth = 50

	frames := captureWithDepth(recursionDepth, 64)
	if len(frames) <= stacktrace.DefaultMaxDepth {
		t.Fatalf("HereDepth(64) captured only %d frames; expected > %d", len(frames), stacktrace.DefaultMaxDepth)
	}

	defaultFrames := captureWithDepth(recursionDepth, 0) // 0 -> default
	if len(defaultFrames) > stacktrace.DefaultMaxDepth {
		t.Fatalf("default capture returned %d frames; expected <= %d", len(defaultFrames), stacktrace.DefaultMaxDepth)
	}
}

// recurseAndCapture recurses `remaining` times and then captures a stack trace
// at the configured depth.
func recurseAndCapture(remaining, depth int) []stacktrace.Frame {
	if remaining > 0 {
		return recurseAndCapture(remaining-1, depth)
	}
	var tr errx.Classified
	if depth > 0 {
		tr = stacktrace.HereDepth(depth)
	} else {
		tr = stacktrace.Here()
	}
	return stacktrace.Extract(errx.Classify(errors.New("deep"), tr))
}

func captureWithDepth(recursion, depth int) []stacktrace.Frame {
	return recurseAndCapture(recursion, depth)
}

// TestWrapDepthAndClassifyDepth verifies the *Depth variants honour the
// supplied depth and remain backwards-compatible.
func TestWrapDepthAndClassifyDepth(t *testing.T) {
	base := errors.New("base")
	werr := stacktrace.WrapDepth("ctx", base, 4)
	frames := stacktrace.Extract(werr)
	if len(frames) == 0 || len(frames) > 4 {
		t.Errorf("WrapDepth(4): expected 1..4 frames, got %d", len(frames))
	}

	cerr := stacktrace.ClassifyDepth(base, 4)
	cframes := stacktrace.Extract(cerr)
	if len(cframes) == 0 || len(cframes) > 4 {
		t.Errorf("ClassifyDepth(4): expected 1..4 frames, got %d", len(cframes))
	}

	nerr := stacktrace.ClassifyNewDepth("new", 4)
	nframes := stacktrace.Extract(nerr)
	if len(nframes) == 0 || len(nframes) > 4 {
		t.Errorf("ClassifyNewDepth(4): expected 1..4 frames, got %d", len(nframes))
	}

	// nil cause should return nil for WrapDepth/ClassifyDepth.
	if got := stacktrace.WrapDepth("ctx", nil, 4); got != nil {
		t.Errorf("expected nil from WrapDepth(nil), got %v", got)
	}
	if got := stacktrace.ClassifyDepth(nil, 4); got != nil {
		t.Errorf("expected nil from ClassifyDepth(nil), got %v", got)
	}
}

// TestExtractPicksUpExternalTracer verifies that an error implementing only the
// public Tracer interface is recognised by Extract.
func TestExtractPicksUpExternalTracer(t *testing.T) {
	external := &fakeTracer{
		msg: "external",
		frames: []stacktrace.Frame{
			{File: "remote.go", Line: 12, Function: "remote.Handler"},
		},
	}

	frames := stacktrace.Extract(external)
	if len(frames) != 1 || frames[0].Function != "remote.Handler" {
		t.Fatalf("Extract did not pick up external Tracer, got %+v", frames)
	}

	// Wrap it and check chain traversal still works.
	wrapped := fmt.Errorf("outer: %w", external)
	frames = stacktrace.Extract(wrapped)
	if len(frames) != 1 || frames[0].Function != "remote.Handler" {
		t.Fatalf("Extract did not pick up wrapped external Tracer, got %+v", frames)
	}
}

// TestExtractAllPicksUpExternalTracer verifies that ExtractAll collects both
// internal and external tracers.
func TestExtractAllPicksUpExternalTracer(t *testing.T) {
	external := &fakeTracer{
		msg: "external",
		frames: []stacktrace.Frame{
			{File: "remote.go", Line: 12, Function: "remote.Handler"},
		},
	}
	wrapped := stacktrace.Wrap("outer", external)

	traces := stacktrace.ExtractAll(wrapped)
	if len(traces) != 2 {
		t.Fatalf("expected 2 traces, got %d", len(traces))
	}

	// outermost should be the internal (Wrap) trace.
	foundLocal := false
	for _, f := range traces[0] {
		if strings.Contains(f.Function, "TestExtractAllPicksUpExternalTracer") {
			foundLocal = true
			break
		}
	}
	if !foundLocal {
		t.Errorf("outermost trace missing local frame: %+v", traces[0])
	}

	// innermost should be the external tracer frames.
	if len(traces[1]) != 1 || traces[1][0].Function != "remote.Handler" {
		t.Errorf("innermost trace not external: %+v", traces[1])
	}
}

// TestFormatPlusVUsesCache exercises %+v concurrently to assert race-safety of
// the cached frames slice.
func TestFormatPlusVConcurrent(t *testing.T) {
	tr := stacktrace.Here()

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = fmt.Sprintf("%+v", tr)
			_ = fmt.Sprintf("%v", tr)
		}()
	}
	wg.Wait()
}
