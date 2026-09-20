package stacktrace_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/go-extras/errx"
	"github.com/go-extras/errx/stacktrace"
)

// hasFrameLines reports whether s contains pkg/errors-style stack frame lines,
// i.e. a function line followed by a tab-indented "file:line" entry.
func hasFrameLines(s string) bool {
	return strings.Contains(s, "\n\t")
}

// TestWrapFormatPlusV covers issue #45 case 1: stacktrace.Wrap renders the
// message plus the captured stack under "%+v". Before the fix the result was an
// fmt.wrapError whose "%+v" collapsed to the message only.
func TestWrapFormatPlusV(t *testing.T) {
	base := errors.New("base failure")
	err := stacktrace.Wrap("boom", base)

	out := fmt.Sprintf("%+v", err)

	if !strings.HasPrefix(out, "boom: base failure") {
		t.Errorf("expected %%+v to start with the message, got:\n%s", out)
	}
	if !strings.Contains(out, "TestWrapFormatPlusV") {
		t.Errorf("expected %%+v to name the capturing function, got:\n%s", out)
	}
	if !hasFrameLines(out) {
		t.Errorf("expected %%+v to contain pkg/errors-style frame lines, got:\n%s", out)
	}
}

// TestWrapFormatPlusVNested covers issue #45 case 2: a chain of stacktrace.Wrap
// calls renders the fully flattened message followed by the outermost trace.
func TestWrapFormatPlusVNested(t *testing.T) {
	base := errors.New("base failure")
	err := stacktrace.Wrap("outer", stacktrace.Wrap("inner", base))

	out := fmt.Sprintf("%+v", err)

	if !strings.HasPrefix(out, "outer: inner: base failure") {
		t.Errorf("expected the flattened message chain, got:\n%s", out)
	}
	if !hasFrameLines(out) || !strings.Contains(out, "TestWrapFormatPlusVNested") {
		t.Errorf("expected the outermost stack frames, got:\n%s", out)
	}
}

// TestPlainWrapFormatPlusV covers issue #45 case 3: a trace-less errx.Wrap keeps
// the historical message-only "%+v" output (no frame lines).
func TestPlainWrapFormatPlusV(t *testing.T) {
	base := errors.New("base failure")
	err := errx.Wrap("plain", base) // no classifications => no trace captured

	out := fmt.Sprintf("%+v", err)

	if out != "plain: base failure" {
		t.Errorf("%%+v = %q, want %q", out, "plain: base failure")
	}
	if hasFrameLines(out) {
		t.Errorf("did not expect frame lines for a trace-less wrap, got:\n%s", out)
	}
}

// TestClassifyFormatPlusV verifies stacktrace.Classify renders the unchanged
// message plus the captured stack under "%+v".
func TestClassifyFormatPlusV(t *testing.T) {
	base := errors.New("base failure")
	err := stacktrace.Classify(base)

	out := fmt.Sprintf("%+v", err)

	if !strings.HasPrefix(out, "base failure") {
		t.Errorf("expected the original message as prefix, got:\n%s", out)
	}
	if !hasFrameLines(out) || !strings.Contains(out, "TestClassifyFormatPlusV") {
		t.Errorf("expected captured stack frames, got:\n%s", out)
	}
}

// TestClassifyNewFormatPlusV verifies stacktrace.ClassifyNew renders the new
// message plus the captured stack under "%+v".
func TestClassifyNewFormatPlusV(t *testing.T) {
	err := stacktrace.ClassifyNew("made up")

	out := fmt.Sprintf("%+v", err)

	if !strings.HasPrefix(out, "made up") {
		t.Errorf("expected the new message as prefix, got:\n%s", out)
	}
	if !hasFrameLines(out) || !strings.Contains(out, "TestClassifyNewFormatPlusV") {
		t.Errorf("expected captured stack frames, got:\n%s", out)
	}
}

// TestWrapFormatPlainVerbs verifies the non-verbose verbs keep printing the
// message only, with no stack frames leaking in.
func TestWrapFormatPlainVerbs(t *testing.T) {
	base := errors.New("base failure")
	err := stacktrace.Wrap("boom", base)

	if v := fmt.Sprintf("%v", err); v != "boom: base failure" {
		t.Errorf("%%v = %q, want %q", v, "boom: base failure")
	}
	if s := fmt.Sprintf("%s", err); s != "boom: base failure" {
		t.Errorf("%%s = %q, want %q", s, "boom: base failure")
	}
	if q := fmt.Sprintf("%q", err); q != `"boom: base failure"` {
		t.Errorf("%%q = %q, want %q", q, `"boom: base failure"`)
	}
}

// countFrameLines returns the number of pkg/errors-style frame entries in s.
// traced.Format writes one "\n\t" per frame, so this also tells how many
// traces were rendered when compared with the frame count of the chain.
func countFrameLines(s string) int {
	return strings.Count(s, "\n\t")
}

// TestFormatPlusVThroughOuterLayers covers issue #54 with real captured traces:
// a trace taken deep in the chain has to survive every outer layer, and exactly
// one trace may be rendered.
func TestFormatPlusVThroughOuterLayers(t *testing.T) {
	tag := errx.NewSentinel("tag")
	base := errors.New("connection reset")
	inner := stacktrace.Wrap("query failed", base)
	want := len(stacktrace.Extract(inner))
	if want == 0 {
		t.Fatal("expected the inner error to carry a trace")
	}

	tests := []struct {
		name    string
		err     error
		message string
	}{
		{"wrap with a classification", errx.Wrap("load user", inner, tag), "load user: query failed: connection reset"},
		{"wrap without classifications", errx.Wrap("load user", inner), "load user: query failed: connection reset"},
		{"classify", errx.Classify(inner, tag), "query failed: connection reset"},
		{"join", errx.Join(inner, errors.New("other")), "query failed: connection reset\nother"},
		{"wrapIf over a traced cause", stacktrace.WrapIf("again", inner), "again: query failed: connection reset"},
		{"nested wraps", errx.Wrap("a", errx.Wrap("b", inner)), "a: b: query failed: connection reset"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := fmt.Sprintf("%+v", tc.err)

			if !strings.HasPrefix(out, tc.message) {
				t.Errorf("expected the message %q first, got:\n%s", tc.message, out)
			}
			if !strings.Contains(out, "TestFormatPlusVThroughOuterLayers") {
				t.Errorf("expected the capturing function in the frames, got:\n%s", out)
			}
			if got := countFrameLines(out); got != want {
				t.Errorf("rendered %d frame lines, want %d (exactly one trace)", got, want)
			}
			if strings.Count(out, tc.message) != 1 {
				t.Errorf("expected the message exactly once, got:\n%s", out)
			}
		})
	}
}

// TestFormatPlusVRendersOutermostTraceOnly pins which trace wins when several
// layers captured one: the outermost, matching stacktrace.Extract.
func TestFormatPlusVRendersOutermostTraceOnly(t *testing.T) {
	inner := stacktrace.Wrap("inner", errors.New("base failure"))
	outer := errx.Wrap("outer", stacktrace.Classify(inner, errx.NewSentinel("tag")))

	out := fmt.Sprintf("%+v", outer)

	if got, want := countFrameLines(out), len(stacktrace.Extract(outer)); got != want {
		t.Errorf("rendered %d frame lines, want %d", got, want)
	}
	if strings.Count(out, "outer: inner: base failure") != 1 {
		t.Errorf("expected the message exactly once, got:\n%s", out)
	}
}
