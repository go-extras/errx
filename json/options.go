package json

import (
	"strings"

	"github.com/go-extras/errx/stacktrace"
)

// Option is a function that configures the JSON serialization behavior.
type Option func(*config)

// AttributeValueTransformer is a function that may rewrite (e.g. redact, hash,
// truncate) attribute values during serialization. It receives the attribute
// key and the original value, and returns the value to be serialized in its place.
//
// Returning the original value unchanged is a no-op. Returning nil is supported
// and produces a JSON null for that attribute's value.
type AttributeValueTransformer func(key string, v any) any

// StackFrameFilter reports whether a stack frame should be kept in the
// serialized output. It receives each frame and returns true to retain it,
// false to drop it. Used with [WithStackFrameFilter].
type StackFrameFilter func(stacktrace.Frame) bool

// WithMaxDepth sets the maximum depth for traversing error chains.
// This prevents issues with deeply nested or potentially circular error chains.
// The default is 32.
//
// A non-positive value (zero or negative) disables the depth limit and allows
// the serializer to traverse the chain to its end. This matches the convention
// used by WithMaxStackFrames.
//
// When a positive depth limit is reached, the serialized error will have a
// message of "(max depth reached)" and no further unwrapping will occur.
// Circular references are still detected and reported as "(circular reference)"
// regardless of the depth setting.
//
// Example:
//
//	jsonBytes, err := json.Marshal(err, json.WithMaxDepth(10))
//
//	// Disable the depth limit:
//	jsonBytes, err := json.Marshal(err, json.WithMaxDepth(0))
func WithMaxDepth(depth int) Option {
	return func(c *config) {
		c.maxDepth = depth
	}
}

// WithMaxStackFrames sets the maximum number of stack frames to include
// in the serialized output. This prevents excessive JSON size when errors
// have deep stack traces. The default is 32.
//
// If the stack trace has more frames than the limit, only the first N frames
// will be included in the serialized output.
//
// Example:
//
//	jsonBytes, err := json.Marshal(err, json.WithMaxStackFrames(10))
func WithMaxStackFrames(frames int) Option {
	return func(c *config) {
		c.maxStackFrames = frames
	}
}

// WithIncludeStandardErrors controls whether standard (non-errx) errors
// in the error chain are included in the serialized output.
// The default is true.
//
// When set to false, only errx errors (those implementing errx.Classified)
// will be serialized in the cause chain. Standard errors will be skipped.
//
// Example:
//
//	// Only include errx errors, skip standard errors
//	jsonBytes, err := json.Marshal(err, json.WithIncludeStandardErrors(false))
func WithIncludeStandardErrors(include bool) Option {
	return func(c *config) {
		c.includeStandardErrors = include
	}
}

// WithStackTrace toggles stack-trace serialization entirely. The default is true
// (stack traces are included when present on the error).
//
// This differs from [WithMaxStackFrames], where 0 means "unlimited" rather than
// "off". Use WithStackTrace(false) when you want to fully suppress stack traces
// from the JSON output — for example, in production log pipelines where stack
// traces are emitted separately or considered sensitive.
//
// Example:
//
//	jsonBytes, err := json.Marshal(err, json.WithStackTrace(false))
func WithStackTrace(include bool) Option {
	return func(c *config) {
		c.includeStackTrace = include
	}
}

// WithAttributes toggles attribute serialization entirely. The default is true.
//
// Use WithAttributes(false) for log pipelines that should never see structured
// metadata (e.g. PII-sensitive paths where attributes may contain user data).
// Setting this to false suppresses the "attributes" field in the serialized output.
//
// Example:
//
//	jsonBytes, err := json.Marshal(err, json.WithAttributes(false))
func WithAttributes(include bool) Option {
	return func(c *config) {
		c.includeAttributes = include
	}
}

// WithSentinels toggles sentinel-text serialization. The default is true.
//
// Use WithSentinels(false) when the sentinel text is considered internal
// implementation detail that should not appear in serialized output. Suppresses
// the "sentinels" field.
//
// Example:
//
//	jsonBytes, err := json.Marshal(err, json.WithSentinels(false))
func WithSentinels(include bool) Option {
	return func(c *config) {
		c.includeSentinels = include
	}
}

// WithAttributeValueTransformer registers a function that is called once per
// attribute value during serialization. The function may rewrite or redact the
// value (e.g. mask credit card numbers, hash user IDs, truncate long strings).
//
// The transformer is applied AFTER attribute inclusion is determined; if
// WithAttributes(false) was set, the transformer is not called.
//
// Example:
//
//	redact := func(key string, v any) any {
//	    if key == "password" || key == "token" {
//	        return "<redacted>"
//	    }
//	    return v
//	}
//	jsonBytes, err := json.Marshal(err, json.WithAttributeValueTransformer(redact))
func WithAttributeValueTransformer(fn AttributeValueTransformer) Option {
	return func(c *config) {
		c.attributeValueTransformer = fn
	}
}

// WithStackTraceTrimPaths strips the given prefix from each stack frame's File
// field during serialization. This is useful for compressing JSON output and
// removing build-host-specific paths (e.g. /home/runner/work/...) from logs.
//
// The trim is a simple prefix match: if the frame's File starts with prefix,
// the prefix is removed. Otherwise the path is left unchanged. Pass an empty
// string to disable trimming (the default).
//
// Example:
//
//	jsonBytes, err := json.Marshal(err,
//	    json.WithStackTraceTrimPaths("/home/runner/work/myproject/"))
func WithStackTraceTrimPaths(prefix string) Option {
	return func(c *config) {
		c.stackTraceTrimPath = prefix
	}
}

// WithStackTraceTrimTop drops the top n frames — the innermost frames, closest
// to where the trace was captured — from each serialized stack trace. This
// removes framework/runtime noise that sits above the meaningful application
// frames (the motivation behind pkg/errors#111 and pkg/errors#129).
//
// Trimming happens before [WithStackFrameFilter] and before the
// [WithMaxStackFrames] cap, so the cap counts only the frames that survive
// trimming. If n is greater than or equal to the number of frames, the stack
// trace is omitted entirely. A value of 0 or negative (the default) keeps all
// frames.
//
// Example:
//
//	jsonBytes, err := json.Marshal(err, json.WithStackTraceTrimTop(2))
func WithStackTraceTrimTop(n int) Option {
	return func(c *config) {
		c.stackTraceTrimTop = n
	}
}

// WithStackFrameFilter installs a predicate that decides, per frame, whether a
// stack frame is kept in the serialized output. The predicate returns true to
// keep the frame and false to drop it. This is useful for stripping
// framework/runtime frames (for example those in "runtime." or "net/http.")
// from rendered traces.
//
// The filter runs after [WithStackTraceTrimTop] and before the
// [WithMaxStackFrames] cap, so the cap counts only frames that pass the filter.
// If filtering removes every frame, the stack trace is omitted entirely.
// Passing a nil filter (the default) keeps all frames.
//
// Example:
//
//	jsonBytes, err := json.Marshal(err, json.WithStackFrameFilter(
//	    func(f stacktrace.Frame) bool {
//	        return !strings.HasPrefix(f.Function, "runtime.")
//	    }))
func WithStackFrameFilter(keep StackFrameFilter) Option {
	return func(c *config) {
		c.stackFrameFilter = keep
	}
}

// WithMaxMessageBytes truncates the Message field of each serialized error to
// at most n bytes. If the message is longer, it is truncated and the suffix
// "...(truncated)" is appended. A value of 0 (the default) means no truncation.
//
// Truncation is applied byte-wise on the UTF-8 representation; the implementation
// is careful not to split multi-byte runes — it backs off to the previous rune
// boundary before appending the suffix.
//
// This is useful for ultra-long messages (e.g. SQL queries embedded in error text)
// that would otherwise dominate log volume.
//
// Example:
//
//	jsonBytes, err := json.Marshal(err, json.WithMaxMessageBytes(256))
func WithMaxMessageBytes(n int) Option {
	return func(c *config) {
		c.maxMessageBytes = n
	}
}

// truncationSuffix is appended when a message is truncated by WithMaxMessageBytes.
const truncationSuffix = "...(truncated)"

// truncateMessage truncates s to at most maxBytes bytes (when maxBytes > 0),
// appending [truncationSuffix] when truncation occurs. It backs off to the
// previous rune boundary to avoid splitting multi-byte runes.
//
// When maxBytes is too small to also fit the truncation suffix, the function
// returns a hard cut at maxBytes (no suffix) to honor the byte limit; the cut
// is still aligned to a valid UTF-8 rune boundary.
func truncateMessage(s string, maxBytes int) string {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s
	}

	// If the suffix alone wouldn't fit, hard-cut to a rune boundary.
	if maxBytes <= len(truncationSuffix) {
		cut := maxBytes
		for cut > 0 && cut < len(s) && isUTF8Cont(s[cut]) {
			cut--
		}
		return s[:cut]
	}

	cut := maxBytes - len(truncationSuffix)
	// Back off to the previous rune boundary to avoid splitting a multi-byte rune.
	for cut > 0 && cut < len(s) && isUTF8Cont(s[cut]) {
		cut--
	}
	return s[:cut] + truncationSuffix
}

// isUTF8Cont reports whether b is a UTF-8 continuation byte (10xxxxxx).
func isUTF8Cont(b byte) bool { return b&0xC0 == 0x80 }

// trimStackPath applies a configured prefix trim to a stack-frame File path.
// Returns the path unchanged when prefix is empty or does not match.
func trimStackPath(file, prefix string) string {
	if prefix == "" {
		return file
	}
	return strings.TrimPrefix(file, prefix)
}
