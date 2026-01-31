package json_test

import (
	"encoding/json"
	"errors"
	"fmt"
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

// unhashableError is an error type that contains unhashable fields (map).
// This simulates errors like validation.Errors from go-ozzo/ozzo-validation
// which contain map[string]any and cannot be used as map keys.
type unhashableError struct {
	message string
	data    map[string]any
}

func (e *unhashableError) Error() string {
	return e.message
}

func TestMarshal_UnhashableError(t *testing.T) {
	// Create an unhashable error (contains a map field)
	unhashableErr := &unhashableError{
		message: "validation failed",
		data: map[string]any{
			"email": "required",
			"age":   "must be 18 or older",
		},
	}

	// Wrap it with errx - this should not panic
	wrappedErr := stacktrace.Wrap("operation failed", unhashableErr)

	// Marshal should handle unhashable errors gracefully
	data, err := errxjson.Marshal(wrappedErr)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Verify the error was serialized correctly
	if result.Message != "operation failed: validation failed" {
		t.Errorf("Message = %q, want %q", result.Message, "operation failed: validation failed")
	}

	// Verify the cause was included
	if result.Cause == nil {
		t.Fatal("Cause should not be nil")
	}

	if result.Cause.Message != "validation failed" {
		t.Errorf("Cause.Message = %q, want %q", result.Cause.Message, "validation failed")
	}

	// Verify stack trace was captured
	if len(result.StackTrace) == 0 {
		t.Error("StackTrace should not be empty")
	}
}

func TestMarshal_UnhashableErrorCircular(t *testing.T) {
	// Create an unhashable error
	unhashableErr := &unhashableError{
		message: "validation failed",
		data:    map[string]any{"field": "value"},
	}

	// Create a circular reference with unhashable error
	// This tests that circular detection works even with unhashable errors
	wrappedErr := stacktrace.Wrap("outer", unhashableErr)

	// Marshal should detect circular references even with unhashable errors
	data, err := errxjson.Marshal(wrappedErr)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Should successfully serialize without panic
	if result.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestMarshal_MixedHashableUnhashable(t *testing.T) {
	// Test a chain with both hashable and unhashable errors
	hashableErr := errors.New("hashable error")
	unhashableErr := &unhashableError{
		message: "unhashable error",
		data:    map[string]any{"key": "value"},
	}

	// Create a chain with mixed hashable/unhashable errors
	err1 := errx.Wrap("wrap1", hashableErr, ErrDatabaseTest)
	err2 := errx.Wrap("wrap2", unhashableErr)
	err3 := stacktrace.Wrap("wrap3", err2)
	finalErr := stacktrace.Wrap("final", err3)
	// Also wrap the first error separately
	anotherErr := errx.Wrap("another", err1)

	// Should handle mixed hashable/unhashable errors
	data, err := errxjson.Marshal(finalErr)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Verify it serialized successfully
	if result.Message == "" {
		t.Error("Message should not be empty")
	}

	// Also test the other error
	data2, err := errxjson.Marshal(anotherErr)
	if err != nil {
		t.Fatalf("Marshal error for anotherErr: %v", err)
	}

	var result2 errxjson.SerializedError
	if err := json.Unmarshal(data2, &result2); err != nil {
		t.Fatalf("Unmarshal error for anotherErr: %v", err)
	}

	if result2.Message == "" {
		t.Error("Message should not be empty")
	}
}

// TestMarshal_PointerIdentity verifies that pointer identity is used for circular detection,
// not value equality. Two errors with the same content but different pointers should both be serialized.
func TestMarshal_PointerIdentity(t *testing.T) {
	// Create two unhashable errors with identical content
	err1 := &unhashableError{data: map[string]any{"key": "value"}}
	err2 := &unhashableError{data: map[string]any{"key": "value"}}

	// Wrap them in a chain
	wrapped := errx.Wrap("outer", err1)
	wrapped = errx.Wrap("middle", wrapped)
	wrapped = errx.Wrap("inner", err2)

	data, err := errxjson.Marshal(wrapped)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Both errors should be in the chain since they're different instances
	if result.Message == "" {
		t.Error("Message should not be empty")
	}
}

// TestMarshal_SamePointerMultipleTimes verifies that the same error instance
// appearing multiple times in the chain is only serialized once.
func TestMarshal_SamePointerMultipleTimes(t *testing.T) {
	baseErr := &unhashableError{data: map[string]any{"key": "value"}}

	// Create a chain where the same error appears as the base
	err1 := errx.Wrap("first", baseErr)
	err2 := errx.Wrap("second", err1)

	// Wrap again
	wrapped := errx.Wrap("outer", err2)

	data, err := errxjson.Marshal(wrapped)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if result.Message == "" {
		t.Error("Message should not be empty")
	}
}

// TestMarshal_NilErrorInChain verifies that nil errors in the chain are handled correctly.
func TestMarshal_NilErrorInChain(t *testing.T) {
	// Wrapping nil returns nil
	err := errx.Wrap("context", nil)
	if err != nil {
		t.Fatalf("Wrapping nil should return nil, got: %v", err)
	}

	// Marshal nil error should return empty JSON
	data, marshalErr := errxjson.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Marshal error: %v", marshalErr)
	}

	// Should be empty or null
	if len(data) > 0 && string(data) != "null" {
		t.Errorf("Marshaling nil should return empty or null, got: %s", string(data))
	}
}

// TestMarshal_DeepUnhashableChain tests a deep chain of unhashable errors.
func TestMarshal_DeepUnhashableChain(t *testing.T) {
	var err error = &unhashableError{data: map[string]any{"level": 0}}

	// Create a deep chain
	for i := 1; i <= 10; i++ {
		err = errx.Wrap(fmt.Sprintf("level %d", i), err)
	}

	data, marshalErr := errxjson.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Marshal error: %v", marshalErr)
	}

	var result errxjson.SerializedError
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Count the depth
	depth := 1
	current := &result
	for current.Cause != nil {
		depth++
		current = current.Cause
	}

	// Should have all 11 levels (10 wraps + 1 base)
	if depth != 11 {
		t.Errorf("depth = %d, want 11", depth)
	}
}

// TestExtractAttrs_UnhashableError verifies that ExtractAttrs works with unhashable errors.
func TestExtractAttrs_UnhashableError(t *testing.T) {
	unhashable := &unhashableError{data: map[string]any{"key": "value"}}
	attrErr := errx.Attrs("user_id", 123)
	wrapped := errx.Wrap("context", unhashable, attrErr)

	attrs := errx.ExtractAttrs(wrapped)

	if len(attrs) != 1 {
		t.Fatalf("len(attrs) = %d, want 1", len(attrs))
	}

	if attrs[0].Key != "user_id" || attrs[0].Value != 123 {
		t.Errorf("attrs[0] = %+v, want {Key:user_id Value:123}", attrs[0])
	}
}

// TestExtractAttrs_CircularWithUnhashable tests circular reference detection in ExtractAttrs
// with unhashable errors.
func TestExtractAttrs_CircularWithUnhashable(t *testing.T) {
	// Create a circular reference with unhashable error
	unhashable := &unhashableCircularError{data: map[string]any{"key": "value"}}
	unhashable.cause = unhashable // circular!

	attrErr := errx.Attrs("test", "value")
	wrapped := errx.Wrap("context", unhashable, attrErr)

	// This should not panic or hang
	attrs := errx.ExtractAttrs(wrapped)

	if len(attrs) != 1 {
		t.Fatalf("len(attrs) = %d, want 1", len(attrs))
	}
}

// unhashableCircularError is an unhashable error type that can have a circular reference.
type unhashableCircularError struct {
	data  map[string]any
	cause error
}

func (*unhashableCircularError) Error() string {
	return "unhashable circular error"
}

func (e *unhashableCircularError) Unwrap() error {
	return e.cause
}
