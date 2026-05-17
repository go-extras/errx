// Package stacktrace provides optional stack trace support for errx errors.
//
// This package extends errx with stack trace capabilities while keeping the core
// errx package minimal and zero-dependency. It offers two usage patterns:
//
//  1. Per-error opt-in using Here() as a Classified:
//     err := errx.Wrap("context", cause, ErrNotFound, stacktrace.Here())
//
//  2. Convenience functions that automatically capture traces:
//     err := stacktrace.Wrap("context", cause, ErrNotFound)
//
// Stack traces can be extracted from any error in the chain using Extract():
//
//	frames := stacktrace.Extract(err)
//	for _, frame := range frames {
//	    fmt.Printf("%s:%d %s\n", frame.File, frame.Line, frame.Function)
//	}
//
// To retrieve every captured trace in a multi-wrap chain, use ExtractAll():
//
//	for _, frames := range stacktrace.ExtractAll(err) {
//	    // outermost trace first
//	}
//
// Traced errors implement fmt.Formatter so that fmt.Sprintf("%+v", err) renders
// the captured stack frames in the de-facto pkg/errors style. The plain "%v" and
// "%s" verbs still produce the underlying error message.
package stacktrace

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"sync"

	"github.com/go-extras/errx"
	"github.com/go-extras/errx/internal/errptr"
)

// DefaultMaxDepth is the default maximum number of stack frames captured by
// Here, Wrap, Classify, and ClassifyNew. Use HereDepth (or the *Depth variants)
// to capture deeper traces.
const DefaultMaxDepth = 32

// MaxDepth is the absolute ceiling on the number of stack frames captured by
// any of the *Depth variants (HereDepth, WrapDepth, ClassifyDepth,
// ClassifyNewDepth). Caller-supplied depths are clamped to this value to bound
// per-call allocation: each frame slot is one uintptr (8 bytes on 64-bit), so
// a stray HereDepth(1_000_000) would otherwise burn ~8MB on every call.
// 256 is generous in practice and matches the upper end of frame counts used
// by mature tracing libraries.
const MaxDepth = 256

// Frame represents a single stack frame with file, line, and function information.
type Frame struct {
	File     string // Full path to the source file
	Line     int    // Line number in the source file
	Function string // Fully qualified function name
}

// String returns a formatted representation of the frame.
func (f Frame) String() string {
	return fmt.Sprintf("%s:%d %s", f.File, f.Line, f.Function)
}

// Tracer is an interface implemented by errors that carry a captured stack
// trace. Extract and ExtractAll inspect the error chain for Tracer
// implementations so third-party errors with pre-captured traces (for example
// from a network boundary) can participate.
type Tracer interface {
	Frames() []Frame
}

// traced is an internal type that implements errx.Classified and captures stack trace.
type traced struct {
	pcs []uintptr // Program counters captured from the stack

	once         sync.Once
	cachedFrames []Frame
}

// Error returns a string representation of the traced error.
// This is primarily for debugging; the trace itself is accessed via Extract().
// It deliberately reports the frame count via len(t.pcs) rather than resolving
// symbols, so that simply printing an error does not force the (relatively
// expensive) runtime.CallersFrames walk.
func (t *traced) Error() string {
	if len(t.pcs) == 0 {
		return "(empty stack trace)"
	}
	return fmt.Sprintf("stack trace: %d frames", len(t.pcs))
}

// Format implements fmt.Formatter. The "%+v" verb renders the captured stack
// trace in the de-facto pkg/errors style (one frame per line, function then
// "\tfile:line"). The "%v" and "%s" verbs fall back to Error(). Unknown verbs
// produce a "%!<verb>(stacktrace.traced=...)" marker, mirroring stdlib
// behaviour for misuse.
func (t *traced) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			for _, f := range t.frames() {
				_, _ = fmt.Fprintf(s, "\n%s\n\t%s:%d", f.Function, f.File, f.Line)
			}
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, t.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", t.Error())
	default:
		_, _ = fmt.Fprintf(s, "%%!%c(stacktrace.traced=%s)", verb, t.Error())
	}
}

// Frames returns the resolved stack frames, satisfying the Tracer interface.
// The result is cached after the first call and is safe for concurrent use.
func (t *traced) Frames() []Frame {
	return t.frames()
}

// frames converts the stored program counters into Frame structs.
// Resolution happens lazily on first call and the result is cached so subsequent
// calls (e.g. from Error(), Extract(), and the json subpackage) reuse the work.
func (t *traced) frames() []Frame {
	if len(t.pcs) == 0 {
		return nil
	}

	t.once.Do(func() {
		frames := runtime.CallersFrames(t.pcs)
		result := make([]Frame, 0, len(t.pcs))
		for {
			frame, more := frames.Next()
			result = append(result, Frame{
				File:     frame.File,
				Line:     frame.Line,
				Function: frame.Function,
			})
			if !more {
				break
			}
		}
		t.cachedFrames = result
	})

	return t.cachedFrames
}

// IsClassified implements the errx.Classified interface marker method.
// It always returns true to identify this as a Classified error.
func (*traced) IsClassified() bool {
	return true
}

// Here captures the current stack trace and returns it as an errx.Classified.
// It can be used with errx.Wrap() or errx.Classify() to attach stack traces to errors.
//
// The stack trace is captured starting from the caller of Here(), skipping the
// Here() function itself and the runtime.Callers call. At most DefaultMaxDepth
// frames are captured; use HereDepth to override the limit.
//
// Example:
//
//	err := errx.Wrap("operation failed", cause, ErrNotFound, stacktrace.Here())
//
// The captured stack trace can later be extracted using Extract().
func Here() errx.Classified {
	return captureStack(2, DefaultMaxDepth) // Skip Here() and runtime.Callers
}

// HereDepth is like Here but allows the caller to specify the maximum number of
// stack frames to capture. Non-positive values fall back to DefaultMaxDepth.
// Values larger than MaxDepth are clamped to MaxDepth to bound allocation, so
// HereDepth(1_000_000) is safe and equivalent to HereDepth(MaxDepth).
//
// Example:
//
//	err := errx.Wrap("deep recursion", cause, stacktrace.HereDepth(64))
func HereDepth(depth int) errx.Classified {
	return captureStack(2, depth) // Skip HereDepth() and runtime.Callers
}

// captureStack captures the current stack trace with the specified skip count.
// skip indicates how many stack frames to skip (0 = captureStack itself).
// depth is the maximum number of frames to capture; non-positive values use
// DefaultMaxDepth, and values above MaxDepth are clamped to MaxDepth so that
// caller-supplied bounds cannot trigger unbounded allocation.
func captureStack(skip, depth int) *traced {
	if depth <= 0 {
		depth = DefaultMaxDepth
	}
	if depth > MaxDepth {
		depth = MaxDepth
	}
	pcs := make([]uintptr, depth)
	n := runtime.Callers(skip+1, pcs) // +1 to skip captureStack itself
	// Copy into an exact-sized slice so the (potentially large) backing array is
	// not retained for the lifetime of every traced error.
	out := make([]uintptr, n)
	copy(out, pcs[:n])
	return &traced{pcs: out}
}

// Extract returns stack frames from the first traced error found in the error chain.
// It traverses the entire error chain (including multi-error branches produced
// by errors.Join) looking for a traced error or a third-party Tracer
// implementation and returns its frames. The outermost trace wins, matching
// pkg/errors semantics.
//
// Returns nil if the error is nil or does not contain any stack trace.
//
// Example:
//
//	frames := stacktrace.Extract(err)
//	if frames != nil {
//	    for _, frame := range frames {
//	        fmt.Printf("%s:%d %s\n", frame.File, frame.Line, frame.Function)
//	    }
//	}
func Extract(err error) []Frame {
	// Delegate to ExtractAll so that multi-error branches (errors.Join) are
	// traversed identically. ExtractAll preserves outermost-first ordering, so
	// the first element is the trace Extract historically returned for
	// single-unwrap chains, plus we now also discover external Tracer
	// implementations hidden inside Join'd trees.
	all := ExtractAll(err)
	if len(all) == 0 {
		return nil
	}
	return all[0]
}

// ExtractAll returns every stack trace found in the error chain, ordered from
// the outermost wrap (closest to the caller) to the innermost. Both the
// internal traced type and external Tracer implementations are collected.
//
// Returns nil if the error is nil or contains no traces. The existing Extract
// behaviour (outermost only) is unchanged.
//
// Example:
//
//	for i, frames := range stacktrace.ExtractAll(err) {
//	    fmt.Printf("--- trace %d ---\n", i)
//	    for _, f := range frames {
//	        fmt.Println(f)
//	    }
//	}
func ExtractAll(err error) [][]Frame {
	if err == nil {
		return nil
	}

	w := &traceWalker{
		seenTraced: make(map[*traced]struct{}),
		// errptr.Get returns a hashable identity that works for both
		// pointer-based and value-based errors without panicking on
		// unhashable types.
		seenErr: make(map[uintptr]struct{}),
	}
	w.walk(err)

	if len(w.traces) == 0 {
		return nil
	}
	return w.traces
}

// traceWalker accumulates traces while walking an error chain, deduplicating
// repeated errors and *traced instances reachable through multiple paths.
type traceWalker struct {
	traces     [][]Frame
	seenTraced map[*traced]struct{}
	seenErr    map[uintptr]struct{}
}

// walk traverses the error chain rooted at err.
func (w *traceWalker) walk(err error) {
	for current := err; current != nil; current = errors.Unwrap(current) {
		if !w.markSeen(current) {
			return
		}
		w.collectFrom(current)
		w.walkClassifications(current)
		if w.walkMultiError(current) {
			return
		}
	}
}

// markSeen records an error in the seen set and returns false if it was
// already visited.
func (w *traceWalker) markSeen(err error) bool {
	id := errptr.Get(err)
	if id == 0 {
		return true
	}
	if _, already := w.seenErr[id]; already {
		return false
	}
	w.seenErr[id] = struct{}{}
	return true
}

// collectFrom records the frames of err if it is a *traced or Tracer.
func (w *traceWalker) collectFrom(err error) {
	switch v := err.(type) {
	case *traced:
		if _, dup := w.seenTraced[v]; dup {
			return
		}
		w.seenTraced[v] = struct{}{}
		if f := v.frames(); f != nil {
			w.traces = append(w.traces, f)
		}
	case Tracer:
		if f := v.Frames(); f != nil {
			w.traces = append(w.traces, f)
		}
	}
}

// walkClassifications recurses into carrier classifications.
func (w *traceWalker) walkClassifications(err error) {
	for _, cls := range extractCarrierClassifications(err) {
		if cls != nil {
			w.walk(cls)
		}
	}
}

// walkMultiError recurses into a multi-error's branches. It returns true when
// err implements the multi-unwrap interface (in which case the standard
// single-unwrap traversal must stop).
func (w *traceWalker) walkMultiError(err error) bool {
	mu, ok := err.(interface{ Unwrap() []error })
	if !ok {
		return false
	}
	for _, c := range mu.Unwrap() {
		if c != nil {
			w.walk(c)
		}
	}
	return true
}

// extractCarrierClassifications uses reflection to surface the classifications
// of an errx carrier (or any struct with a "classifications" field of type
// []errx.Classified). It mirrors the technique used by the json subpackage so
// that ExtractAll can reach traced entries attached as classifications.
func extractCarrierClassifications(err error) []errx.Classified {
	if err == nil {
		return nil
	}

	v := reflect.ValueOf(err)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}

	field := v.FieldByName("classifications")
	if !field.IsValid() || field.Kind() != reflect.Slice {
		return nil
	}

	result := make([]errx.Classified, 0, field.Len())
	for i := 0; i < field.Len(); i++ {
		itemVal := field.Index(i)
		if itemVal.CanAddr() {
			// UnsafePointer() returns the pointer value stored in the reflect.Value.
			ptr := itemVal.Addr().UnsafePointer()
			item := *(*errx.Classified)(ptr)
			result = append(result, item)
			continue
		}
		newVal := reflect.New(itemVal.Type()).Elem()
		newVal.Set(itemVal)
		if newVal.CanAddr() {
			ptr := newVal.Addr().UnsafePointer()
			item := *(*errx.Classified)(ptr)
			result = append(result, item)
		}
	}
	return result
}

// Wrap wraps an error with additional context text and optional classifications,
// automatically capturing a stack trace at the call site.
//
// This is a convenience function equivalent to:
//
//	errx.Wrap(text, cause, append(classifications, stacktrace.Here())...)
//
// If cause is nil, Wrap returns nil.
//
// Example:
//
//	err := stacktrace.Wrap("failed to process order", cause, ErrNotFound)
func Wrap(text string, cause error, classifications ...errx.Classified) error {
	if cause == nil {
		return nil
	}
	// Capture stack with skip=2 to skip Wrap() and runtime.Callers
	trace := captureStack(2, DefaultMaxDepth)
	// Defensive copy: appending to the caller's variadic slice can mutate the
	// caller's backing array when they spread a slice with spare capacity
	// (e.g. `cls := make([]errx.Classified, 0, 4); cls = append(cls, c1); stacktrace.Wrap("x", base, cls...)`).
	// Allocating a fresh slice avoids that aliasing entirely.
	ncls := make([]errx.Classified, len(classifications)+1)
	copy(ncls, classifications)
	ncls[len(classifications)] = trace
	return errx.Wrap(text, cause, ncls...)
}

// WrapDepth is like Wrap but allows the caller to specify the maximum number of
// stack frames to capture. Non-positive values fall back to DefaultMaxDepth,
// and values larger than MaxDepth are clamped to MaxDepth.
func WrapDepth(text string, cause error, depth int, classifications ...errx.Classified) error {
	if cause == nil {
		return nil
	}
	trace := captureStack(2, depth)
	// Defensive copy — see comment on Wrap for the aliasing scenario this avoids.
	ncls := make([]errx.Classified, len(classifications)+1)
	copy(ncls, classifications)
	ncls[len(classifications)] = trace
	return errx.Wrap(text, cause, ncls...)
}

// Classify attaches one or more classifications to an error, automatically
// capturing a stack trace at the call site.
//
// This is a convenience function equivalent to:
//
//	errx.Classify(cause, append(classifications, stacktrace.Here())...)
//
// If cause is nil, Classify returns nil.
//
// Example:
//
//	err := stacktrace.Classify(cause, ErrNotFound)
func Classify(cause error, classifications ...errx.Classified) error {
	if cause == nil {
		return nil
	}
	// Capture stack with skip=2 to skip Classify() and runtime.Callers
	trace := captureStack(2, DefaultMaxDepth)
	// Defensive copy — see comment on Wrap for the aliasing scenario this avoids.
	ncls := make([]errx.Classified, len(classifications)+1)
	copy(ncls, classifications)
	ncls[len(classifications)] = trace
	return errx.Classify(cause, ncls...)
}

// ClassifyDepth is like Classify but allows the caller to specify the maximum
// number of stack frames to capture. Non-positive values fall back to
// DefaultMaxDepth, and values larger than MaxDepth are clamped to MaxDepth.
func ClassifyDepth(cause error, depth int, classifications ...errx.Classified) error {
	if cause == nil {
		return nil
	}
	trace := captureStack(2, depth)
	// Defensive copy — see comment on Wrap for the aliasing scenario this avoids.
	ncls := make([]errx.Classified, len(classifications)+1)
	copy(ncls, classifications)
	ncls[len(classifications)] = trace
	return errx.Classify(cause, ncls...)
}

// ClassifyNew creates a new error with the given text and immediately classifies it
// with one or more classifications, automatically capturing a stack trace at the call site.
//
// This is a convenience function equivalent to:
//
//	errx.ClassifyNew(text, append(classifications, stacktrace.Here())...)
//
// This function is useful when you want to create a new error, classify it, and
// capture a stack trace in a single step, reducing verbosity.
//
// Example:
//
//	var ErrNotFound = errx.NewSentinel("not found")
//	var ErrDatabase = errx.NewSentinel("database error")
//
//	// Instead of:
//	// err := stacktrace.Classify(errors.New("user record missing"), ErrNotFound, ErrDatabase)
//
//	// You can write:
//	err := stacktrace.ClassifyNew("user record missing", ErrNotFound, ErrDatabase)
//
//	fmt.Println(err.Error())                        // Output: user record missing
//	fmt.Println(errors.Is(err, ErrNotFound))        // Output: true
//	fmt.Println(stacktrace.Extract(err) != nil)     // Output: true
func ClassifyNew(text string, classifications ...errx.Classified) error {
	// Capture stack with skip=2 to skip ClassifyNew() and runtime.Callers
	trace := captureStack(2, DefaultMaxDepth)
	// Defensive copy — see comment on Wrap for the aliasing scenario this avoids.
	ncls := make([]errx.Classified, len(classifications)+1)
	copy(ncls, classifications)
	ncls[len(classifications)] = trace
	return errx.ClassifyNew(text, ncls...)
}

// ClassifyNewDepth is like ClassifyNew but allows the caller to specify the
// maximum number of stack frames to capture. Non-positive values fall back to
// DefaultMaxDepth, and values larger than MaxDepth are clamped to MaxDepth.
func ClassifyNewDepth(text string, depth int, classifications ...errx.Classified) error {
	trace := captureStack(2, depth)
	// Defensive copy — see comment on Wrap for the aliasing scenario this avoids.
	ncls := make([]errx.Classified, len(classifications)+1)
	copy(ncls, classifications)
	ncls[len(classifications)] = trace
	return errx.ClassifyNew(text, ncls...)
}
