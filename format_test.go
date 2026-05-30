package errx_test

import (
	"errors"
	"fmt"
	"io"
	"testing"

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
