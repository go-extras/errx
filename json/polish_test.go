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
