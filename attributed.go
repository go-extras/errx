package errx

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// Attr represents a key-value pair for structured error context.
type Attr struct {
	Key   string
	Value any
}

type AttrMap = map[string]any

// String returns a string representation of the Attr.
func (a Attr) String() string {
	return fmt.Sprintf("%s=%+v", a.Key, a.Value)
}

// Attrs is a slice of Attr structs.
type Attrs []Attr

// String returns a string representation of the Attrs slice.
func (al Attrs) String() string {
	parts := make([]string, 0, len(al))
	for _, attr := range al {
		parts = append(parts, attr.String())
	}
	return strings.Join(parts, " ")
}

// ToSlogAttrs converts errx.Attrs to []slog.Attr for use with structured logging.
// This enables seamless integration with slog and slog-compatible loggers.
//
// Example:
//
//	err := errx.WithAttrs("user_id", 123, "action", "delete")
//	attrs := errx.ExtractAttrs(err)
//	slogAttrs := attrs.ToSlogAttrs()
//	slog.Error("operation failed", slogAttrs...)
func (al Attrs) ToSlogAttrs() []slog.Attr {
	if len(al) == 0 {
		return nil
	}

	result := make([]slog.Attr, len(al))
	for i, attr := range al {
		result[i] = slog.Any(attr.Key, attr.Value)
	}
	return result
}

// WithAttrs creates an error with structured attributes for additional context.
// Attributes can be extracted later using ExtractAttrs.
//
// # Recommended Usage
//
// WithAttrs is typically used in combination with Wrap or Classify to create rich errors
// with both meaningful error messages and structured metadata:
//
//	// RECOMMENDED: Combine with Wrap for context + attributes
//	attrErr := errx.WithAttrs("user_id", 123, "action", "delete")
//	return errx.Wrap("failed to delete user", baseErr, attrErr)
//
//	// RECOMMENDED: Combine with Classify for classification + attributes
//	attrErr := errx.WithAttrs("retry_count", 3)
//	return errx.Classify(baseErr, ErrRetryable, attrErr)
//
// Using WithAttrs alone produces a less informative error message that only shows
// the attribute list. For better error messages, always combine it with Wrap or Classify.
//
// # Input Formats
//
// WithAttrs accepts multiple input formats:
//   - Key-value pairs: WithAttrs("key1", value1, "key2", value2)
//   - Attr structs: WithAttrs(Attr{Key: "key1", Value: value1}, Attr{Key: "key2", Value: value2})
//   - Attr slices: WithAttrs([]Attr{{Key: "key1", Value: value1}, {Key: "key2", Value: value2}})
//   - Mixed formats: WithAttrs("key1", value1, Attr{Key: "key2", Value: value2})
//
// The arguments are processed following a structured pattern (similar to slog):
//   - If an argument is an Attr, it is used as is.
//   - If an argument is an []Attr or Attrs, all attributes are appended.
//   - If an argument is a string and this is not the last argument,
//     the following argument is treated as the value and the two are combined into an Attr.
//   - Otherwise, the argument is treated as a value with key "!BADKEY".
//
// The "!BADKEY" key is used for malformed arguments to help identify issues during debugging.
// This behavior matches the slog package's handling of malformed key-value pairs.
//
// Examples:
//
//	WithAttrs("key", "value")                    // Normal key-value pair
//	WithAttrs("key")                             // Odd number: Attr{Key: "!BADKEY", Value: "key"}
//	WithAttrs(123)                               // Non-string: Attr{Key: "!BADKEY", Value: 123}
//	WithAttrs("key", 123)                        // String key with int value: Attr{Key: "key", Value: 123}
//	WithAttrs(Attr{Key: "k", Value: "v"})        // Direct Attr usage
//	WithAttrs([]Attr{{Key: "k", Value: "v"}})    // Slice of Attrs
func WithAttrs(attrs ...any) Classified {
	parsedAttrs := parseAttrs(attrs)
	return &attributed{
		attrs: parsedAttrs,
	}
}

// parseAttrs converts various input formats into a slice of Attr.
// The arguments are processed as follows:
//   - If an argument is an Attr, it is used as is.
//   - If an argument is an []Attr, all attributes are appended.
//   - If an argument is a string and this is not the last argument,
//     the following argument is treated as the value and the two are combined
//     into an Attr.
//   - Otherwise, the argument is treated as a value with key "!BADKEY".
func parseAttrs(attrs []any) []Attr {
	if len(attrs) == 0 {
		return nil
	}

	result := make([]Attr, 0, len(attrs))

	for i := 0; i < len(attrs); i++ {
		switch v := attrs[i].(type) {
		case Attr:
			// Attr struct is used as-is
			result = append(result, v)
		case []Attr:
			// Slice of Attr structs - all appended
			result = append(result, v...)
		case Attrs:
			// Slice of Attr structs - all appended
			result = append(result, v...)
		case string:
			// String key: if there's a next argument, treat it as value
			if i+1 < len(attrs) {
				result = append(result, Attr{Key: v, Value: attrs[i+1]})
				i++ // Skip the next element as it's the value
			} else {
				// String at the end with no value - use !BADKEY pattern
				result = append(result, Attr{Key: "!BADKEY", Value: v})
			}
		default:
			// Any other type is treated as a value with !BADKEY
			result = append(result, Attr{Key: "!BADKEY", Value: v})
		}
	}

	return result
}

// FromAttrMap creates an error from a map of attributes.
// This is a convenience function for creating attributed errors from existing maps.
//
// # Order Non-Determinism
//
// WARNING: The order of attributes in the resulting error is non-deterministic because
// Go map iteration order is randomized. If you need deterministic ordering, use WithAttrs
// with a slice of Attr instead:
//
//	// Non-deterministic order
//	err := errx.FromAttrMap(map[string]any{"key1": "val1", "key2": "val2"})
//
//	// Deterministic order
//	err := errx.WithAttrs(
//	    errx.Attr{Key: "key1", Value: "val1"},
//	    errx.Attr{Key: "key2", Value: "val2"},
//	)
func FromAttrMap(attrMap AttrMap) Classified {
	attrs := make([]Attr, 0, len(attrMap))
	for k, v := range attrMap {
		attrs = append(attrs, Attr{Key: k, Value: v})
	}
	return WithAttrs(attrs)
}

type attributed struct {
	attrs []Attr
}

func (ae *attributed) Error() string {
	if len(ae.attrs) == 0 {
		return "(empty attribute list)"
	}

	return Attrs(ae.attrs).String()
}

// Attrs returns the structured attributes associated with this error.
func (ae *attributed) Attrs() []Attr {
	return ae.attrs
}

// IsClassified implements the Classified interface marker method.
// It always returns true to identify this as a Classified error.
func (*attributed) IsClassified() bool {
	return true
}

// HasAttrs checks if an error contains structured attributes.
// It returns true if the error or any wrapped error is an attributed error.
func HasAttrs(err error) bool {
	if err == nil {
		return false
	}

	var aErr *attributed
	return errors.As(err, &aErr)
}

// ExtractAttrs extracts and merges all structured attributes from an error chain.
// It traverses the entire error chain and collects attributes from all attributed instances.
//
// The order of attributes in the result is stable for a given error graph, but this
// ordering is not a semantic guarantee. Callers should not rely on attribute ordering
// for precedence or any other logic. If you need a map with specific merge semantics,
// consider converting the result to a map with your own collision-handling rules.
//
// Returns nil if the error is nil or does not contain any attributes.
func ExtractAttrs(err error) Attrs {
	if err == nil {
		return nil
	}

	var allAttrs []Attr
	visited := make(map[error]bool)
	attributedErrorsFound := make(map[*attributed]bool)

	// Use a queue for breadth-first traversal to handle multi-errors
	queue := []error{err}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Skip if already visited (avoid cycles)
		if visited[current] {
			continue
		}
		visited[current] = true

		// Check if current error is an attributed error directly
		if aErr, ok := current.(*attributed); ok {
			if !attributedErrorsFound[aErr] {
				attributedErrorsFound[aErr] = true
				allAttrs = append(allAttrs, aErr.attrs...)
			}
		}

		// If this is a carrier with classifications, add them to the queue
		// This ensures we traverse all attached attributed errors
		if c, ok := current.(*carrier); ok {
			for _, cls := range c.classifications {
				queue = append(queue, cls)
			}
		}

		// Continue traversing the unwrap chain
		// Handle multi-error case (Go 1.20+)
		type unwrapper interface {
			Unwrap() []error
		}
		if u, ok := current.(unwrapper); ok {
			queue = append(queue, u.Unwrap()...)
		} else if next := errors.Unwrap(current); next != nil {
			queue = append(queue, next)
		}
	}

	if len(allAttrs) == 0 {
		return nil
	}

	return allAttrs
}
