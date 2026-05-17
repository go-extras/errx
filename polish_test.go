package errx_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/go-extras/errx"
)

// TestToKVArgs_Empty verifies that ToKVArgs returns nil for an empty AttrList.
func TestToKVArgs_Empty(t *testing.T) {
	var al errx.AttrList
	if got := al.ToKVArgs(); got != nil {
		t.Errorf("ToKVArgs() on empty = %#v, want nil", got)
	}
}

// TestToKVArgs_RoundTrip verifies the flat key/value form for non-empty AttrLists.
func TestToKVArgs_RoundTrip(t *testing.T) {
	al := errx.AttrList{
		{Key: "user_id", Value: 123},
		{Key: "action", Value: "delete"},
		{Key: "ok", Value: true},
	}

	got := al.ToKVArgs()

	want := []any{"user_id", 123, "action", "delete", "ok", true}
	if len(got) != len(want) {
		t.Fatalf("len(ToKVArgs()) = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("ToKVArgs()[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

// TestToKVArgs_ExtractedFromError exercises the typical usage pattern: pull attrs
// off an error chain and pass them to a logger that wants flat key/value pairs.
func TestToKVArgs_ExtractedFromError(t *testing.T) {
	base := errors.New("base")
	attrErr := errx.Attrs("user_id", 42, "action", "list")
	err := errx.Wrap("op failed", base, attrErr)

	args := errx.ExtractAttrs(err).ToKVArgs()
	if len(args) != 4 {
		t.Fatalf("expected 4 flat args, got %d (%#v)", len(args), args)
	}
	if args[0] != "user_id" || args[1] != 42 || args[2] != "action" || args[3] != "list" {
		t.Errorf("unexpected flat args: %#v", args)
	}
}

// TestAttrListString_OutputUnchanged is a table test pinning the exact byte output of
// AttrList.String() across a variety of inputs. The refactor to strings.Builder must
// not change the output format.
func TestAttrListString_OutputUnchanged(t *testing.T) {
	cases := []struct {
		name string
		al   errx.AttrList
		want string
	}{
		{
			name: "empty",
			al:   errx.AttrList{},
			want: "",
		},
		{
			name: "single string",
			al:   errx.AttrList{{Key: "k", Value: "v"}},
			want: "k=v",
		},
		{
			name: "multi mixed",
			al: errx.AttrList{
				{Key: "user_id", Value: 123},
				{Key: "action", Value: "delete"},
				{Key: "ok", Value: true},
			},
			want: "user_id=123 action=delete ok=true",
		},
		{
			name: "value with spaces",
			al:   errx.AttrList{{Key: "msg", Value: "hello world"}},
			want: "msg=hello world",
		},
		{
			name: "struct value uses %+v",
			al:   errx.AttrList{{Key: "p", Value: struct{ X, Y int }{1, 2}}},
			want: "p={X:1 Y:2}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.al.String()
			if got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestClassify_NoClassifications_ReturnsSameError verifies the short-circuit:
// classify(cause) with no classifications MUST return cause unchanged so
// errors.Is(returned, cause) holds by identity.
func TestClassify_NoClassifications_ReturnsSameError(t *testing.T) {
	cause := errors.New("base")
	got := errx.Classify(cause)

	if got != cause { //nolint:errorlint // identity check is the contract under test
		t.Fatalf("Classify(cause) returned a different error pointer; want identity")
	}
	if !errors.Is(got, cause) {
		t.Errorf("errors.Is(returned, cause) = false, want true")
	}
}

// TestClassify_NoClassifications_NilCause keeps the existing nil behavior.
func TestClassify_NoClassifications_NilCause(t *testing.T) {
	if got := errx.Classify(nil); got != nil {
		t.Errorf("Classify(nil) = %v, want nil", got)
	}
}

// TestClassify_WithClassifications_StillWraps verifies that the short-circuit
// only applies when there are zero classifications.
func TestClassify_WithClassifications_StillWraps(t *testing.T) {
	cause := errors.New("base")
	cls := errx.NewSentinel("cls")

	got := errx.Classify(cause, cls)
	if got == cause { //nolint:errorlint // identity check is intentional
		t.Fatal("Classify(cause, cls) returned the same pointer; expected a wrapping carrier")
	}
	if !errors.Is(got, cls) {
		t.Errorf("returned error does not match the classification sentinel")
	}
	if !errors.Is(got, cause) {
		t.Errorf("returned error does not unwrap to cause")
	}
}

// TestDisplayTextOrEmpty_WithDisplayable returns the displayable message.
func TestDisplayTextOrEmpty_WithDisplayable(t *testing.T) {
	d := errx.NewDisplayable("Invalid email format")
	wrapped := errx.Wrap("validation failed", d)

	if got := errx.DisplayTextOrEmpty(wrapped); got != "Invalid email format" {
		t.Errorf("DisplayTextOrEmpty() = %q, want %q", got, "Invalid email format")
	}
}

// TestDisplayTextOrEmpty_NoDisplayable returns "" without leaking err.Error().
func TestDisplayTextOrEmpty_NoDisplayable(t *testing.T) {
	err := errors.New("internal db connection failed at 10.0.0.42")
	got := errx.DisplayTextOrEmpty(err)
	if got != "" {
		t.Errorf("DisplayTextOrEmpty() on non-displayable = %q, want \"\"", got)
	}
	// Defense-in-depth: make sure no part of the internal message leaked.
	if strings.Contains(got, "10.0.0.42") {
		t.Errorf("DisplayTextOrEmpty leaked internal text: %q", got)
	}
}

// TestDisplayTextOrEmpty_Nil returns "".
func TestDisplayTextOrEmpty_Nil(t *testing.T) {
	if got := errx.DisplayTextOrEmpty(nil); got != "" {
		t.Errorf("DisplayTextOrEmpty(nil) = %q, want \"\"", got)
	}
}
