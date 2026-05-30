package json_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/go-extras/errx"
	errxjson "github.com/go-extras/errx/json"
	"github.com/go-extras/errx/stacktrace"
)

// TestWithStackTrace_False suppresses stack-trace serialization entirely.
func TestWithStackTrace_False(t *testing.T) {
	testErr := stacktrace.Wrap("op", errors.New("base"))

	data, err := errxjson.Marshal(testErr, errxjson.WithStackTrace(false))
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(result.StackTrace) != 0 {
		t.Errorf("StackTrace = %v, want empty (suppressed by WithStackTrace(false))", result.StackTrace)
	}

	// Also assert the field is omitted from the raw JSON (omitempty).
	if strings.Contains(string(data), `"stack_trace"`) {
		t.Errorf("expected stack_trace field to be omitted from JSON; got: %s", string(data))
	}
}

// TestWithStackTrace_TrueIsDefault keeps the default behavior intact.
func TestWithStackTrace_TrueIsDefault(t *testing.T) {
	testErr := stacktrace.Wrap("op", errors.New("base"))

	data, err := errxjson.Marshal(testErr) // no option => default true
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(result.StackTrace) == 0 {
		t.Errorf("expected stack trace by default; got none")
	}
}

// TestWithAttributes_False suppresses attribute serialization.
func TestWithAttributes_False(t *testing.T) {
	attrErr := errx.Attrs("user_id", 42)
	testErr := errx.Classify(errors.New("base"), attrErr)

	data, err := errxjson.Marshal(testErr, errxjson.WithAttributes(false))
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	if strings.Contains(string(data), `"attributes"`) {
		t.Errorf("expected attributes to be omitted; got: %s", string(data))
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if len(result.Attributes) != 0 {
		t.Errorf("Attributes = %v, want empty", result.Attributes)
	}
}

// TestWithSentinels_False suppresses sentinel serialization.
func TestWithSentinels_False(t *testing.T) {
	sentinel := errx.NewSentinel("sentinel-text")
	testErr := errx.Classify(errors.New("base"), sentinel)

	data, err := errxjson.Marshal(testErr, errxjson.WithSentinels(false))
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	if strings.Contains(string(data), `"sentinels"`) {
		t.Errorf("expected sentinels to be omitted; got: %s", string(data))
	}
}

// TestWithAttributeValueTransformer rewrites attribute values during serialization.
func TestWithAttributeValueTransformer(t *testing.T) {
	attrErr := errx.Attrs("user_id", 42, "password", "hunter2", "action", "login")
	testErr := errx.Classify(errors.New("base"), attrErr)

	redact := func(key string, v any) any {
		if key == "password" {
			return "<redacted>"
		}
		return v
	}

	data, err := errxjson.Marshal(testErr, errxjson.WithAttributeValueTransformer(redact))
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	gotByKey := make(map[string]any)
	for _, a := range result.Attributes {
		gotByKey[a.Key] = a.Value
	}

	if gotByKey["password"] != "<redacted>" {
		t.Errorf("password value = %#v, want \"<redacted>\"", gotByKey["password"])
	}
	if gotByKey["user_id"] != float64(42) { // JSON numbers come back as float64
		t.Errorf("user_id value = %#v, want 42", gotByKey["user_id"])
	}
	if gotByKey["action"] != "login" {
		t.Errorf("action value = %#v, want \"login\"", gotByKey["action"])
	}
}

// TestWithAttributeValueTransformer_NotCalledWhenAttributesDisabled ensures the
// transformer is only invoked when attributes are actually serialized.
func TestWithAttributeValueTransformer_NotCalledWhenAttributesDisabled(t *testing.T) {
	attrErr := errx.Attrs("k", "v")
	testErr := errx.Classify(errors.New("base"), attrErr)

	called := false
	xfm := func(_ string, v any) any { called = true; return v }

	_, err := errxjson.Marshal(testErr,
		errxjson.WithAttributes(false),
		errxjson.WithAttributeValueTransformer(xfm),
	)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if called {
		t.Error("attribute value transformer was called even though WithAttributes(false) was set")
	}
}

// TestWithStackTraceTrimPaths trims a configured prefix from each frame.
func TestWithStackTraceTrimPaths(t *testing.T) {
	testErr := stacktrace.Wrap("op", errors.New("base"))

	// First, capture the raw frames so we know a prefix that should match.
	frames := stacktrace.Extract(testErr)
	if len(frames) == 0 {
		t.Fatal("expected stack frames")
	}
	// Find a common prefix across frames (most should share at least one directory).
	prefix := frames[0].File
	// Trim back to the last "/" so we have a real prefix.
	if idx := strings.LastIndex(prefix, "/"); idx > 0 {
		prefix = prefix[:idx+1]
	}

	data, err := errxjson.Marshal(testErr, errxjson.WithStackTraceTrimPaths(prefix))
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(result.StackTrace) == 0 {
		t.Fatal("expected non-empty stack trace")
	}
	for _, f := range result.StackTrace {
		if strings.HasPrefix(f.File, prefix) {
			t.Errorf("frame.File %q still has prefix %q after trim", f.File, prefix)
		}
	}
}

// TestWithStackTraceTrimPaths_EmptyIsNoOp confirms empty prefix preserves paths.
func TestWithStackTraceTrimPaths_EmptyIsNoOp(t *testing.T) {
	testErr := stacktrace.Wrap("op", errors.New("base"))

	withDefault, err := errxjson.Marshal(testErr)
	if err != nil {
		t.Fatalf("Marshal default: %v", err)
	}
	withEmpty, err := errxjson.Marshal(testErr, errxjson.WithStackTraceTrimPaths(""))
	if err != nil {
		t.Fatalf("Marshal empty: %v", err)
	}

	if string(withDefault) != string(withEmpty) {
		t.Errorf("empty trim prefix changed output:\n  default: %s\n  empty:   %s", withDefault, withEmpty)
	}
}

// TestWithMaxMessageBytes_Truncates verifies messages longer than the limit are
// truncated and suffixed.
func TestWithMaxMessageBytes_Truncates(t *testing.T) {
	long := strings.Repeat("a", 500)
	testErr := errors.New(long)

	data, err := errxjson.Marshal(testErr, errxjson.WithMaxMessageBytes(64))
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(result.Message) > 64 {
		t.Errorf("len(Message) = %d, want <= 64", len(result.Message))
	}
	if !strings.HasSuffix(result.Message, "...(truncated)") {
		t.Errorf("expected truncation suffix; got: %q", result.Message)
	}
}

// TestWithMaxMessageBytes_NoTruncationWhenShort verifies short messages are unchanged.
func TestWithMaxMessageBytes_NoTruncationWhenShort(t *testing.T) {
	testErr := errors.New("short")
	data, err := errxjson.Marshal(testErr, errxjson.WithMaxMessageBytes(64))
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if result.Message != "short" {
		t.Errorf("Message = %q, want %q (no truncation expected)", result.Message, "short")
	}
}

// TestWithMaxMessageBytes_ZeroIsNoOp verifies the default (0) disables truncation.
func TestWithMaxMessageBytes_ZeroIsNoOp(t *testing.T) {
	long := strings.Repeat("a", 500)
	testErr := errors.New(long)

	data, err := errxjson.Marshal(testErr, errxjson.WithMaxMessageBytes(0))
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if result.Message != long {
		t.Errorf("expected unchanged message at zero limit; got %d bytes", len(result.Message))
	}
}

// TestWithMaxMessageBytes_UTF8Safe verifies that multi-byte runes are not split.
func TestWithMaxMessageBytes_UTF8Safe(t *testing.T) {
	// "é" is 2 bytes in UTF-8; repeat to make a string longer than the cap.
	long := strings.Repeat("é", 200) // 400 bytes
	testErr := errors.New(long)

	data, err := errxjson.Marshal(testErr, errxjson.WithMaxMessageBytes(64))
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// The truncated portion (before the suffix) must be valid UTF-8.
	if !strings.HasSuffix(result.Message, "...(truncated)") {
		t.Fatalf("expected suffix; got: %q", result.Message)
	}
	prefix := strings.TrimSuffix(result.Message, "...(truncated)")
	for _, r := range prefix {
		if r == 0xFFFD { // U+FFFD REPLACEMENT CHARACTER signals broken UTF-8
			t.Errorf("truncation produced an invalid UTF-8 sequence: %q", prefix)
			break
		}
	}
}

// The trim* helpers build a deterministic, deeply-nested stack with uniquely
// named frames (innermost first: trimChainC, trimChainB, trimChainA, then the
// test function and runtime). //go:noinline keeps each frame distinct so the
// trim/filter tests below can reason about frame order precisely.
//
//go:noinline
func trimChainC() error { return stacktrace.Wrap("op", errors.New("base")) }

//go:noinline
func trimChainB() error { return trimChainC() }

//go:noinline
func trimChainA() error { return trimChainB() }

// unmarshalFrames marshals err with opts and decodes the resulting stack trace.
func unmarshalFrames(t *testing.T, err error, opts ...errxjson.Option) ([]byte, errxjson.SerializedError) {
	t.Helper()
	data, mErr := errxjson.Marshal(err, opts...)
	if mErr != nil {
		t.Fatalf("Marshal error: %v", mErr)
	}
	var result errxjson.SerializedError
	if uErr := json.Unmarshal(data, &result); uErr != nil {
		t.Fatalf("Unmarshal error: %v", uErr)
	}
	return data, result
}

// TestWithStackTraceTrimTop drops the top (innermost) n frames.
func TestWithStackTraceTrimTop(t *testing.T) {
	testErr := trimChainA()
	frames := stacktrace.Extract(testErr)
	if len(frames) < 3 {
		t.Fatalf("expected at least 3 frames, got %d", len(frames))
	}

	_, result := unmarshalFrames(t, testErr, errxjson.WithStackTraceTrimTop(2))

	if got, want := len(result.StackTrace), len(frames)-2; got != want {
		t.Fatalf("len(StackTrace) = %d, want %d", got, want)
	}
	// The first surviving frame is the original frames[2].
	if result.StackTrace[0].Function != frames[2].Function || result.StackTrace[0].Line != frames[2].Line {
		t.Errorf("first frame = %s:%d, want %s:%d",
			result.StackTrace[0].Function, result.StackTrace[0].Line, frames[2].Function, frames[2].Line)
	}
	// The two trimmed functions must be gone.
	for _, f := range result.StackTrace {
		if f.Function == frames[0].Function || f.Function == frames[1].Function {
			t.Errorf("frame %q survived trimming of the top 2 frames", f.Function)
		}
	}
}

// TestWithStackTraceTrimTop_ZeroOrNegativeIsNoOp confirms 0/negative keep all frames.
func TestWithStackTraceTrimTop_ZeroOrNegativeIsNoOp(t *testing.T) {
	testErr := trimChainA()

	withDefault, mErr := errxjson.Marshal(testErr)
	if mErr != nil {
		t.Fatalf("Marshal default: %v", mErr)
	}
	for _, n := range []int{0, -5} {
		withTrim, mErr := errxjson.Marshal(testErr, errxjson.WithStackTraceTrimTop(n))
		if mErr != nil {
			t.Fatalf("Marshal trim %d: %v", n, mErr)
		}
		if string(withDefault) != string(withTrim) {
			t.Errorf("WithStackTraceTrimTop(%d) changed output:\n  default: %s\n  trim:    %s", n, withDefault, withTrim)
		}
	}
}

// TestWithStackTraceTrimTop_AllTrimmedOmitsField drops the field when n >= len.
func TestWithStackTraceTrimTop_AllTrimmedOmitsField(t *testing.T) {
	testErr := trimChainA()
	frames := stacktrace.Extract(testErr)

	for _, n := range []int{len(frames), len(frames) + 10} {
		data, result := unmarshalFrames(t, testErr, errxjson.WithStackTraceTrimTop(n))
		if len(result.StackTrace) != 0 {
			t.Errorf("n=%d: expected empty stack trace, got %d frames", n, len(result.StackTrace))
		}
		if strings.Contains(string(data), "stack_trace") {
			t.Errorf("n=%d: expected stack_trace field to be omitted, got: %s", n, data)
		}
	}
}

// TestWithStackFrameFilter_DropsRejected removes frames the predicate rejects.
func TestWithStackFrameFilter_DropsRejected(t *testing.T) {
	testErr := trimChainA()
	frames := stacktrace.Extract(testErr)

	dropped := 0
	for _, f := range frames {
		if strings.Contains(f.Function, "trimChainB") {
			dropped++
		}
	}
	if dropped == 0 {
		t.Fatal("expected at least one trimChainB frame to drop")
	}

	keep := func(f stacktrace.Frame) bool { return !strings.Contains(f.Function, "trimChainB") }
	_, result := unmarshalFrames(t, testErr, errxjson.WithStackFrameFilter(keep))

	if got, want := len(result.StackTrace), len(frames)-dropped; got != want {
		t.Fatalf("len(StackTrace) = %d, want %d", got, want)
	}
	var sawA, sawC bool
	for _, f := range result.StackTrace {
		if strings.Contains(f.Function, "trimChainB") {
			t.Errorf("filtered frame %q survived", f.Function)
		}
		sawA = sawA || strings.Contains(f.Function, "trimChainA")
		sawC = sawC || strings.Contains(f.Function, "trimChainC")
	}
	if !sawA || !sawC {
		t.Errorf("expected sibling frames to remain (trimChainA=%v, trimChainC=%v)", sawA, sawC)
	}
}

// TestWithStackFrameFilter_NilIsNoOp confirms a nil filter keeps all frames.
func TestWithStackFrameFilter_NilIsNoOp(t *testing.T) {
	testErr := trimChainA()

	withDefault, mErr := errxjson.Marshal(testErr)
	if mErr != nil {
		t.Fatalf("Marshal default: %v", mErr)
	}
	withNil, mErr := errxjson.Marshal(testErr, errxjson.WithStackFrameFilter(nil))
	if mErr != nil {
		t.Fatalf("Marshal nil filter: %v", mErr)
	}
	if string(withDefault) != string(withNil) {
		t.Errorf("nil filter changed output:\n  default: %s\n  nil:     %s", withDefault, withNil)
	}
}

// TestStackFrameTrim_Ordering verifies the apply order: trim top, then filter,
// then cap — so the cap counts only post-filter survivors. With TrimTop(1)
// removing trimChainC and the filter removing trimChainB, a cap of 2 yields
// trimChainA followed by the test function.
func TestStackFrameTrim_Ordering(t *testing.T) {
	testErr := trimChainA()
	frames := stacktrace.Extract(testErr)
	if len(frames) < 4 {
		t.Fatalf("expected at least 4 frames, got %d", len(frames))
	}

	keep := func(f stacktrace.Frame) bool { return !strings.Contains(f.Function, "trimChainB") }
	_, result := unmarshalFrames(t, testErr,
		errxjson.WithStackTraceTrimTop(1),
		errxjson.WithStackFrameFilter(keep),
		errxjson.WithMaxStackFrames(2),
	)

	if len(result.StackTrace) != 2 {
		t.Fatalf("len(StackTrace) = %d, want 2 (cap counts post-filter survivors)", len(result.StackTrace))
	}
	// frames[0]=trimChainC trimmed, frames[1]=trimChainB filtered → survivors
	// start at frames[2] (trimChainA), then frames[3] (the test function).
	if result.StackTrace[0].Function != frames[2].Function {
		t.Errorf("first survivor = %q, want %q", result.StackTrace[0].Function, frames[2].Function)
	}
	if result.StackTrace[1].Function != frames[3].Function {
		t.Errorf("second survivor = %q, want %q", result.StackTrace[1].Function, frames[3].Function)
	}
	for _, f := range result.StackTrace {
		if strings.Contains(f.Function, "trimChainC") || strings.Contains(f.Function, "trimChainB") {
			t.Errorf("frame %q should have been removed", f.Function)
		}
	}
}
