package json_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-extras/errx"
	errxjson "github.com/go-extras/errx/json"
)

// Bug #1: errx.Classify duplicates the cause message at every nesting level.
// The carrier's Error() delegates to its cause, so when the top-level node IS a
// carrier (which Classify always returns), the cause was serialized as a child
// even though it carries no new info. After the fix, the carrier is peeled at
// recursion entry: its classifications/displayable/attrs are pulled into the
// current level, then we advance past the carrier so the duplicated cause is
// not emitted.
func TestMarshal_ClassifyTopLevelNoDuplicateCause(t *testing.T) {
	sentinel := errx.NewSentinel("test-sentinel")
	err := errx.Classify(errors.New("inner message"), sentinel)

	data, marshalErr := errxjson.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Marshal error: %v", marshalErr)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if result.Message != "inner message" {
		t.Errorf("Message = %q, want %q", result.Message, "inner message")
	}
	if len(result.Sentinels) != 1 || result.Sentinels[0] != "test-sentinel" {
		t.Errorf("Sentinels = %v, want [\"test-sentinel\"]", result.Sentinels)
	}
	// The cause should be omitted because the inner error carries no additional info.
	if result.Cause != nil {
		t.Errorf("Cause = %+v, want nil (carrier should be peeled, no duplicate)", result.Cause)
	}
}

// Nested Classify should not multiply redundancy.
func TestMarshal_NestedClassifyNoDuplicateCause(t *testing.T) {
	sa := errx.NewSentinel("sentA")
	sb := errx.NewSentinel("sentB")
	err := errx.Classify(errx.Classify(errors.New("inner"), sa), sb)

	data, marshalErr := errxjson.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Marshal error: %v", marshalErr)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if result.Message != "inner" {
		t.Errorf("Message = %q, want %q", result.Message, "inner")
	}
	// Both sentinels should appear at the single top level; sentB came from the
	// outer carrier, sentA from the peeled inner carrier.
	if len(result.Sentinels) != 2 {
		t.Fatalf("Sentinels = %v, want 2 entries", result.Sentinels)
	}
	gotB, gotA := false, false
	for _, s := range result.Sentinels {
		if s == "sentB" {
			gotB = true
		}
		if s == "sentA" {
			gotA = true
		}
	}
	if !gotB || !gotA {
		t.Errorf("Sentinels = %v, want both sentB and sentA", result.Sentinels)
	}
	if result.Cause != nil {
		t.Errorf("Cause = %+v, want nil (no redundant duplication)", result.Cause)
	}
}

// Bug #2: Diamond / shared-cause patterns were wrongly reported as "(circular reference)"
// because the visited map was threaded across all branches. After the fix, the
// visited tracker scopes to the current path so siblings can independently
// serialize a shared cause.
func TestMarshal_SharedCauseAcrossSiblingsNotCircular(t *testing.T) {
	shared := errors.New("shared cause")
	err := errors.Join(errx.Wrap("branch1", shared), errx.Wrap("branch2", shared))

	data, marshalErr := errxjson.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Marshal error: %v", marshalErr)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(result.Causes) != 2 {
		t.Fatalf("len(Causes) = %d, want 2", len(result.Causes))
	}

	for i, c := range result.Causes {
		if c.Cause == nil {
			t.Fatalf("Causes[%d].Cause is nil", i)
		}
		if c.Cause.Message == "(circular reference)" {
			t.Errorf("Causes[%d].Cause.Message = %q, sibling sharing must not be flagged circular",
				i, c.Cause.Message)
		}
		if c.Cause.Message != "shared cause" {
			t.Errorf("Causes[%d].Cause.Message = %q, want %q", i, c.Cause.Message, "shared cause")
		}
	}
}

// Note: a test for true self-referential Unwrap cycles is intentionally
// omitted. The standard library's errors.As (invoked by IsDisplayable and
// HasAttrs) has no cycle protection and would loop indefinitely on a true
// Unwrap cycle before our own visited tracker ever runs. The visited
// tracker's role here is solely to prevent re-descending the same pointer
// when our recursion would otherwise do so on a DAG — which is also what
// max-depth ultimately bounds. Per-path tracking is correct semantically
// (siblings in a DAG must not poison each other) and preserves the same
// bounded behavior on any pointer that legitimately appears twice on a
// single descent.

// Bug #3: Sentinels were leaking upward across carrier levels because
// extractFromCarrierCauses walked up to 2 carrier hops and accumulated
// classifications from deeper levels into the current level. After the fix,
// each level only contains its own classifications.
func TestMarshal_SentinelsDoNotLeakAcrossLevels(t *testing.T) {
	ErrA := errx.NewSentinel("sentA")
	ErrB := errx.NewSentinel("sentB")
	ErrC := errx.NewSentinel("sentC")
	err := errx.Wrap("L1", errx.Wrap("L2", errx.Classify(errors.New("base"), ErrA), ErrB), ErrC)

	data, marshalErr := errxjson.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Marshal error: %v", marshalErr)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// L1: only sentC
	if len(result.Sentinels) != 1 || result.Sentinels[0] != "sentC" {
		t.Errorf("L1 Sentinels = %v, want [\"sentC\"]", result.Sentinels)
	}
	if result.Cause == nil {
		t.Fatal("L1.Cause is nil")
	}

	// L2: only sentB
	l2 := result.Cause
	if len(l2.Sentinels) != 1 || l2.Sentinels[0] != "sentB" {
		t.Errorf("L2 Sentinels = %v, want [\"sentB\"]", l2.Sentinels)
	}
	if l2.Cause == nil {
		t.Fatal("L2.Cause is nil")
	}

	// L3 (the Classify level): only sentA
	l3 := l2.Cause
	if len(l3.Sentinels) != 1 || l3.Sentinels[0] != "sentA" {
		t.Errorf("L3 Sentinels = %v, want [\"sentA\"]", l3.Sentinels)
	}
}

// Bug #4: A single non-JSON-serializable attribute value used to abort the
// entire Marshal call. After the fix, the bad value is replaced with a
// fmt.Sprintf("%+v", v) fallback so the rest of the report is preserved.
func TestMarshal_BadAttributeValueDoesNotAbort(t *testing.T) {
	attrErr := errx.Attrs("callback", func() {}, "user_id", 42)
	err := errx.Classify(errors.New("base"), attrErr)

	data, marshalErr := errxjson.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Marshal error: %v", marshalErr)
	}
	if data == nil {
		t.Fatal("Marshal returned nil bytes")
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if result.Message != "base" {
		t.Errorf("Message = %q, want %q", result.Message, "base")
	}

	if len(result.Attributes) != 2 {
		t.Fatalf("len(Attributes) = %d, want 2", len(result.Attributes))
	}

	attrMap := make(map[string]any)
	for _, a := range result.Attributes {
		attrMap[a.Key] = a.Value
	}
	cb, ok := attrMap["callback"].(string)
	if !ok {
		t.Fatalf("callback value = %T, want string fallback", attrMap["callback"])
	}
	if cb == "" {
		t.Error("callback fallback string is empty")
	}
	if attrMap["user_id"] != float64(42) {
		t.Errorf("user_id = %v, want 42", attrMap["user_id"])
	}
}

// Lock in correct passthrough for value types that JSON natively supports.
func TestMarshal_AttrPassThroughTypes(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	attrErr := errx.Attrs("when", ts, "marshaler", customMarshaler{n: 7})
	err := errx.Classify(errors.New("base"), attrErr)

	data, marshalErr := errxjson.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Marshal error: %v", marshalErr)
	}

	s := string(data)
	// time.Time marshals as RFC 3339 string under "value"
	if !strings.Contains(s, `"key":"when","value":"2026-01-02T03:04:05Z"`) {
		t.Errorf("output does not contain expected time encoding: %s", s)
	}
	// custom json.Marshaler value should pass through its custom encoding
	if !strings.Contains(s, `"key":"marshaler","value":{"custom":7}`) {
		t.Errorf("output does not contain expected custom marshaler encoding: %s", s)
	}
}

type customMarshaler struct{ n int }

func (c customMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(`{"custom":` + itoa(c.n) + `}`), nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := false
	if n < 0 {
		negative = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
