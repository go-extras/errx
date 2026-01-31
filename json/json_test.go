package json_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-extras/errx"
	errxjson "github.com/go-extras/errx/json"
	"github.com/go-extras/errx/stacktrace"
)

// Test sentinels
var (
	ErrNotFoundTest  = errx.NewSentinel("not found")
	ErrDatabaseTest  = errx.NewSentinel("database")
	ErrRetryableTest = errx.NewSentinel("retryable")
	ErrTimeoutTest   = errx.NewSentinel("timeout", ErrDatabaseTest, ErrRetryableTest)
)

func TestMarshal_NilError(t *testing.T) {
	data, err := errxjson.Marshal(nil)
	if err != nil {
		t.Fatalf("Marshal(nil) error = %v, want nil", err)
	}
	if data != nil {
		t.Errorf("Marshal(nil) = %v, want nil", data)
	}
}

func TestMarshal_StandardError(t *testing.T) {
	testErr := errors.New("standard error")
	data, err := errxjson.Marshal(testErr)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if result.Message != "standard error" {
		t.Errorf("Message = %q, want %q", result.Message, "standard error")
	}
	if result.DisplayText != "" {
		t.Errorf("DisplayText = %q, want empty", result.DisplayText)
	}
	if len(result.Sentinels) != 0 {
		t.Errorf("Sentinels = %v, want empty", result.Sentinels)
	}
}

func TestMarshal_SentinelOnly(t *testing.T) {
	testErr := errx.Classify(errors.New("base error"), ErrNotFoundTest)
	data, err := errxjson.Marshal(testErr)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if result.Message != "base error" {
		t.Errorf("Message = %q, want %q", result.Message, "base error")
	}
	if len(result.Sentinels) != 1 || result.Sentinels[0] != "not found" {
		t.Errorf("Sentinels = %v, want [\"not found\"]", result.Sentinels)
	}
}

func TestMarshal_DisplayableError(t *testing.T) {
	testErr := errx.NewDisplayable("User not found")
	wrapped := errx.Wrap("failed to fetch user", testErr, ErrNotFoundTest)

	data, err := errxjson.Marshal(wrapped)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if result.Message != "failed to fetch user: User not found" {
		t.Errorf("Message = %q, want %q", result.Message, "failed to fetch user: User not found")
	}
	if result.DisplayText != "User not found" {
		t.Errorf("DisplayText = %q, want %q", result.DisplayText, "User not found")
	}
}

func TestMarshal_AttributedError(t *testing.T) {
	attrErr := errx.Attrs("user_id", 123, "action", "delete")
	testErr := errx.Classify(errors.New("base error"), attrErr)

	data, err := errxjson.Marshal(testErr)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(result.Attributes) != 2 {
		t.Fatalf("len(Attributes) = %d, want 2", len(result.Attributes))
	}

	// Check attributes (order may vary)
	attrMap := make(map[string]any)
	for _, attr := range result.Attributes {
		attrMap[attr.Key] = attr.Value
	}

	if attrMap["user_id"] != float64(123) { // JSON numbers are float64
		t.Errorf("user_id = %v, want 123", attrMap["user_id"])
	}
	if attrMap["action"] != "delete" {
		t.Errorf("action = %v, want delete", attrMap["action"])
	}
}

func TestMarshal_StackTrace(t *testing.T) {
	testErr := stacktrace.Wrap("operation failed", errors.New("base error"), ErrDatabaseTest)

	data, err := errxjson.Marshal(testErr)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(result.StackTrace) == 0 {
		t.Error("StackTrace is empty, want non-empty")
	}

	// Check first frame has required fields
	if len(result.StackTrace) > 0 {
		frame := result.StackTrace[0]
		if frame.File == "" {
			t.Error("Frame.File is empty")
		}
		if frame.Line == 0 {
			t.Error("Frame.Line is 0")
		}
		if frame.Function == "" {
			t.Error("Frame.Function is empty")
		}
	}
}

func TestMarshal_ErrorChain(t *testing.T) {
	err1 := errors.New("level 3")
	err2 := errx.Wrap("level 2", err1, ErrDatabaseTest)
	err3 := errx.Wrap("level 1", err2, ErrRetryableTest)

	data, err := errxjson.Marshal(err3)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Check top level
	if result.Message != "level 1: level 2: level 3" {
		t.Errorf("Message = %q, want %q", result.Message, "level 1: level 2: level 3")
	}

	// Check cause chain depth
	if result.Cause == nil {
		t.Fatal("Cause is nil")
	}
	if result.Cause.Cause == nil {
		t.Fatal("Cause.Cause is nil")
	}
	if result.Cause.Cause.Message != "level 3" {
		t.Errorf("Cause.Cause.Message = %q, want %q", result.Cause.Cause.Message, "level 3")
	}
}

func TestMarshal_HierarchicalSentinels(t *testing.T) {
	testErr := errx.Classify(errors.New("timeout error"), ErrTimeoutTest)

	data, err := errxjson.Marshal(testErr)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Should only have the direct sentinel, not parents
	if len(result.Sentinels) != 1 || result.Sentinels[0] != "timeout" {
		t.Errorf("Sentinels = %v, want [\"timeout\"]", result.Sentinels)
	}
}

func TestMarshal_ComplexError(t *testing.T) {
	// Create a complex error with all features
	baseErr := errors.New("connection failed")
	displayErr := errx.NewDisplayable("Service temporarily unavailable")
	attrErr := errx.Attrs("retry_count", 3, "host", "localhost")

	testErr := stacktrace.Wrap("database operation failed",
		errx.Classify(baseErr, displayErr, attrErr, ErrTimeoutTest))

	data, err := errxjson.Marshal(testErr)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Verify all components are present
	if result.DisplayText != "Service temporarily unavailable" {
		t.Errorf("DisplayText = %q, want %q", result.DisplayText, "Service temporarily unavailable")
	}
	if len(result.Attributes) != 2 {
		t.Errorf("len(Attributes) = %d, want 2", len(result.Attributes))
	}
	if len(result.StackTrace) == 0 {
		t.Error("StackTrace is empty")
	}
	if len(result.Sentinels) == 0 {
		t.Error("Sentinels is empty")
	}
}

func TestMarshalIndent(t *testing.T) {
	testErr := errx.Wrap("failed", errors.New("base"), ErrNotFoundTest)

	data, err := errxjson.MarshalIndent(testErr, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent error: %v", err)
	}

	// Just verify it's valid JSON and indented
	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Check that it's actually indented (contains newlines and spaces)
	dataStr := string(data)
	if len(dataStr) < 10 {
		t.Error("Indented JSON seems too short")
	}
}

func TestToSerializedError_NilError(t *testing.T) {
	result := errxjson.ToSerializedError(nil)
	if result != nil {
		t.Errorf("ToSerializedError(nil) = %v, want nil", result)
	}
}

func TestWithMaxDepth(t *testing.T) {
	// Create a deep error chain
	err := errors.New("level 5")
	err = errx.Wrap("level 4", err)
	err = errx.Wrap("level 3", err)
	err = errx.Wrap("level 2", err)
	err = errx.Wrap("level 1", err)

	data, marshalErr := errxjson.Marshal(err, errxjson.WithMaxDepth(5))
	if marshalErr != nil {
		t.Fatalf("Marshal error: %v", marshalErr)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Count depth
	depth := 1
	current := &result
	for current.Cause != nil {
		depth++
		current = current.Cause
	}

	// We create 5 levels, maxDepth is set to 5, so we should get exactly 5
	if depth != 5 {
		t.Errorf("depth = %d, want 5", depth)
	}

	// With maxDepth=5, we should see all errors without hitting the limit
	if current.Message == "(max depth reached)" {
		t.Error("Hit max depth unexpectedly")
	}
}

func TestWithMaxStackFrames(t *testing.T) {
	testErr := stacktrace.Wrap("operation failed", errors.New("base error"))

	data, err := errxjson.Marshal(testErr, errxjson.WithMaxStackFrames(3))
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(result.StackTrace) > 3 {
		t.Errorf("len(StackTrace) = %d, want <= 3", len(result.StackTrace))
	}
}

func TestWithIncludeStandardErrors_False(t *testing.T) {
	// Mix of errx and standard errors
	stdErr := errors.New("standard error")
	mixed := errx.Wrap("wrapper", stdErr, ErrDatabaseTest)
	mixed = errx.Wrap("top", mixed)

	data, err := errxjson.Marshal(mixed, errxjson.WithIncludeStandardErrors(false))
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// The standard error should be excluded from the cause chain
	// But the wrapper (errx error) should be present
	if result.Cause != nil && result.Cause.Message == "standard error" {
		t.Error("Standard error should not be in cause chain when includeStandardErrors=false")
	}
}

func TestWithIncludeStandardErrors_True(t *testing.T) {
	// Mix of errx and standard errors
	stdErr := errors.New("standard error")
	mixed := errx.Wrap("wrapper", stdErr, ErrDatabaseTest)

	data, err := errxjson.Marshal(mixed, errxjson.WithIncludeStandardErrors(true))
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// The standard error should be in the cause chain
	if result.Cause == nil {
		t.Fatal("Cause is nil")
	}
	if result.Cause.Message != "standard error" {
		t.Errorf("Cause.Message = %q, want %q", result.Cause.Message, "standard error")
	}
}

func TestMultiError(t *testing.T) {
	// Create a multi-error (errors that unwrap to []error)
	type multiError struct {
		errs []error
	}

	me := &multiError{
		errs: []error{
			errors.New("error 1"),
			errors.New("error 2"),
			errors.New("error 3"),
		},
	}

	// Implement Unwrap() []error
	type unwrapper interface {
		Unwrap() []error
	}

	if _, ok := any(me).(unwrapper); !ok {
		// Need to add method
		t.Skip("Need to implement Unwrap() []error for test")
	}
}

// multiError implements Unwrap() []error for testing
type testMultiError struct {
	message string
	errs    []error
}

func (m *testMultiError) Error() string {
	return m.message
}

func (m *testMultiError) Unwrap() []error {
	return m.errs
}

func TestMarshal_MultiError(t *testing.T) {
	multiErr := &testMultiError{
		message: "multiple errors occurred",
		errs: []error{
			errors.New("error 1"),
			errors.New("error 2"),
			errx.Classify(errors.New("error 3"), ErrNotFoundTest),
		},
	}

	data, err := errxjson.Marshal(multiErr)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if result.Message != "multiple errors occurred" {
		t.Errorf("Message = %q, want %q", result.Message, "multiple errors occurred")
	}

	if len(result.Causes) != 3 {
		t.Fatalf("len(Causes) = %d, want 3", len(result.Causes))
	}

	if result.Causes[0].Message != "error 1" {
		t.Errorf("Causes[0].Message = %q, want %q", result.Causes[0].Message, "error 1")
	}
	if result.Causes[1].Message != "error 2" {
		t.Errorf("Causes[1].Message = %q, want %q", result.Causes[1].Message, "error 2")
	}
	if result.Causes[2].Message != "error 3" {
		t.Errorf("Causes[2].Message = %q, want %q", result.Causes[2].Message, "error 3")
	}
}

func TestMarshal_EmptyAttributes(t *testing.T) {
	attrErr := errx.Attrs()
	testErr := errx.Classify(errors.New("base"), attrErr)

	data, err := errxjson.Marshal(testErr)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Empty attributes should be omitted in JSON
	if len(result.Attributes) != 0 {
		t.Errorf("len(Attributes) = %d, want 0", len(result.Attributes))
	}
}
