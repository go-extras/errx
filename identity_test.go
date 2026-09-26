package errx_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/go-extras/errx"
	errxjson "github.com/go-extras/errx/json"
	"github.com/go-extras/errx/stacktrace"
)

// inlineCause is stored by value at the start of inlineWrapper, so pointers
// to the wrapper and its cause have the same address but different types.
type inlineCause struct{ cause error }

func (*inlineCause) Error() string   { return "boom" }
func (e *inlineCause) Unwrap() error { return e.cause }
func (*inlineCause) Frames() []stacktrace.Frame {
	return []stacktrace.Frame{{File: "db.go", Line: 42, Function: "db.Query"}}
}

type inlineWrapper struct{ in inlineCause }

func (*inlineWrapper) Error() string   { return "op: boom" }
func (e *inlineWrapper) Unwrap() error { return &e.in }

type zeroError struct{}

func (zeroError) Error() string { return "zero" }

type zeroTracedError struct{}

func (zeroTracedError) Error() string { return "traced" }
func (zeroTracedError) Frames() []stacktrace.Frame {
	return []stacktrace.Frame{{File: "b.go", Line: 1, Function: "b"}}
}

type zeroAttrsError struct{}

func (zeroAttrsError) Error() string { return "attrs" }
func (zeroAttrsError) Unwrap() error { return errx.Attrs("key", "value") }

type zeroWrapper struct{}

func (zeroWrapper) Error() string { return "op: traced" }
func (zeroWrapper) Unwrap() error { return zeroTracedError{} }

// TestIdentity_TraceTraversal covers both address collisions from issue #56.
func TestIdentity_TraceTraversal(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want []stacktrace.Frame
	}{
		{"first field", &inlineWrapper{}, (&inlineCause{}).Frames()},
		{"zero-size siblings", errors.Join(zeroError{}, zeroTracedError{}), zeroTracedError{}.Frames()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !stacktrace.HasTrace(tt.err) {
				t.Error("HasTrace = false, want true")
			}
			if got := stacktrace.Extract(tt.err); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Extract = %v, want %v", got, tt.want)
			}
			if got := stacktrace.ExtractAll(tt.err); !reflect.DeepEqual(got, [][]stacktrace.Frame{tt.want}) {
				t.Errorf("ExtractAll = %v, want one trace %v", got, tt.want)
			}
		})
	}
}

// TestIdentity_AttributeTraversal checks that collisions do not hide attributes.
func TestIdentity_AttributeTraversal(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"first field", &inlineWrapper{in: inlineCause{cause: errx.Attrs("key", "value")}}},
		{"zero-size siblings", errors.Join(zeroError{}, zeroAttrsError{})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errx.HasAttrs(tt.err) {
				t.Error("HasAttrs = false, want true")
			}
			want := errx.AttrList{{Key: "key", Value: "value"}}
			if got := errx.ExtractAttrs(tt.err); !reflect.DeepEqual(got, want) {
				t.Errorf("ExtractAttrs = %v, want %v", got, want)
			}
		})
	}
}

// TestIdentity_Marshal verifies colliding causes are serialized in full.
func TestIdentity_Marshal(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		message  string
		cause    string
		function string
	}{
		{"first field", &inlineWrapper{}, "op: boom", "boom", "db.Query"},
		{"zero-size chain", zeroWrapper{}, "op: traced", "traced", "b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := errxjson.Marshal(tt.err)
			if err != nil {
				t.Fatal(err)
			}
			var got errxjson.SerializedError
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if got.Message != tt.message || got.Cause == nil || got.Cause.Message != tt.cause {
				t.Fatalf("incorrect cause chain: %s", data)
			}
			if len(got.Cause.StackTrace) != 1 || got.Cause.StackTrace[0].Function != tt.function {
				t.Errorf("cause trace missing: %s", data)
			}
			if got.Cause.Cause != nil {
				t.Errorf("unexpected extra cause: %s", data)
			}
		})
	}
}

type nilAttrsA struct{}

func (*nilAttrsA) Error() string { return "a" }
func (*nilAttrsA) Unwrap() error { return errx.Attrs("a", 1) }

type nilAttrsB struct{}

func (*nilAttrsB) Error() string { return "b" }
func (*nilAttrsB) Unwrap() error { return errx.Attrs("b", 2) }

// TestIdentity_TypedNilBranches verifies that untrackable IDs are not recorded.
func TestIdentity_TypedNilBranches(t *testing.T) {
	err := errors.Join((*nilAttrsA)(nil), (*nilAttrsB)(nil))
	want := errx.AttrList{{Key: "a", Value: 1}, {Key: "b", Value: 2}}
	if got := errx.ExtractAttrs(err); !reflect.DeepEqual(got, want) {
		t.Errorf("ExtractAttrs = %v, want %v", got, want)
	}
}

type nilCycleError struct{}

var nilCycleUnwraps int

func (*nilCycleError) Error() string { return "nil cycle" }
func (e *nilCycleError) Unwrap() error {
	nilCycleUnwraps++
	if nilCycleUnwraps > 1 {
		// Bound the reproducer so a broken walker fails instead of hanging.
		panic("typed-nil cycle visited more than once")
	}
	return e
}

// TestIdentity_TypedNilCycle preserves ExtractAttrs' termination on nil cycles.
func TestIdentity_TypedNilCycle(t *testing.T) {
	nilCycleUnwraps = 0
	if got := errx.ExtractAttrs((*nilCycleError)(nil)); got != nil {
		t.Errorf("ExtractAttrs = %v, want nil", got)
	}
	if nilCycleUnwraps != 1 {
		t.Errorf("Unwrap calls = %d, want 1", nilCycleUnwraps)
	}
}

type identityCycle struct{ next error }

func (*identityCycle) Error() string   { return "cycle" }
func (e *identityCycle) Unwrap() error { return e.next }

type identityBranches []error

func (identityBranches) Error() string     { return "branches" }
func (e identityBranches) Unwrap() []error { return e }

// TestIdentity_CyclesAndSharedNodes verifies termination and deduplication.
func TestIdentity_CyclesAndSharedNodes(t *testing.T) {
	a, b := &identityCycle{}, &identityCycle{}
	a.next, b.next = b, a
	if errx.HasAttrs(a) || errx.ExtractAttrs(a) != nil || stacktrace.HasTrace(a) || stacktrace.ExtractAll(a) != nil {
		t.Fatal("cycle without metadata must not produce attributes or traces")
	}
	shared := errx.Classify(errors.New("shared"), errx.Attrs("key", "value"), stacktrace.Here())
	// A custom multi-error can contain nil children and need not be comparable.
	err := identityBranches{nil, a, shared, shared}
	if !errx.HasAttrs(err) || len(errx.ExtractAttrs(err)) != 1 {
		t.Error("shared attributes should be found once after the cyclic branch")
	}
	if !stacktrace.HasTrace(err) || len(stacktrace.ExtractAll(err)) != 1 {
		t.Error("shared trace should be found once after the cyclic branch")
	}
}
