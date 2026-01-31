package errx_test

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"testing"

	"github.com/go-extras/errx"
)

func TestWithAttrs(t *testing.T) {
	attributed := errx.Attrs(
		errx.Attr{Key: "user_id", Value: 123},
		errx.Attr{Key: "action", Value: "delete"},
	)

	if attributed == nil {
		t.Fatal("expected non-nil error")
	}

	// Verify attributes are attached
	extractedAttrs := errx.ExtractAttrs(attributed)
	if len(extractedAttrs) != 2 {
		t.Errorf("expected 2 attrs, got %d", len(extractedAttrs))
	}
}

func TestFromAttrMap(t *testing.T) {
	attrs := map[string]any{
		"user_id": 123,
		"action":  "delete",
		"count":   5,
	}

	attributed := errx.FromAttrMap(attrs)

	if attributed == nil {
		t.Fatal("expected non-nil error")
	}

	// FromAttrMap creates an attributed error
	extractedAttrs := errx.ExtractAttrs(attributed)
	if len(extractedAttrs) != len(attrs) {
		t.Errorf("expected %d attrs, got %d", len(attrs), len(extractedAttrs))
	}
}

func TestHasAttrs_WithAttributed(t *testing.T) {
	attributed := errx.Attrs(errx.Attr{Key: "key", Value: "value"})

	if !errx.HasAttrs(attributed) {
		t.Error("expected HasAttrs to return true")
	}
}

func TestHasAttrs_WithRegularError(t *testing.T) {
	err := errors.New("test error")

	if errx.HasAttrs(err) {
		t.Error("expected HasAttrs to return false")
	}
}

func TestHasAttrs_WithNil(t *testing.T) {
	if errx.HasAttrs(nil) {
		t.Error("expected HasAttrs to return false for nil")
	}
}

func TestHasAttrs_WithWrappedAttributed(t *testing.T) {
	attributed := errx.Attrs(errx.Attr{Key: "key", Value: "value"})
	wrapped := fmt.Errorf("context: %w", attributed)

	if !errx.HasAttrs(wrapped) {
		t.Error("expected HasAttrs to return true for wrapped attributed")
	}
}

func TestExtractAttrs_WithAttributed(t *testing.T) {
	attributed := errx.Attrs(
		errx.Attr{Key: "user_id", Value: 123},
		errx.Attr{Key: "action", Value: "delete"},
	)

	attrs := errx.ExtractAttrs(attributed)

	if len(attrs) != 2 {
		t.Fatalf("expected 2 attrs, got %d", len(attrs))
	}
	if attrs[0].Key != "user_id" || attrs[0].Value != 123 {
		t.Errorf("expected user_id=123, got %s=%v", attrs[0].Key, attrs[0].Value)
	}
	if attrs[1].Key != "action" || attrs[1].Value != "delete" {
		t.Errorf("expected action=delete, got %s=%v", attrs[1].Key, attrs[1].Value)
	}
}

func TestExtractAttrs_WithRegularError(t *testing.T) {
	err := errors.New("test error")
	attrs := errx.ExtractAttrs(err)

	if len(attrs) != 0 {
		t.Errorf("expected empty slice, got %d attrs", len(attrs))
	}
}

func TestExtractAttrs_WithNil(t *testing.T) {
	attrs := errx.ExtractAttrs(nil)

	if len(attrs) != 0 {
		t.Errorf("expected empty slice, got %d attrs", len(attrs))
	}
}

func TestExtractAttrs_WithWrappedAttributed(t *testing.T) {
	attributed := errx.Attrs(errx.Attr{Key: "key", Value: "value"})
	wrapped := fmt.Errorf("context: %w", attributed)

	attrs := errx.ExtractAttrs(wrapped)

	if len(attrs) != 1 {
		t.Fatalf("expected 1 attr, got %d", len(attrs))
	}
	if attrs[0].Key != "key" || attrs[0].Value != "value" {
		t.Errorf("expected key=value, got %s=%v", attrs[0].Key, attrs[0].Value)
	}
}

func TestExtractAttrs_WithDeepChain(t *testing.T) {
	attributed := errx.Attrs(errx.Attr{Key: "deep", Value: "value"})
	level1 := fmt.Errorf("level1: %w", attributed)
	level2 := fmt.Errorf("level2: %w", level1)
	level3 := fmt.Errorf("level3: %w", level2)

	attrs := errx.ExtractAttrs(level3)

	if len(attrs) != 1 {
		t.Fatalf("expected 1 attr, got %d", len(attrs))
	}
	if attrs[0].Key != "deep" || attrs[0].Value != "value" {
		t.Errorf("expected deep=value, got %s=%v", attrs[0].Key, attrs[0].Value)
	}
}

func TestExtractAttrs_WithMultipleAttributed(t *testing.T) {
	attributed1 := errx.Attrs(errx.Attr{Key: "first", Value: 1})
	wrapped := fmt.Errorf("layer1: %w", attributed1)
	attributed2 := errx.Classify(wrapped, errx.Attrs(errx.Attr{Key: "second", Value: 2}))

	attrs := errx.ExtractAttrs(attributed2)

	// Should have attrs from both levels
	if len(attrs) < 1 {
		t.Fatalf("expected at least 1 attr, got %d", len(attrs))
	}
}

func TestFromAttrMap_PreservesValues(t *testing.T) {
	original := map[string]any{
		"string":  "hello",
		"int":     42,
		"float":   3.14,
		"bool":    true,
		"nil":     nil,
		"complex": map[string]int{"nested": 1},
	}

	attributed := errx.FromAttrMap(original)
	attrs := errx.ExtractAttrs(attributed)

	if len(attrs) != len(original) {
		t.Fatalf("expected %d attrs, got %d", len(original), len(attrs))
	}

	attrMap := make(map[string]any)
	for _, attr := range attrs {
		attrMap[attr.Key] = attr.Value
	}

	for key, val := range original {
		extracted, ok := attrMap[key]
		if !ok {
			t.Errorf("missing key %q", key)
			continue
		}
		if !reflect.DeepEqual(extracted, val) {
			t.Errorf("key %q: expected %v, got %v", key, val, extracted)
		}
	}
}

func TestWithAttrs_WithEmptyAttrs(t *testing.T) {
	attributed := errx.Attrs()

	if attributed == nil {
		t.Fatal("expected non-nil error")
	}

	attrs := errx.ExtractAttrs(attributed)
	if len(attrs) != 0 {
		t.Errorf("expected 0 attrs, got %d", len(attrs))
	}
}

func TestFromAttrMap_WithEmptyMap(t *testing.T) {
	attributed := errx.FromAttrMap(make(map[string]any))

	if attributed == nil {
		t.Fatal("expected non-nil error")
	}

	attrs := errx.ExtractAttrs(attributed)
	if len(attrs) != 0 {
		t.Errorf("expected 0 attrs, got %d", len(attrs))
	}
}

func TestFromAttrMap_WithNilMap(t *testing.T) {
	attributed := errx.FromAttrMap(nil)

	if attributed == nil {
		t.Fatal("expected non-nil error")
	}

	attrs := errx.ExtractAttrs(attributed)
	if len(attrs) != 0 {
		t.Errorf("expected 0 attrs, got %d", len(attrs))
	}
}

func TestAttrs_WithWrap(t *testing.T) {
	err := errors.New("test error")
	attributed := errx.Classify(err, errx.Attrs(errx.Attr{Key: "user_id", Value: 123}))
	ErrNotFound := errx.NewSentinel("not found")
	wrapped := errx.Wrap("operation failed", attributed, ErrNotFound)

	// Should preserve attributes
	if !errx.HasAttrs(wrapped) {
		t.Error("expected wrapped error to have attrs")
	}

	attrs := errx.ExtractAttrs(wrapped)
	if len(attrs) != 1 {
		t.Fatalf("expected 1 attr, got %d", len(attrs))
	}
	if attrs[0].Key != "user_id" || attrs[0].Value != 123 {
		t.Errorf("expected user_id=123, got %s=%v", attrs[0].Key, attrs[0].Value)
	}
}

func TestAttrs_WithClassify(t *testing.T) {
	err := errors.New("test error")
	attributedCls := errx.Attrs(errx.Attr{Key: "key", Value: "value"})
	ErrNotFound := errx.NewSentinel("not found")
	classified := errx.Classify(err, attributedCls, ErrNotFound)

	// Should preserve attributes
	if !errx.HasAttrs(classified) {
		t.Error("expected classified error to have attrs")
	}

	// Should match sentinel
	if !errors.Is(classified, ErrNotFound) {
		t.Error("expected error to match sentinel")
	}

	attrs := errx.ExtractAttrs(classified)
	if len(attrs) != 1 {
		t.Fatalf("expected 1 attr, got %d", len(attrs))
	}
	if attrs[0].Key != "key" || attrs[0].Value != "value" {
		t.Errorf("expected key=value, got %s=%v", attrs[0].Key, attrs[0].Value)
	}
}

func TestAttrs_WithDisplayable(t *testing.T) {
	displayErr := errx.NewDisplayable("user message")
	attributed := errx.Classify(displayErr, errx.Attrs(errx.Attr{Key: "code", Value: 404}))

	// Should be displayable
	if !errx.IsDisplayable(attributed) {
		t.Error("expected error to be displayable")
	}

	// Should have attrs
	if !errx.HasAttrs(attributed) {
		t.Error("expected error to have attrs")
	}

	// DisplayText should work
	text := errx.DisplayText(attributed)
	if text != "user message" {
		t.Errorf("expected 'user message', got %q", text)
	}

	// ExtractAttrs should work
	attrs := errx.ExtractAttrs(attributed)
	if len(attrs) != 1 {
		t.Fatalf("expected 1 attr, got %d", len(attrs))
	}
	if attrs[0].Key != "code" || attrs[0].Value != 404 {
		t.Errorf("expected code=404, got %s=%v", attrs[0].Key, attrs[0].Value)
	}
}

func TestAttrs_ComplexScenario(t *testing.T) {
	// Create a rich error with multiple features
	ErrValidation := errx.NewSentinel("validation error")
	displayErr := errx.NewDisplayable("Invalid input provided")
	attributedCls := errx.Attrs(
		errx.Attr{Key: "field", Value: "email"},
		errx.Attr{Key: "value", Value: "invalid@"},
	)
	classified := errx.Classify(displayErr, attributedCls, ErrValidation)
	wrapped := errx.Wrap("failed to process request", classified)
	final := fmt.Errorf("handler error: %w", wrapped)

	// Should match sentinel
	if !errors.Is(final, ErrValidation) {
		t.Error("expected error to match ErrValidation")
	}

	// Should be displayable
	if !errx.IsDisplayable(final) {
		t.Error("expected error to be displayable")
	}

	// Should have attrs
	if !errx.HasAttrs(final) {
		t.Error("expected error to have attrs")
	}

	// DisplayText should extract user message
	text := errx.DisplayText(final)
	if text != "Invalid input provided" {
		t.Errorf("expected 'Invalid input provided', got %q", text)
	}

	// ExtractAttrs should get all attributes
	attrs := errx.ExtractAttrs(final)
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attrs, got %d", len(attrs))
	}
}

func TestExtractAttrs_OrderPreservation(t *testing.T) {
	attributed := errx.Attrs(
		errx.Attr{Key: "first", Value: 1},
		errx.Attr{Key: "second", Value: 2},
		errx.Attr{Key: "third", Value: 3},
	)

	attrs := errx.ExtractAttrs(attributed)

	if len(attrs) != 3 {
		t.Fatalf("expected 3 attrs, got %d", len(attrs))
	}

	// Check order is preserved
	if attrs[0].Key != "first" || attrs[0].Value != 1 {
		t.Errorf("expected first attr: first=1, got %s=%v", attrs[0].Key, attrs[0].Value)
	}
	if attrs[1].Key != "second" || attrs[1].Value != 2 {
		t.Errorf("expected second attr: second=2, got %s=%v", attrs[1].Key, attrs[1].Value)
	}
	if attrs[2].Key != "third" || attrs[2].Value != 3 {
		t.Errorf("expected third attr: third=3, got %s=%v", attrs[2].Key, attrs[2].Value)
	}
}

// TestWithAttrs_OddNumberOfArgs tests Attrs with odd number of string arguments
func TestWithAttrs_OddNumberOfArgs(t *testing.T) {
	// When a string is the last argument with no value, it gets !BADKEY
	attributed := errx.Attrs("key1", "value1", "key2")

	attrs := errx.ExtractAttrs(attributed)

	if len(attrs) != 2 {
		t.Fatalf("expected 2 attrs, got %d", len(attrs))
	}

	// First pair should be normal
	if attrs[0].Key != "key1" || attrs[0].Value != "value1" {
		t.Errorf("expected key1=value1, got %s=%v", attrs[0].Key, attrs[0].Value)
	}

	// Second should have !BADKEY as key and "key2" as value
	if attrs[1].Key != "!BADKEY" || attrs[1].Value != "key2" {
		t.Errorf("expected !BADKEY=key2, got %s=%v", attrs[1].Key, attrs[1].Value)
	}
}

// TestWithAttrs_NonStringNonAttr tests Attrs with non-string, non-Attr values
func TestWithAttrs_NonStringNonAttr(t *testing.T) {
	// Non-string, non-Attr values get !BADKEY as key
	attributed := errx.Attrs(123, 456, true)

	attrs := errx.ExtractAttrs(attributed)

	if len(attrs) != 3 {
		t.Fatalf("expected 3 attrs, got %d", len(attrs))
	}

	// All should have !BADKEY as key
	if attrs[0].Key != "!BADKEY" || attrs[0].Value != 123 {
		t.Errorf("expected !BADKEY=123, got %s=%v", attrs[0].Key, attrs[0].Value)
	}
	if attrs[1].Key != "!BADKEY" || attrs[1].Value != 456 {
		t.Errorf("expected !BADKEY=456, got %s=%v", attrs[1].Key, attrs[1].Value)
	}
	if attrs[2].Key != "!BADKEY" || attrs[2].Value != true {
		t.Errorf("expected !BADKEY=true, got %s=%v", attrs[2].Key, attrs[2].Value)
	}
}

// TestWithAttrs_MixedFormats tests Attrs with mixed input formats
func TestWithAttrs_MixedFormats(t *testing.T) {
	// Mix of key-value pairs, Attr structs, and slices
	attributed := errx.Attrs(
		"key1", "value1",
		errx.Attr{Key: "key2", Value: "value2"},
		[]errx.Attr{{Key: "key3", Value: "value3"}, {Key: "key4", Value: "value4"}},
		"key5", 123,
	)

	attrs := errx.ExtractAttrs(attributed)

	if len(attrs) != 5 {
		t.Fatalf("expected 5 attrs, got %d", len(attrs))
	}

	expected := []errx.Attr{
		{Key: "key1", Value: "value1"},
		{Key: "key2", Value: "value2"},
		{Key: "key3", Value: "value3"},
		{Key: "key4", Value: "value4"},
		{Key: "key5", Value: 123},
	}

	for i, exp := range expected {
		if attrs[i].Key != exp.Key || attrs[i].Value != exp.Value {
			t.Errorf("attr %d: expected %s=%v, got %s=%v", i, exp.Key, exp.Value, attrs[i].Key, attrs[i].Value)
		}
	}
}

// TestWithAttrs_StringFollowedByNonString tests string key followed by non-string value
func TestWithAttrs_StringFollowedByNonString(t *testing.T) {
	// String keys can have any type of value
	attributed := errx.Attrs(
		"int_key", 42,
		"bool_key", true,
		"nil_key", nil,
		"struct_key", struct{ Name string }{Name: "test"},
	)

	attrs := errx.ExtractAttrs(attributed)

	if len(attrs) != 4 {
		t.Fatalf("expected 4 attrs, got %d", len(attrs))
	}

	if attrs[0].Key != "int_key" || attrs[0].Value != 42 {
		t.Errorf("expected int_key=42, got %s=%v", attrs[0].Key, attrs[0].Value)
	}
	if attrs[1].Key != "bool_key" || attrs[1].Value != true {
		t.Errorf("expected bool_key=true, got %s=%v", attrs[1].Key, attrs[1].Value)
	}
	if attrs[2].Key != "nil_key" || attrs[2].Value != nil {
		t.Errorf("expected nil_key=nil, got %s=%v", attrs[2].Key, attrs[2].Value)
	}
	if attrs[3].Key != "struct_key" {
		t.Errorf("expected struct_key, got %s", attrs[3].Key)
	}
}

// TestWithAttrs_AttrsTypeAlias tests using AttrList type alias
func TestWithAttrs_AttrsTypeAlias(t *testing.T) {
	// AttrList is an alias for []Attr and should work the same
	attrsList := errx.AttrList{
		{Key: "key1", Value: "value1"},
		{Key: "key2", Value: "value2"},
	}

	attributed := errx.Attrs(attrsList)

	attrs := errx.ExtractAttrs(attributed)

	if len(attrs) != 2 {
		t.Fatalf("expected 2 attrs, got %d", len(attrs))
	}

	if attrs[0].Key != "key1" || attrs[0].Value != "value1" {
		t.Errorf("expected key1=value1, got %s=%v", attrs[0].Key, attrs[0].Value)
	}
	if attrs[1].Key != "key2" || attrs[1].Value != "value2" {
		t.Errorf("expected key2=value2, got %s=%v", attrs[1].Key, attrs[1].Value)
	}
}

// multiError is a test helper type that implements Unwrap() []error for Go 1.20+ multi-error support
type multiError struct {
	errs []error
}

func (*multiError) Error() string {
	return "multiple errors occurred"
}

func (m *multiError) Unwrap() []error {
	return m.errs
}

// TestExtractAttrs_WithMultiError tests ExtractAttrs with Go 1.20+ multi-errors
func TestExtractAttrs_WithMultiError(t *testing.T) {
	// Create attributed errors
	attr1 := errx.Attrs("key1", "value1")
	attr2 := errx.Attrs("key2", "value2")
	attr3 := errx.Attrs("key3", "value3")

	// Create a multi-error containing attributed errors
	multiErr := &multiError{
		errs: []error{attr1, attr2, attr3},
	}

	// Extract attributes from multi-error
	attrs := errx.ExtractAttrs(multiErr)

	if len(attrs) != 3 {
		t.Fatalf("expected 3 attrs, got %d", len(attrs))
	}

	// Verify all attributes are extracted
	expectedKeys := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for _, attr := range attrs {
		expectedValue, ok := expectedKeys[attr.Key]
		if !ok {
			t.Errorf("unexpected key: %s", attr.Key)
			continue
		}
		if attr.Value != expectedValue {
			t.Errorf("for key %s: expected %v, got %v", attr.Key, expectedValue, attr.Value)
		}
	}
}

// TestExtractAttrs_WithNestedMultiError tests ExtractAttrs with nested multi-errors
func TestExtractAttrs_WithNestedMultiError(t *testing.T) {
	// Create attributed errors
	attr1 := errx.Attrs("key1", "value1")
	attr2 := errx.Attrs("key2", "value2")

	// Create nested multi-errors
	innerMulti := &multiError{
		errs: []error{attr1},
	}

	outerMulti := &multiError{
		errs: []error{innerMulti, attr2},
	}

	// Extract attributes from nested multi-error
	attrs := errx.ExtractAttrs(outerMulti)

	if len(attrs) != 2 {
		t.Fatalf("expected 2 attrs, got %d", len(attrs))
	}

	// Verify all attributes are extracted
	expectedKeys := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	for _, attr := range attrs {
		expectedValue, ok := expectedKeys[attr.Key]
		if !ok {
			t.Errorf("unexpected key: %s", attr.Key)
			continue
		}
		if attr.Value != expectedValue {
			t.Errorf("for key %s: expected %v, got %v", attr.Key, expectedValue, attr.Value)
		}
	}
}

// TestHasAttrs_WithMultiError tests HasAttrs with multi-errors
func TestHasAttrs_WithMultiError(t *testing.T) {
	// Create attributed error
	attr := errx.Attrs("key", "value")

	// Create multi-error containing attributed error
	multiErr := &multiError{
		errs: []error{attr, errors.New("regular error")},
	}

	// Should detect attributes in multi-error
	if !errx.HasAttrs(multiErr) {
		t.Error("expected HasAttrs to return true for multi-error containing attributed error")
	}

	// Multi-error without attributed errors
	multiErrNoAttrs := &multiError{
		errs: []error{errors.New("error1"), errors.New("error2")},
	}

	if errx.HasAttrs(multiErrNoAttrs) {
		t.Error("expected HasAttrs to return false for multi-error without attributed errors")
	}
}

func TestAttrs_ToSlogAttrs(t *testing.T) {
	tests := []struct {
		name     string
		attrs    errx.AttrList
		expected []slog.Attr
	}{
		{
			name: "basic conversion",
			attrs: errx.AttrList{
				{Key: "user_id", Value: 123},
				{Key: "action", Value: "delete"},
			},
			expected: []slog.Attr{
				slog.Any("user_id", 123),
				slog.Any("action", "delete"),
			},
		},
		{
			name: "mixed types",
			attrs: errx.AttrList{
				{Key: "string", Value: "test"},
				{Key: "int", Value: 42},
				{Key: "bool", Value: true},
				{Key: "float", Value: 3.14},
			},
			expected: []slog.Attr{
				slog.Any("string", "test"),
				slog.Any("int", 42),
				slog.Any("bool", true),
				slog.Any("float", 3.14),
			},
		},
		{
			name:     "empty attrs",
			attrs:    errx.AttrList{},
			expected: nil,
		},
		{
			name:     "nil attrs",
			attrs:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.attrs.ToSlogAttrs()

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d attrs, got %d", len(tt.expected), len(result))
			}

			for i := range result {
				if !result[i].Equal(tt.expected[i]) {
					t.Errorf("attr %d: expected %v, got %v", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

func TestAttrs_ToSlogAttrs_Integration(t *testing.T) {
	// Create an error with attributes
	err := errx.Attrs("user_id", 123, "action", "delete", "timestamp", "2024-01-01")

	// Extract attributes
	attrs := errx.ExtractAttrs(err)
	if attrs == nil {
		t.Fatal("expected non-nil attrs")
	}

	// Convert to slog.Attr
	slogAttrs := attrs.ToSlogAttrs()
	if slogAttrs == nil {
		t.Fatal("expected non-nil slog attrs")
	}

	if len(slogAttrs) != 3 {
		t.Errorf("expected 3 slog attrs, got %d", len(slogAttrs))
	}

	// Verify the attributes can be used with slog
	// (This is a compile-time check more than runtime)
	_ = []any{slogAttrs[0], slogAttrs[1], slogAttrs[2]}
}

func TestAttrs_ToSlogArgs(t *testing.T) {
	tests := []struct {
		name     string
		attrs    errx.AttrList
		expected []any
	}{
		{
			name: "basic conversion",
			attrs: errx.AttrList{
				{Key: "user_id", Value: 123},
				{Key: "action", Value: "delete"},
			},
			expected: []any{
				slog.Any("user_id", 123),
				slog.Any("action", "delete"),
			},
		},
		{
			name: "mixed types",
			attrs: errx.AttrList{
				{Key: "string", Value: "test"},
				{Key: "int", Value: 42},
				{Key: "bool", Value: true},
				{Key: "float", Value: 3.14},
			},
			expected: []any{
				slog.Any("string", "test"),
				slog.Any("int", 42),
				slog.Any("bool", true),
				slog.Any("float", 3.14),
			},
		},
		{
			name:     "empty attrs",
			attrs:    errx.AttrList{},
			expected: nil,
		},
		{
			name:     "nil attrs",
			attrs:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.attrs.ToSlogArgs()

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d args, got %d", len(tt.expected), len(result))
			}

			for i := range result {
				// Convert both to slog.Attr for comparison
				resultAttr, ok1 := result[i].(slog.Attr)
				expectedAttr, ok2 := tt.expected[i].(slog.Attr)

				if !ok1 || !ok2 {
					t.Errorf("arg %d: expected slog.Attr, got %T and %T", i, result[i], tt.expected[i])
					continue
				}

				if !resultAttr.Equal(expectedAttr) {
					t.Errorf("arg %d: expected %v, got %v", i, expectedAttr, resultAttr)
				}
			}
		})
	}
}

func TestAttrs_ToSlogArgs_Integration(t *testing.T) {
	// Create an error with attributes
	err := errx.Attrs("user_id", 123, "action", "delete", "timestamp", "2024-01-01")

	// Extract attributes
	attrs := errx.ExtractAttrs(err)
	if attrs == nil {
		t.Fatal("expected non-nil attrs")
	}

	// Convert to []any for use with slog.Error
	slogArgs := attrs.ToSlogArgs()
	if slogArgs == nil {
		t.Fatal("expected non-nil slog args")
	}

	if len(slogArgs) != 3 {
		t.Errorf("expected 3 slog args, got %d", len(slogArgs))
	}

	// Verify each element is a slog.Attr
	for i, arg := range slogArgs {
		if _, ok := arg.(slog.Attr); !ok {
			t.Errorf("arg %d: expected slog.Attr, got %T", i, arg)
		}
	}
}
