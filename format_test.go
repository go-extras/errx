package errx_test

import (
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/go-extras/errx"
)

// fmtClassification is an external errx.Classified that is also a fmt.Formatter.
// It models the stacktrace subpackage's *traced value — a classification that
// renders extra detail under "%+v" — without importing stacktrace, so the tests
// for the core package's own Format wiring stay self-contained. Under "%+v" it
// appends a recognizable marker; for every other verb it writes nothing, which
// lets the tests assert that carrier/wrapped delegate to classifications only
// under the "+" flag.
type fmtClassification struct{ marker string }

func (fmtClassification) Error() string      { return "fmt-classification" }
func (fmtClassification) IsClassified() bool { return true }

func (f fmtClassification) Format(s fmt.State, verb rune) {
	if verb == 'v' && s.Flag('+') {
		_, _ = io.WriteString(s, f.marker)
	}
}

// plainClassification is an external errx.Classified that is NOT a fmt.Formatter,
// used to confirm that non-Formatter classifications contribute nothing to "%+v".
type plainClassification struct{}

func (plainClassification) Error() string      { return "plain-classification" }
func (plainClassification) IsClassified() bool { return true }

// TestClassifyFormat exercises the fmt.Formatter implementation on the carrier
// produced by Classify (and, implicitly, ClassifyNew), across every verb.
func TestClassifyFormat(t *testing.T) {
	base := errors.New("base failure")
	withFmt := errx.Classify(base, fmtClassification{marker: "\n>>frames<<"})
	withPlain := errx.Classify(base, plainClassification{})

	tests := []struct {
		name   string
		format string
		err    error
		want   string
	}{
		{"plus-v delegates to Formatter classification", "%+v", withFmt, "base failure\n>>frames<<"},
		{"plus-v skips non-Formatter classification", "%+v", withPlain, "base failure"},
		{"v prints message only", "%v", withFmt, "base failure"},
		{"s prints message only", "%s", withFmt, "base failure"},
		{"q prints quoted message", "%q", withFmt, `"base failure"`},
		{"unknown verb prints marker", "%d", withFmt, "%!d(errx.carrier=base failure)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := fmt.Sprintf(tc.format, tc.err); got != tc.want {
				t.Errorf("Sprintf(%q) = %q, want %q", tc.format, got, tc.want)
			}
		})
	}
}

// TestWrapFormat exercises the fmt.Formatter implementation on the wrapper
// produced by Wrap when classifications are attached. The wrap context text must
// prefix the message, and the Formatter classification must still surface under
// "%+v".
func TestWrapFormat(t *testing.T) {
	base := errors.New("base failure")
	withFmt := errx.Wrap("ctx", base, fmtClassification{marker: "\n>>frames<<"})
	withPlain := errx.Wrap("ctx", base, plainClassification{})

	tests := []struct {
		name   string
		format string
		err    error
		want   string
	}{
		{"plus-v delegates to Formatter classification", "%+v", withFmt, "ctx: base failure\n>>frames<<"},
		{"plus-v skips non-Formatter classification", "%+v", withPlain, "ctx: base failure"},
		{"v prints message only", "%v", withFmt, "ctx: base failure"},
		{"s prints message only", "%s", withFmt, "ctx: base failure"},
		{"q prints quoted message", "%q", withFmt, `"ctx: base failure"`},
		{"unknown verb prints marker", "%d", withFmt, "%!d(errx.wrapped=ctx: base failure)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := fmt.Sprintf(tc.format, tc.err); got != tc.want {
				t.Errorf("Sprintf(%q) = %q, want %q", tc.format, got, tc.want)
			}
		})
	}
}

// TestFormatMultipleFormatterClassifications pins the ordering when more than one
// classification is a fmt.Formatter (e.g. two captured traces): formatClassifications
// appends each in slice order after the message, so a refactor that reorders or
// dedupes them would fail here.
func TestFormatMultipleFormatterClassifications(t *testing.T) {
	base := errors.New("base failure")
	a := fmtClassification{marker: "\n>>A<<"}
	b := fmtClassification{marker: "\n>>B<<"}

	if got := fmt.Sprintf("%+v", errx.Classify(base, a, b)); got != "base failure\n>>A<<\n>>B<<" {
		t.Errorf("carrier %%+v = %q, want %q", got, "base failure\n>>A<<\n>>B<<")
	}
	if got := fmt.Sprintf("%+v", errx.Wrap("ctx", base, a, b)); got != "ctx: base failure\n>>A<<\n>>B<<" {
		t.Errorf("wrapped %%+v = %q, want %q", got, "ctx: base failure\n>>A<<\n>>B<<")
	}
}

// TestWrapNoClassificationsFormat confirms that a trace-less, classification-less
// Wrap (which avoids the carrier entirely and behaves like fmt.Errorf) still
// renders the message only under "%+v".
func TestWrapNoClassificationsFormat(t *testing.T) {
	base := errors.New("base failure")
	err := errx.Wrap("ctx", base)

	if got := fmt.Sprintf("%+v", err); got != "ctx: base failure" {
		t.Errorf("%%+v = %q, want %q", got, "ctx: base failure")
	}
}

// TestFormatPreservesChainSemantics guards the two-layer (wrapped -> carrier ->
// cause) shape that the Formatter change must not disturb: a single Unwrap of a
// Wrap result still yields a Classified carrier, and errors.Is reaches both the
// cause and the attached classification.
func TestFormatPreservesChainSemantics(t *testing.T) {
	tag := errx.NewSentinel("tag")
	base := errors.New("base failure")
	err := errx.Wrap("ctx", base, tag)

	if !errors.Is(err, base) {
		t.Error("expected errors.Is to match the cause")
	}
	if !errors.Is(err, tag) {
		t.Error("expected errors.Is to match the classification")
	}

	inner := errors.Unwrap(err)
	if _, ok := errx.CarrierClassifications(inner); !ok {
		t.Error("expected a single Unwrap to expose the classification carrier")
	}
}

// cyclicError builds an unwrap chain that loops back on itself, which the
// standard library's own Unwrap-based helpers walk forever. Formatting must
// stop regardless.
type cyclicError struct {
	msg  string
	next error
}

func (c *cyclicError) Error() string { return c.msg }
func (c *cyclicError) Unwrap() error { return c.next }

// TestFormatReachesFormatterBelowOuterLayers covers issue #54: a classification
// that renders under "%+v" (a captured stack trace in practice) must still
// surface when the error is wrapped again by a layer that carries none of its
// own. Before the fix each layer looked only at its own classifications, so the
// marker disappeared as soon as anything wrapped it.
func TestFormatReachesFormatterBelowOuterLayers(t *testing.T) {
	base := errors.New("base failure")
	marker := "\n>>frames<<"
	inner := errx.Classify(base, fmtClassification{marker: marker})
	tag := errx.NewSentinel("tag")

	tests := []struct {
		name string
		err  error
		want string
	}{
		{"wrap with a classification", errx.Wrap("ctx", inner, tag), "ctx: base failure" + marker},
		{"wrap without classifications", errx.Wrap("ctx", inner), "ctx: base failure" + marker},
		{"classify", errx.Classify(inner, tag), "base failure" + marker},
		{"join", errx.Join(inner, errors.New("other")), "base failure\nother" + marker},
		{"nested wraps", errx.Wrap("a", errx.Wrap("b", errx.Wrap("c", inner))), "a: b: c: base failure" + marker},
		{"fmt.Errorf in the middle", errx.Wrap("outer", fmt.Errorf("mid: %w", inner)), "outer: mid: base failure" + marker},
		{"attributes only at the outer level", errx.Wrap("ctx", inner, errx.Attrs("k", "v")), "ctx: base failure" + marker},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := fmt.Sprintf("%+v", tc.err); got != tc.want {
				t.Errorf("%%+v = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestFormatStopsAtOutermostLevelWithFormatter pins the search order: the walk
// stops at the first level that has something to render, so a chain wrapped
// with a trace at several layers prints one trace, not one per layer.
func TestFormatStopsAtOutermostLevelWithFormatter(t *testing.T) {
	base := errors.New("base failure")
	inner := errx.Classify(base, fmtClassification{marker: "\n>>inner<<"})
	outer := errx.Wrap("ctx", inner, fmtClassification{marker: "\n>>outer<<"})

	got := fmt.Sprintf("%+v", outer)
	if got != "ctx: base failure\n>>outer<<" {
		t.Errorf("%%+v = %q, want only the outermost level rendered", got)
	}
}

// TestFormatWithoutAnyFormatterClassification confirms the trailer stays empty
// when nothing in the chain renders, so a trace-less error keeps its historical
// message-only output.
func TestFormatWithoutAnyFormatterClassification(t *testing.T) {
	base := errors.New("base failure")
	err := errx.Wrap("ctx", errx.Classify(base, plainClassification{}), errx.NewSentinel("tag"))

	if got := fmt.Sprintf("%+v", err); got != "ctx: base failure" {
		t.Errorf("%%+v = %q, want %q", got, "ctx: base failure")
	}
}

// TestFormatTerminatesOnCyclicChain guards the depth bound in the "%+v" walk.
// A cycle must not hang the formatter, however the error got built.
func TestFormatTerminatesOnCyclicChain(t *testing.T) {
	a := &cyclicError{msg: "a"}
	b := &cyclicError{msg: "b", next: a}
	a.next = b

	done := make(chan string, 1)
	go func() {
		done <- fmt.Sprintf("%+v", errx.Wrap("ctx", a, plainClassification{}))
	}()

	select {
	case got := <-done:
		if got != "ctx: a" {
			t.Errorf("%%+v = %q, want %q", got, "ctx: a")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("formatting a cyclic chain did not terminate")
	}
}

// TestWrapWithoutClassificationsSemantics pins the parts of a classification-less
// Wrap that must stay identical to the fmt.Errorf("%s: %w", …) result it
// replaced: the message, a single Unwrap yielding the cause, and errors.Is.
func TestWrapWithoutClassificationsSemantics(t *testing.T) {
	base := errors.New("base failure")
	err := errx.Wrap("ctx", base)

	if err.Error() != "ctx: base failure" {
		t.Errorf("Error() = %q, want %q", err.Error(), "ctx: base failure")
	}
	if unwrapped := errors.Unwrap(err); unwrapped != base {
		t.Errorf("Unwrap() = %v, want the cause itself", unwrapped)
	}
	if !errors.Is(err, base) {
		t.Error("expected errors.Is to match the cause")
	}
	if _, ok := errx.CarrierClassifications(errors.Unwrap(err)); ok {
		t.Error("expected no carrier below a classification-less wrap")
	}
}

// TestJoinFormat exercises the fmt.Formatter implementation on the aggregate
// returned by Join across every verb.
func TestJoinFormat(t *testing.T) {
	marker := "\n>>frames<<"
	first := errors.New("first")
	second := errx.Classify(errors.New("second"), fmtClassification{marker: marker})
	err := errx.Join(first, second)

	tests := []struct {
		name   string
		format string
		want   string
	}{
		{"plus-v renders the branch classification", "%+v", "first\nsecond" + marker},
		{"v prints messages only", "%v", "first\nsecond"},
		{"s prints messages only", "%s", "first\nsecond"},
		{"q prints quoted messages", "%q", `"first\nsecond"`},
		{"unknown verb prints marker", "%d", "%!d(errx.joinError=first\nsecond)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := fmt.Sprintf(tc.format, err); got != tc.want {
				t.Errorf("Sprintf(%q) = %q, want %q", tc.format, got, tc.want)
			}
		})
	}
}
