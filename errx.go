// Package errx provides rich error handling utilities with classification sentinels,
// displayable messages, and structured attributes.
//
// It enables wrapping errors with classification sentinels that can be checked using errors.Is,
// attaching user-facing displayable messages, and adding structured metadata for logging.
//
// # Core Concepts
//
// The package provides three main error categories:
//
// Classification Sentinels: For programmatic error checking using errors.Is. These sentinels
// are used to identify specific error conditions in code, such as "not found" or
// "access denied". The sentinel text is intentionally NOT visible in the error
// message chain to keep error messages clean.
//
// Displayable Errors: For user-facing error messages. These errors represent messages that
// are safe and appropriate to display directly to end users. They can be extracted
// from any error chain using DisplayText, which returns just the displayable message
// without internal context.
//
// Attributed Errors: For attaching structured metadata (key-value pairs) to errors for
// logging and debugging. These attributes can be created using Attrs and extracted
// from any error chain using ExtractAttrs, enabling rich contextual information without
// cluttering error messages.
//
// # Subpackages
//
// The core package remains zero-dependency. Optional subpackages build on it:
//   - stacktrace: capture and extract stack traces for errx errors
//   - json: serialize errx errors and their metadata to JSON
//   - compat: work with the standard error interface while still using errx classifications
//
// # When to Use
//
// Use Classification sentinels (NewSentinel) when:
//   - You need to check for specific error conditions programmatically
//   - The error type is more important than the error message
//   - You want to attach error classifications without polluting error messages
//
// Use Displayable errors (NewDisplayable) when:
//   - You need to return user-friendly error messages from APIs
//   - The error should be safe to display to end users
//   - You want to separate internal error context from user messages
//
// Use Attributed errors (Attrs) when:
//   - You need to attach structured metadata for logging and debugging
//   - You want to add contextual information like user IDs, request IDs, or operation details
//   - You're building errors that will be logged with structured logging systems
//
// Use Wrap when:
//   - You need to add context to an error
//   - You want to attach classification sentinels to existing errors
//   - You're propagating errors up the call stack
//
// Use Classify when:
//   - You want to attach classification sentinels WITHOUT adding context text
//   - You need to mark an error for programmatic checking but keep the original message
//   - You're at a layer where the error message is already sufficient
//
// # Example Usage
//
//	// Define classification sentinels
//	var ErrNotFound = errx.NewSentinel("resource not found")
//
//	// Create displayable error
//	func validateInput(email string) error {
//	    if !isValid(email) {
//	        return errx.NewDisplayable("Invalid email format")
//	    }
//	    return nil
//	}
//
//	// Wrap with context and sentinels
//	func fetchUser(id string) error {
//	    displayErr := errx.NewDisplayable("User not found")
//	    return errx.Wrap("failed to fetch user", displayErr, ErrNotFound)
//	}
//
//	// Classify without adding context
//	func processRecord(err error) error {
//	    return errx.Classify(err, ErrNotFound)  // Preserves original message
//	}
//
//	// Add structured attributes for logging
//	func deleteUser(userID int) error {
//	    attrErr := errx.Attrs("user_id", userID, "action", "delete")
//	    return errx.Wrap("failed to delete user", baseErr, attrErr)
//	}
//
//	// Check for specific errors
//	if errors.Is(err, ErrNotFound) {
//	    // Handle not found case
//	}
//
//	// Extract displayable message
//	if errx.IsDisplayable(err) {
//	    return errx.DisplayText(err)  // Returns: "User not found"
//	}
//
//	// Extract structured attributes for logging
//	if errx.HasAttrs(err) {
//	    attrs := errx.ExtractAttrs(err)  // Returns: [{Key: "user_id", Value: 123}, ...]
//	}
package errx

import (
	"errors"
	"fmt"
	"io"

	"github.com/go-extras/errx/internal/errptr"
)

// isNilClassified reports whether the given Classified is nil in any form:
// either an untyped-nil interface, or a typed-nil interface (e.g. a
// `(*sentinel)(nil)` stored in a Classified). The latter case slips past a
// plain `c != nil` check because the interface still has a non-nil type
// pointer. We delegate to errptr.Get, which returns 0 for both forms by
// inspecting the interface header — keeping all unsafe-pointer code in the
// single internal/errptr package rather than duplicating it here.
func isNilClassified(c Classified) bool {
	return errptr.Get(c) == 0
}

// nonNilClassifications returns a slice containing only the non-nil entries
// of cs (filtering out both untyped-nil and typed-nil interface values, which
// would otherwise panic when their methods dereference receiver state).
//
// Returns the input slice unchanged (with zero allocations) when no nil
// entries are present, which is the common case. Otherwise allocates a new
// slice and copies the surviving entries into it.
func nonNilClassifications(cs []Classified) []Classified {
	// Fast path: scan once; if every entry is non-nil, return cs as-is.
	nilIdx := -1
	for i, c := range cs {
		if isNilClassified(c) {
			nilIdx = i
			break
		}
	}
	if nilIdx == -1 {
		return cs
	}
	out := make([]Classified, 0, len(cs)-1)
	out = append(out, cs[:nilIdx]...)
	for _, c := range cs[nilIdx+1:] {
		if !isNilClassified(c) {
			out = append(out, c)
		}
	}
	return out
}

// Classified is an interface for errors that can be classified.
// This interface can be implemented by external packages to extend the library.
// Internally, there are four categories of Classified implementations:
//
//  1. Sentinel errors (*sentinel): Pure markers for programmatic error
//     checking using errors.Is.
//
//  2. Displayable errors (*displayable): Errors with messages safe to display to
//     end users.
//
//  3. Attributed errors (*attributed): Errors that carry structured metadata (key-value pairs)
//     for logging and debugging.
//
//  4. Traced errors (stacktrace.*traced): Errors that capture stack traces (in stacktrace subpackage).
//
// The IsClassified() method serves as a type marker to distinguish Classified errors
// from regular Go errors. All implementations should return true.
type Classified interface {
	error
	// IsClassified is a marker method that identifies this error as a Classified error.
	// It should always return true for valid Classified implementations.
	// This method allows programmatic distinction between regular errors and errx Classified errors.
	IsClassified() bool
}

// Ensure sentinel implements Classified interface
var _ Classified = (*sentinel)(nil)

type sentinel struct {
	text    string
	parents []Classified
}

func (s *sentinel) Error() string {
	return s.text
}

func (s *sentinel) Unwrap() error {
	if len(s.parents) == 0 {
		return nil
	}
	// Return first parent for standard unwrapping
	return s.parents[0]
}

func (s *sentinel) Is(target error) bool {
	// Check if target is this sentinel
	if target == s {
		return true
	}

	// Check if target matches any parent
	for _, parent := range s.parents {
		if errors.Is(parent, target) {
			return true
		}
	}

	return false
}

// As checks if the target matches any parent errors.
func (s *sentinel) As(target any) bool {
	// Check parents via errors.As
	for _, parent := range s.parents {
		if errors.As(parent, target) {
			return true
		}
	}
	return false
}

// IsClassified implements the Classified interface marker method.
// It always returns true to identify this as a Classified error.
func (*sentinel) IsClassified() bool {
	return true
}

// NewSentinel creates a new classification sentinel with the given text.
// Classification sentinels are used for programmatic error checking with errors.Is.
// The sentinel text is intentionally not visible in error message chains.
//
// Optional parent sentinels can be provided to create a hierarchy. A sentinel with parents
// will match itself and all of its parents via errors.Is.
//
// # Circular References
//
// WARNING: Creating circular parent references will cause infinite loops when using errors.Is.
// It is the caller's responsibility to avoid circular hierarchies. For example:
//
//	// DON'T DO THIS - creates a circular reference
//	parent := errx.NewSentinel("parent")
//	child := errx.NewSentinel("child", parent)
//	// Then somehow making parent reference child would create a cycle
//
// The package does not detect or prevent circular references for performance reasons.
// Always ensure your sentinel hierarchies form a directed acyclic graph (DAG).
//
// Example:
//
//	// Simple sentinel
//	ErrDatabase := errx.NewSentinel("database error")
//
//	// Sentinel with parent (hierarchical)
//	ErrTimeout := errx.NewSentinel("timeout", ErrDatabase)
//	// Now ErrTimeout will match both itself and ErrDatabase
//
//	// Sentinel with multiple parents
//	ErrCritical := errx.NewSentinel("critical")
//	ErrDatabaseCritical := errx.NewSentinel("critical database error", ErrDatabase, ErrCritical)
//	// Matches itself, ErrDatabase, and ErrCritical
func NewSentinel(text string, parents ...Classified) Classified {
	parents = nonNilClassifications(parents)
	if len(parents) == 0 {
		return &sentinel{text: text}
	}
	return &sentinel{text: text, parents: parents}
}

// Wrap wraps an error with additional context text and optional classification sentinels.
// The attached classification sentinels can be used later to identify the error using errors.Is,
// as well as add displayable errors.
// If err is nil, Wrap returns nil.
//
// If no classifications are provided, Wrap behaves like fmt.Errorf with %w,
// avoiding unnecessary carrier allocation.
//
// When one or more classifications are attached, the returned error implements
// fmt.Formatter: fmt.Sprintf("%+v", err) prints the message followed by any
// classification that itself renders under "%+v" (most notably a stack trace
// captured via the stacktrace subpackage), in the de-facto pkg/errors style.
// "%v" and "%s" still print the message only. Unwrapping the result once yields
// the underlying classification carrier, so errors.Is/errors.As and
// CarrierClassifications behave exactly as before.
func Wrap(text string, cause error, classifications ...Classified) error {
	if cause == nil {
		return nil
	}
	classifications = nonNilClassifications(classifications)
	if len(classifications) == 0 {
		return fmt.Errorf("%s: %w", text, cause)
	}
	// Mirror the (text, carrier) shape that fmt.Errorf("%s: %w", text, carrier)
	// used to build, but with our own wrapper type so the result can implement
	// fmt.Formatter (fmt.wrapError does not). The classifications are already
	// filtered above, so the carrier is constructed directly rather than via
	// classify (which would re-filter).
	return &wrapped{text: text, cause: &carrier{classifications: classifications, cause: cause}}
}

// Classify attaches one or more classification sentinels to an existing error.
// The attached classification sentinels can be used later to identify the error using errors.Is.
// If err is nil, Classify returns nil.
//
// Example:
//
//	var ErrNotFound = errx.NewSentinel("resource not found")
//
//	baseErr := errors.New("resource missing")
//	classifiedErr := errx.Classify(baseErr, ErrNotFound)
//
//	fmt.Println(errors.Is(classifiedErr, ErrNotFound)) // Output: true
func Classify(cause error, classifications ...Classified) error {
	return classify(cause, classifications...)
}

// ClassifyNew creates a new error with the given text and immediately classifies it
// with one or more classification sentinels. This is a convenience function equivalent
// to calling errx.Classify(errors.New(text), classifications...).
//
// This function is useful when you want to create a new error and classify it in a
// single step, reducing verbosity compared to the two-step approach.
//
// Example:
//
//	var ErrNotFound = errx.NewSentinel("resource not found")
//	var ErrDatabase = errx.NewSentinel("database error")
//
//	// Instead of:
//	// err := errx.Classify(errors.New("user record missing"), ErrNotFound, ErrDatabase)
//
//	// You can write:
//	err := errx.ClassifyNew("user record missing", ErrNotFound, ErrDatabase)
//
//	fmt.Println(err.Error())                        // Output: user record missing
//	fmt.Println(errors.Is(err, ErrNotFound))        // Output: true
//	fmt.Println(errors.Is(err, ErrDatabase))        // Output: true
func ClassifyNew(text string, classifications ...Classified) error {
	return classify(simpleError(text), classifications...)
}

func classify(cause error, classifications ...Classified) error {
	if cause == nil {
		return nil
	}
	classifications = nonNilClassifications(classifications)
	// Symmetric short-circuit with Wrap: avoid wrapping in a no-op carrier
	// when there are no classifications to attach. The cause is returned
	// unchanged so errors.Is(returned, cause) holds via identity.
	if len(classifications) == 0 {
		return cause
	}
	return &carrier{classifications: classifications, cause: cause}
}

type carrier struct {
	classifications []Classified
	cause           error
}

func (c *carrier) Error() string {
	// IMPORTANT: classification sentinel text is intentionally NOT shown here
	return c.cause.Error()
}

func (c *carrier) Unwrap() error {
	return c.cause
}

func (c *carrier) Is(target error) bool {
	if errors.Is(c.cause, target) {
		return true
	}

	for _, cls := range c.classifications {
		if errors.Is(cls, target) {
			return true
		}
	}

	return false
}

func (c *carrier) As(target any) bool {
	if errors.As(c.cause, target) {
		return true
	}

	for _, cls := range c.classifications {
		if errors.As(cls, target) {
			return true
		}
	}

	return false
}

// Format implements fmt.Formatter so that carriers produced by Classify and
// ClassifyNew take part in pkg/errors-style "%+v" rendering. Under "%+v" the
// error message is written first, then every classification that is itself a
// fmt.Formatter is appended — most notably a stack trace captured by the
// stacktrace subpackage, whose *traced type writes its frames. "%v" and "%s"
// render the message only (unchanged), "%q" renders the quoted message, and
// unknown verbs produce a stdlib-style "%!<verb>(errx.carrier=...)" marker.
//
// Because sentinel text is intentionally hidden from the message chain (see
// Error), plain sentinels contribute nothing here; only Formatter
// classifications surface, and only under the "+" flag.
//
// Width and precision flags (e.g. "%10v", "%.3s") are not honored, matching the
// stacktrace subpackage's traced.Format and github.com/pkg/errors; the message
// is written verbatim.
func (c *carrier) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = io.WriteString(s, c.Error())
			formatClassifications(s, verb, c.classifications)
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, c.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", c.Error())
	default:
		_, _ = fmt.Fprintf(s, "%%!%c(errx.carrier=%s)", verb, c.Error())
	}
}

// formatClassifications writes, for the given verb, the rendering of every
// classification in cs that implements fmt.Formatter. It is the shared "%+v"
// tail used by carrier.Format and wrapped.Format: after the message has been
// written, captured stack traces (the stacktrace subpackage's *traced type is a
// fmt.Formatter) append their frames here. Classifications that are not
// Formatters — plain sentinels, displayable, and attributed errors — render
// nothing, keeping them invisible in formatted output exactly as they are under
// "%v" and "%s".
func formatClassifications(s fmt.State, verb rune, cs []Classified) {
	for _, cls := range cs {
		if f, ok := cls.(fmt.Formatter); ok {
			f.Format(s, verb)
		}
	}
}

// wrapped is the error returned by Wrap when one or more classifications are
// attached. It pairs the wrap context text with the carrier that holds the
// classifications and the underlying cause.
//
// It exists so the wrap layer can implement fmt.Formatter; the standard library's
// fmt.wrapError (produced by the fmt.Errorf("%s: %w", …) form Wrap used before)
// does not, which left "%+v" unable to reach a captured stack trace. Error and
// Unwrap deliberately match that former wrapper byte-for-byte: Error is
// "text: cause" and Unwrap exposes the carrier (so a single Unwrap still yields a
// Classified error and CarrierClassifications reports false for the wrapper
// itself, both relied upon by callers and the json subpackage).
type wrapped struct {
	text  string
	cause *carrier
}

func (w *wrapped) Error() string {
	return w.text + ": " + w.cause.Error()
}

func (w *wrapped) Unwrap() error {
	return w.cause
}

// Format implements fmt.Formatter for the wrap layer. It mirrors carrier.Format:
// "%+v" prints the "text: cause" message and then appends the carrier's
// Formatter classifications (e.g. a captured stack trace); "%v"/"%s" print the
// message only, "%q" the quoted message, and unknown verbs a stdlib-style
// marker. The message is written from Error() rather than by delegating to the
// carrier so the cause text is not duplicated.
func (w *wrapped) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = io.WriteString(s, w.Error())
			formatClassifications(s, verb, w.cause.classifications)
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, w.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", w.Error())
	default:
		_, _ = fmt.Fprintf(s, "%%!%c(errx.wrapped=%s)", verb, w.Error())
	}
}

// simpleError is a simple error type that just holds a text message.
// It's used internally by ClassifyNew to create a basic error.
type simpleError string

func (e simpleError) Error() string {
	return string(e)
}

// CarrierClassifications returns the classification sentinels attached to err
// if err is an internal carrier produced by Classify, ClassifyNew, or Wrap
// (when called with one or more classifications).
//
// The second return value reports whether err is a carrier. It returns
// (nil, false) when err is nil or is not a carrier.
//
// This helper is primarily intended for subpackages (such as the json
// subpackage) that need to introspect the classifications attached at a
// specific level of the error chain without traversing it via errors.Is.
// It exists to avoid reflection or unsafe-pointer tricks against unexported
// carrier internals.
//
// The returned slice aliases the carrier's internal storage; callers must
// treat it as read-only.
func CarrierClassifications(err error) ([]Classified, bool) {
	if err == nil {
		return nil, false
	}
	c, ok := err.(*carrier)
	if !ok {
		return nil, false
	}
	return c.classifications, true
}
