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
	"fmt"

	"github.com/go-extras/errx"
	"github.com/go-extras/errx/internal/errptr"
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

// config holds serialization configuration. It is small enough to be passed
// by value so callers avoid an unnecessary heap allocation per Marshal call.
type config struct {
	maxDepth              int
	maxStackFrames        int
	includeStandardErrors bool

	// Opt-in toggles for serialized sections. All default to true so that the
	// out-of-the-box JSON output matches the pre-1.3 behavior; setting any to
	// false suppresses that section in the output.
	includeStackTrace bool
	includeAttributes bool
	includeSentinels  bool

	// attributeValueTransformer, when non-nil, may rewrite/redact attribute
	// values just before they are placed into the serialized output.
	attributeValueTransformer AttributeValueTransformer

	// stackTraceTrimPath, when non-empty, is stripped (prefix match) from each
	// stack frame's File field.
	stackTraceTrimPath string

	// maxMessageBytes, when > 0, truncates each serialized error's Message to
	// at most that many bytes (with a "...(truncated)" suffix).
	maxMessageBytes int
}

// defaultConfig returns the default configuration as a value (no allocation).
func defaultConfig() config {
	return config{
		maxDepth:              32,
		maxStackFrames:        32,
		includeStandardErrors: true,
		includeStackTrace:     true,
		includeAttributes:     true,
		includeSentinels:      true,
	}
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
		opt(&cfg)
	}

	// visited is lazily allocated on the first recursion; single-node errors
	// never trigger the allocation.
	var visited visitedSet
	serialized := toSerializedError(err, &cfg, &visited, 0)
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
		opt(&cfg)
	}

	var visited visitedSet
	serialized := toSerializedError(err, &cfg, &visited, 0)
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
		opt(&cfg)
	}

	var visited visitedSet
	return toSerializedError(err, &cfg, &visited, 0)
}

// visitedSet tracks errors visited along the current recursion path.
// It is scoped per-path (not per-branch): entries are added on entry to
// toSerializedError and removed on exit, so true cycles on a single descent
// are still detected, while siblings of a DAG do not poison each other.
//
// The set is allocated lazily on first use through enterVisited so that
// single-node errors never pay for the map allocation.
type visitedSet map[uintptr]bool

// enterVisited records ptr in the visited set, allocating it lazily on first
// use. It returns true if ptr was already present (a cycle on the current
// path). A zero ptr is treated as "not trackable" — neither recorded nor
// flagged as a cycle.
func enterVisited(visited *visitedSet, ptr uintptr) bool {
	if ptr == 0 {
		return false
	}
	if *visited == nil {
		*visited = visitedSet{ptr: true}
		return false
	}
	if (*visited)[ptr] {
		return true
	}
	(*visited)[ptr] = true
	return false
}

// exitVisited removes ptr from the visited set so siblings of a DAG branch
// do not poison each other. Safe to call with a zero ptr or a nil set.
func exitVisited(visited *visitedSet, ptr uintptr) {
	if ptr == 0 || *visited == nil {
		return
	}
	delete(*visited, ptr)
}

// toSerializedError recursively converts an error to SerializedError.
//
// At entry, it peels leading carriers so the "node" at this level is the
// first non-carrier in the chain. This avoids emitting the duplicate cause
// that errx.Classify and nested Classify previously produced. After the
// node, it absorbs at most one immediately-following carrier (the one
// errx.Wrap inserted alongside the fmt-wrap), so each Wrap level surfaces
// only its own classifications and deeper-level sentinels never leak up.
func toSerializedError(err error, cfg *config, visited *visitedSet, depth int) *SerializedError {
	if err == nil {
		return nil
	}

	// Check depth limit. A non-positive maxDepth means "unlimited", matching
	// WithMaxStackFrames semantics.
	if cfg.maxDepth > 0 && depth >= cfg.maxDepth {
		return &SerializedError{
			Message: "(max depth reached)",
		}
	}

	node, levelCls, nextCause := peelLevel(err)

	// Per-path cycle detection. Pop on exit so DAGs serialize fully.
	ptr := errptr.Get(node)
	if enterVisited(visited, ptr) {
		return &SerializedError{Message: "(circular reference)"}
	}
	defer exitVisited(visited, ptr)

	result := &SerializedError{
		Message: truncateMessage(node.Error(), cfg.maxMessageBytes),
	}

	// For the chain-wide extractors we pass the original err (rather than
	// the peeled node) so any classifications absorbed by this level (such
	// as a displayable/attributed/stacktrace classification attached via
	// errx.Classify or errx.Wrap) remain visible — they live in the
	// carrier's classifications slice, which errors.As only reaches via
	// the carrier's own As method.

	// Displayable: chain-wide (errors.As walks the error chain).
	if errx.IsDisplayable(err) {
		result.DisplayText = errx.DisplayText(err)
	}

	// Sentinels: level-scoped. Only this level's absorbed carrier
	// classifications and the node itself (if it's a pure sentinel) count.
	if cfg.includeSentinels {
		result.Sentinels = sentinelsForLevel(levelCls, node)
	}

	// Attributes: chain-wide (errx.ExtractAttrs walks the chain).
	// Per-value JSON-serializability is validated to keep one bad value
	// from aborting the whole marshal.
	if cfg.includeAttributes {
		serializeAttributes(err, cfg, result)
	}

	// Stack trace: chain-wide.
	if cfg.includeStackTrace {
		serializeStackTrace(err, cfg, result)
	}

	// Cause(s).
	serializeCauses(node, nextCause, cfg, visited, depth, result)

	return result
}

// peelLevel walks from err and returns the message-bearing node for this
// level, the classifications that belong to this level, and the cause to
// recurse into.
//
//   - When err itself is a carrier (top-level Classify, or recursion landing
//     on a carrier), all consecutive leading carriers are absorbed into this
//     level. The node is the first non-carrier; the next cause is its raw
//     Unwrap (no further absorption — that belongs to the next level). This
//     collapses redundant nested Classify levels without losing
//     classifications.
//
//   - When err is not a carrier, the node is err itself and exactly one
//     immediately-following carrier is absorbed (this is the carrier inserted
//     by the same errx.Wrap call as the fmt-wrap). Subsequent carriers
//     belong to a deeper level and are passed through as nextCause.
//
// Carrier detection uses the public errx.CarrierClassifications helper to
// avoid reflection or unsafe-pointer access to errx internals.
func peelLevel(err error) (node error, classifications []errx.Classified, nextCause error) {
	cur := err

	if cls, ok := errx.CarrierClassifications(cur); ok {
		// Classify-shaped entry: absorb all consecutive leading carriers.
		classifications = append(classifications, cls...)
		for {
			next := errors.Unwrap(cur)
			if next == nil {
				// Carrier with a nil cause shouldn't happen in normal use,
				// but be defensive: treat the last carrier as the node.
				return cur, classifications, nil
			}
			cur = next
			more, ok := errx.CarrierClassifications(cur)
			if !ok {
				break
			}
			classifications = append(classifications, more...)
		}
		node = cur
		nextCause = errors.Unwrap(node)
		return node, classifications, nextCause
	}

	// Wrap-shaped entry (or plain error): node is err, absorb at most one
	// immediate carrier behind it.
	node = cur
	nextCause = errors.Unwrap(node)
	if cls, ok := errx.CarrierClassifications(nextCause); ok {
		classifications = append(classifications, cls...)
		nextCause = errors.Unwrap(nextCause)
	}
	return node, classifications, nextCause
}

// sentinelsForLevel returns the pure-sentinel texts attributable to this
// level only. It examines the carrier classifications absorbed for this
// level and, if the node itself is a pure sentinel, includes it. Sentinels
// from deeper levels are intentionally not included here so they do not
// leak across error levels.
func sentinelsForLevel(classifications []errx.Classified, node error) []string {
	var sentinels []string
	// seen is allocated lazily once we are about to record a second sentinel;
	// in the common single-sentinel case it stays nil.
	var seen map[string]bool

	addPureSentinels(classifications, &sentinels, &seen)
	addSelfAsPureSentinel(node, &sentinels, &seen)

	return sentinels
}

// serializeAttributes extracts and serializes attributes from an error.
// Attribute values whose JSON encoding fails (e.g. a func() value) are
// replaced with a fmt.Sprintf("%+v", v) fallback string so a single bad
// value does not abort the entire marshal. If a non-nil
// AttributeValueTransformer is configured, it is applied to each value
// before the JSON-serializability check.
func serializeAttributes(err error, cfg *config, result *SerializedError) {
	attrs := errx.ExtractAttrs(err)
	if len(attrs) == 0 {
		return
	}
	result.Attributes = make([]SerializedAttr, len(attrs))
	for i, attr := range attrs {
		value := attr.Value
		if cfg.attributeValueTransformer != nil {
			value = cfg.attributeValueTransformer(attr.Key, value)
		}
		if !isJSONSerializable(value) {
			value = fmt.Sprintf("%+v", value)
		}
		result.Attributes[i] = SerializedAttr{
			Key:   attr.Key,
			Value: value,
		}
	}
}

// isJSONSerializable reports whether v can be encoded by encoding/json
// without producing an error. nil is treated as serializable (becomes
// JSON null).
func isJSONSerializable(v any) bool {
	if v == nil {
		return true
	}
	_, err := json.Marshal(v)
	return err == nil
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
			File:     trimStackPath(frames[i].File, cfg.stackTraceTrimPath),
			Line:     frames[i].Line,
			Function: frames[i].Function,
		}
	}
}

// unwrapper is the multi-error unwrap interface.
type unwrapper interface {
	Unwrap() []error
}

// serializeCauses handles unwrapping and serialization of error causes.
// nextCause is the already-peeled single cause from peelLevel (carriers
// stripped off where appropriate). Multi-error unwrap is handled on the
// node itself, not via nextCause.
func serializeCauses(node, nextCause error, cfg *config, visited *visitedSet, depth int, result *SerializedError) {
	// Multi-error path.
	if u, ok := node.(unwrapper); ok {
		serializeMultiError(u, cfg, visited, depth, result)
		return
	}

	// Single-cause path (carrier already peeled by peelLevel).
	if nextCause == nil {
		return
	}
	if !cfg.includeStandardErrors && !isErrxError(nextCause) {
		return
	}
	result.Cause = toSerializedError(nextCause, cfg, visited, depth+1)
}

// serializeMultiError serializes multiple error causes. Each branch gets a
// fresh copy of the visited set so a pointer that appears in a sibling
// branch is not mistakenly flagged as a cycle.
func serializeMultiError(u unwrapper, cfg *config, visited *visitedSet, depth int, result *SerializedError) {
	unwrapped := u.Unwrap()
	if len(unwrapped) == 0 {
		return
	}
	result.Causes = make([]*SerializedError, 0, len(unwrapped))
	for _, ue := range unwrapped {
		if ue == nil || (!cfg.includeStandardErrors && !isErrxError(ue)) {
			continue
		}
		branchVisited := copyVisited(*visited)
		serialized := toSerializedError(ue, cfg, &branchVisited, depth+1)
		if serialized != nil {
			result.Causes = append(result.Causes, serialized)
		}
	}
}

// copyVisited returns a shallow copy of the visited set for use as the
// independent path tracker of a new branch. A nil input returns a nil set
// so the branch can keep its lazy-allocation behavior.
func copyVisited(v visitedSet) visitedSet {
	if v == nil {
		return nil
	}
	cp := make(visitedSet, len(v))
	for k := range v {
		cp[k] = true
	}
	return cp
}

// rememberSentinel records text in the seen set, allocating it lazily on the
// first call. It returns true if text was already present.
func rememberSentinel(text string, seen *map[string]bool) bool {
	if *seen == nil {
		*seen = map[string]bool{text: true}
		return false
	}
	if (*seen)[text] {
		return true
	}
	(*seen)[text] = true
	return false
}

// addPureSentinels adds pure sentinel classifications to the sentinels list.
func addPureSentinels(classifications []errx.Classified, sentinels *[]string, seen *map[string]bool) {
	for _, cls := range classifications {
		if !isPureSentinel(cls) {
			continue
		}
		text := cls.Error()
		if rememberSentinel(text, seen) {
			continue
		}
		*sentinels = append(*sentinels, text)
	}
}

// isPureSentinel checks if a classified error is a pure sentinel.
func isPureSentinel(cls errx.Classified) bool {
	return !errx.IsDisplayable(cls) && !errx.HasAttrs(cls) && stacktrace.Extract(cls) == nil
}

// addSelfAsPureSentinel checks if the error itself is a pure sentinel and adds it.
func addSelfAsPureSentinel(err error, sentinels *[]string, seen *map[string]bool) {
	cls, ok := err.(errx.Classified)
	if !ok || !cls.IsClassified() {
		return
	}
	if !isPureSentinel(cls) {
		return
	}
	text := err.Error()
	if rememberSentinel(text, seen) {
		return
	}
	*sentinels = append(*sentinels, text)
}

// isErrxError checks if an error is an errx error (implements Classified).
func isErrxError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(errx.Classified)
	return ok
}
