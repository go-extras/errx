// Package json provides JSON serialization capabilities for errx errors.
//
// This package extends errx with JSON serialization while keeping the core
// errx package minimal and zero-dependency. It serializes all errx error types
// including sentinels, displayable errors, attributed errors, and stack traces.
//
// # Basic Usage
//
//	err := errx.Wrap("failed to process", cause, ErrNotFound)
//	jsonBytes, err := json.Marshal(err)
//
// # Pretty Printing
//
//	jsonBytes, err := json.MarshalIndent(err, "", "  ")
//
// # Configuration
//
//	jsonBytes, err := json.Marshal(err,
//	    json.WithMaxDepth(16),
//	    json.WithMaxStackFrames(10))
package json

import (
	"encoding/json"
	"errors"
	"reflect"

	"github.com/go-extras/errx"
	"github.com/go-extras/errx/stacktrace"
)

// SerializedError represents the JSON structure of an errx error.
// It captures all aspects of an errx error including classifications,
// attributes, stack traces, and the error chain.
type SerializedError struct {
	// Message is the error message from Error()
	Message string `json:"message"`

	// DisplayText contains the displayable error message if one exists
	DisplayText string `json:"display_text,omitempty"`

	// Sentinels lists all classification sentinel texts found in this error
	Sentinels []string `json:"sentinels,omitempty"`

	// Attributes contains structured key-value pairs attached to this error
	Attributes []SerializedAttr `json:"attributes,omitempty"`

	// StackTrace contains stack frames if a stack trace was captured
	StackTrace []SerializedFrame `json:"stack_trace,omitempty"`

	// Cause is the wrapped error (single unwrap)
	Cause *SerializedError `json:"cause,omitempty"`

	// Causes contains multiple wrapped errors (multi-error unwrap)
	Causes []*SerializedError `json:"causes,omitempty"`
}

// SerializedAttr represents a single attribute key-value pair.
type SerializedAttr struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// SerializedFrame represents a single stack frame.
type SerializedFrame struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
}

// config holds serialization configuration.
type config struct {
	maxDepth              int
	maxStackFrames        int
	includeStandardErrors bool
}

// defaultConfig returns the default configuration.
func defaultConfig() *config {
	return &config{
		maxDepth:              32,
		maxStackFrames:        32,
		includeStandardErrors: true,
	}
}

// visitedErrors tracks visited errors during serialization to detect circular references.
// It uses pointer identity rather than value equality, which works for all error types
// including those with unhashable fields.
type visitedErrors struct {
	// Map of error pointer addresses to track visited errors
	// We use uintptr as the key since it's always hashable
	visited map[uintptr]bool
}

// newVisitedErrors creates a new visitedErrors tracker.
func newVisitedErrors() *visitedErrors {
	return &visitedErrors{
		visited: make(map[uintptr]bool),
	}
}

// contains checks if an error has been visited based on pointer identity.
func (v *visitedErrors) contains(err error) bool {
	if err == nil {
		return false
	}
	// Get the pointer address of the error interface's data pointer
	// This works for all error types, including those with unhashable fields
	// UnsafePointer() is the preferred method, converted to uintptr which is always hashable
	ptr := uintptr(reflect.ValueOf(err).UnsafePointer())
	return v.visited[ptr]
}

// add marks an error as visited based on pointer identity.
func (v *visitedErrors) add(err error) {
	if err == nil {
		return
	}
	ptr := uintptr(reflect.ValueOf(err).UnsafePointer())
	v.visited[ptr] = true
}

// Marshal serializes an error to JSON bytes.
// It returns nil, nil for nil errors.
//
// Example:
//
//	err := errx.Wrap("failed", cause, ErrNotFound)
//	jsonBytes, err := json.Marshal(err)
func Marshal(err error, opts ...Option) ([]byte, error) {
	if err == nil {
		return nil, nil
	}

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	serialized := toSerializedError(err, cfg, newVisitedErrors(), 0)
	return json.Marshal(serialized)
}

// MarshalIndent serializes an error to pretty-printed JSON bytes.
// It returns nil, nil for nil errors.
//
// Example:
//
//	jsonBytes, err := json.MarshalIndent(err, "", "  ")
func MarshalIndent(err error, prefix, indent string, opts ...Option) ([]byte, error) {
	if err == nil {
		return nil, nil
	}

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	serialized := toSerializedError(err, cfg, newVisitedErrors(), 0)
	return json.MarshalIndent(serialized, prefix, indent)
}

// ToSerializedError converts an error to a SerializedError struct.
// It returns nil for nil errors.
// This is useful when you want to manipulate the structure before serializing.
//
// Example:
//
//	serialized := json.ToSerializedError(err)
//	// Manipulate serialized...
//	jsonBytes, _ := json.Marshal(serialized)
func ToSerializedError(err error, opts ...Option) *SerializedError {
	if err == nil {
		return nil
	}

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return toSerializedError(err, cfg, newVisitedErrors(), 0)
}

// toSerializedError recursively converts an error to SerializedError.
func toSerializedError(err error, cfg *config, visited *visitedErrors, depth int) *SerializedError {
	if err == nil {
		return nil
	}

	// Check depth limit
	if depth >= cfg.maxDepth {
		return &SerializedError{
			Message: "(max depth reached)",
		}
	}

	// Check for circular references
	if visited.contains(err) {
		return &SerializedError{
			Message: "(circular reference)",
		}
	}
	visited.add(err)

	result := &SerializedError{
		Message: err.Error(),
	}

	// Extract displayable text
	if errx.IsDisplayable(err) {
		result.DisplayText = errx.DisplayText(err)
	}

	// Extract sentinels - only from this error level, not the whole chain
	result.Sentinels = extractSentinelsFromError(err)

	// Extract attributes
	serializeAttributes(err, result)

	// Extract stack trace
	serializeStackTrace(err, cfg, result)

	// Handle unwrapping
	serializeCauses(err, cfg, visited, depth, result)

	return result
}

// serializeAttributes extracts and serializes attributes from an error.
func serializeAttributes(err error, result *SerializedError) {
	attrs := errx.ExtractAttrs(err)
	if len(attrs) == 0 {
		return
	}
	result.Attributes = make([]SerializedAttr, len(attrs))
	for i, attr := range attrs {
		result.Attributes[i] = SerializedAttr{
			Key:   attr.Key,
			Value: attr.Value,
		}
	}
}

// serializeStackTrace extracts and serializes stack frames from an error.
func serializeStackTrace(err error, cfg *config, result *SerializedError) {
	frames := stacktrace.Extract(err)
	if len(frames) == 0 {
		return
	}
	limit := len(frames)
	if cfg.maxStackFrames > 0 && limit > cfg.maxStackFrames {
		limit = cfg.maxStackFrames
	}
	result.StackTrace = make([]SerializedFrame, limit)
	for i := 0; i < limit; i++ {
		result.StackTrace[i] = SerializedFrame{
			File:     frames[i].File,
			Line:     frames[i].Line,
			Function: frames[i].Function,
		}
	}
}

// serializeCauses handles unwrapping and serialization of error causes.
func serializeCauses(err error, cfg *config, visited *visitedErrors, depth int, result *SerializedError) {
	// Check for multi-error first
	type unwrapper interface {
		Unwrap() []error
	}
	if u, ok := err.(unwrapper); ok {
		serializeMultiError(u, cfg, visited, depth, result)
		return
	}

	// Handle single unwrap
	serializeSingleCause(err, cfg, visited, depth, result)
}

// serializeMultiError serializes multiple error causes.
func serializeMultiError(u unwrapper, cfg *config, visited *visitedErrors, depth int, result *SerializedError) {
	unwrapped := u.Unwrap()
	if len(unwrapped) == 0 {
		return
	}
	result.Causes = make([]*SerializedError, 0, len(unwrapped))
	for _, ue := range unwrapped {
		if ue == nil || (!cfg.includeStandardErrors && !isErrxError(ue)) {
			continue
		}
		serialized := toSerializedError(ue, cfg, visited, depth+1)
		if serialized != nil {
			result.Causes = append(result.Causes, serialized)
		}
	}
}

// unwrapper is the multi-error unwrap interface.
type unwrapper interface {
	Unwrap() []error
}

// serializeSingleCause serializes a single error cause.
func serializeSingleCause(err error, cfg *config, visited *visitedErrors, depth int, result *SerializedError) {
	cause := errors.Unwrap(err)
	if cause == nil {
		return
	}

	// If the cause is a carrier, skip it and go to its inner cause
	if isCarrier(cause) {
		innerCause := errors.Unwrap(cause)
		if innerCause != nil && (cfg.includeStandardErrors || isErrxError(innerCause)) {
			result.Cause = toSerializedError(innerCause, cfg, visited, depth+1)
		}
		return
	}

	if cfg.includeStandardErrors || isErrxError(cause) {
		result.Cause = toSerializedError(cause, cfg, visited, depth+1)
	}
}

// extractSentinelsFromError extracts sentinel texts from the error and its immediate cause if it's a carrier.
func extractSentinelsFromError(err error) []string {
	if err == nil {
		return nil
	}

	var sentinels []string
	seenSentinels := make(map[string]bool)

	// Check if err itself is a carrier and extract its classifications
	carrierClassifications := extractCarrierClassifications(err)
	addPureSentinels(carrierClassifications, &sentinels, seenSentinels)

	// Also check causes if they're carriers (common pattern from Wrap and stacktrace.Wrap)
	// Look up to 2 levels deep to handle nested carriers
	extractFromCarrierCauses(err, &sentinels, seenSentinels)

	// Also check if err itself is a pure sentinel
	addSelfAsPureSentinel(err, &sentinels, seenSentinels)

	return sentinels
}

// addPureSentinels adds pure sentinel classifications to the sentinels list.
func addPureSentinels(classifications []errx.Classified, sentinels *[]string, seen map[string]bool) {
	for _, cls := range classifications {
		if isPureSentinel(cls) {
			text := cls.Error()
			if !seen[text] {
				*sentinels = append(*sentinels, text)
				seen[text] = true
			}
		}
	}
}

// isPureSentinel checks if a classified error is a pure sentinel.
func isPureSentinel(cls errx.Classified) bool {
	return !errx.IsDisplayable(cls) && !errx.HasAttrs(cls) && stacktrace.Extract(cls) == nil
}

// extractFromCarrierCauses extracts sentinels from carrier causes up to 2 levels deep.
func extractFromCarrierCauses(err error, sentinels *[]string, seen map[string]bool) {
	current := err
	for i := 0; i < 2; i++ {
		cause := errors.Unwrap(current)
		if cause == nil || !isCarrier(cause) {
			break
		}
		carrierClassifications := extractCarrierClassifications(cause)
		addPureSentinels(carrierClassifications, sentinels, seen)
		current = cause
	}
}

// addSelfAsPureSentinel checks if the error itself is a pure sentinel and adds it.
func addSelfAsPureSentinel(err error, sentinels *[]string, seen map[string]bool) {
	cls, ok := err.(errx.Classified)
	if !ok || !cls.IsClassified() {
		return
	}
	if !isPureSentinel(cls) {
		return
	}
	text := err.Error()
	if !seen[text] {
		*sentinels = append(*sentinels, text)
		seen[text] = true
	}
}

// extractCarrierClassifications uses reflection to extract classifications from a carrier.
func extractCarrierClassifications(err error) []errx.Classified {
	if err == nil {
		return nil
	}

	v := reflect.ValueOf(err)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	// Check if this is a carrier by looking for "classifications" field
	clsField := v.FieldByName("classifications")
	if !clsField.IsValid() {
		return nil
	}

	// Extract classifications slice using unsafe to bypass export restrictions
	var result []errx.Classified
	for i := 0; i < clsField.Len(); i++ {
		itemVal := clsField.Index(i)
		// Use unsafe to get interface value from unexported field
		if itemVal.CanAddr() {
			ptr := itemVal.UnsafePointer()
			item := *(*errx.Classified)(ptr)
			result = append(result, item)
		} else {
			// If can't addr, create a new addressable value
			newVal := reflect.New(itemVal.Type()).Elem()
			newVal.Set(itemVal)
			if newVal.CanAddr() {
				ptr := newVal.UnsafePointer()
				item := *(*errx.Classified)(ptr)
				result = append(result, item)
			}
		}
	}

	return result
}

// isErrxError checks if an error is an errx error (implements Classified).
func isErrxError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(errx.Classified)
	return ok
}

// isCarrier checks if an error is a carrier type (has classifications field).
func isCarrier(err error) bool {
	if err == nil {
		return false
	}

	v := reflect.ValueOf(err)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return false
	}

	return v.FieldByName("classifications").IsValid()
}
