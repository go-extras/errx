# errx/stacktrace

Optional stack trace support for errx errors.

## Overview

The `stacktrace` package extends `errx` with stack trace capabilities while keeping the
core `errx` package minimal and zero-dependency. It provides three usage patterns:

1. **Per-error opt-in** using `Here()` as a `Classified`
2. **Automatic capture** using `stacktrace.Wrap()`, `stacktrace.Classify()` and
   `stacktrace.ClassifyNew()`
3. **Conditional capture** using `WrapIf`, `ClassifyIf` or `HereIf` to avoid duplicate traces
   when a cause already carries one.

## Installation

```bash
go get github.com/go-extras/errx/stacktrace@latest
```

## Usage

### Option 1: Per-Error Opt-In

Use `Here()` to capture stack traces only where needed:

```go
import (
    "github.com/go-extras/errx"
    "github.com/go-extras/errx/stacktrace"
)

var ErrNotFound = errx.NewSentinel("not found")

// Capture stack trace at this specific error site
err := errx.Wrap("operation failed", cause, ErrNotFound, stacktrace.Here())
```

### Option 2: Automatic Capture

Use `stacktrace.Wrap()` or `stacktrace.Classify()` for automatic trace capture:

```go
// Automatically captures stack trace
err := stacktrace.Wrap("operation failed", cause, ErrNotFound)

// Or with Classify
err = stacktrace.Classify(cause, ErrRetryable)

// Create and classify a new error
err = stacktrace.ClassifyNew("user record missing", ErrNotFound)
```

### Extracting Stack Traces

Extract and use stack traces from any error in the chain:

```go
frames := stacktrace.Extract(err)
if frames != nil {
    for _, frame := range frames {
        fmt.Printf("%s:%d %s\n", frame.File, frame.Line, frame.Function)
    }
}
```

`Extract` returns the first trace in the chain. Use `ExtractAll` when several layers or
branches may carry traces; it returns them outermost first.

### Conditional Capture

`WrapIf` and `ClassifyIf` capture a trace only when the cause does not already carry one.
Their `*Depth` variants let callers choose the maximum capture depth. `HereIf` and
`HereIfDepth` return `nil` when a trace is already present, so they can be passed to
`errx.Wrap`, `errx.Classify` or `errx.ClassifyNew` without adding a duplicate. Use
`HasTrace` to check for a non-empty trace before choosing your own behavior.

```go
err := stacktrace.WrapIf("load user", cause)
if stacktrace.HasTrace(err) {
    fmt.Println("trace is available")
}
```

### Formatting Traces

Errors returned by `stacktrace.Wrap`, `Classify` and `ClassifyNew` implement
`fmt.Formatter`. When an error carries a trace, `%+v` prints its message followed by the
stack frames in the `pkg/errors` style; `%v` and `%s` print only the message. `errx.Wrap`
and `errx.Classify` also format traces attached with `stacktrace.Here()`.

```go
fmt.Printf("%+v\n", err) // message followed by stack frames
```

## Integration with errx Features

Stack traces work seamlessly with all errx features:

```go
var ErrNotFound = errx.NewSentinel("not found")

// Combine stack traces with displayable errors and attributes
displayErr := errx.NewDisplayable("User not found")
attrErr := errx.Attrs("user_id", "12345", "action", "fetch")

err := stacktrace.Wrap("failed to get user profile",
    errx.Classify(displayErr, ErrNotFound, attrErr))

// All features work together
fmt.Println("Error:", err.Error())
fmt.Println("Displayable:", errx.DisplayText(err))
fmt.Println("Is not found:", errors.Is(err, ErrNotFound))
fmt.Println("Has attributes:", errx.HasAttrs(err))
fmt.Println("Has stack trace:", stacktrace.Extract(err) != nil)
```

## API

### Functions

- `Here() errx.Classified` captures a trace for use with `errx.Wrap`, `errx.Classify` or
  `errx.ClassifyNew`.
- `HereDepth(depth int) errx.Classified` captures up to the requested number of frames.
- `Wrap(text string, cause error, classifications ...errx.Classified) error` and
  `WrapDepth(text string, cause error, depth int, classifications ...errx.Classified) error`
  wrap an error and capture a trace.
- `Classify(cause error, classifications ...errx.Classified) error` and
  `ClassifyDepth(cause error, depth int, classifications ...errx.Classified) error`
  add classifications and capture a trace.
- `ClassifyNew(text string, classifications ...errx.Classified) error` and
  `ClassifyNewDepth(text string, depth int, classifications ...errx.Classified) error`
  create and classify an error with a trace.
- `WrapIf(text string, cause error, classifications ...errx.Classified) error` and
  `WrapIfDepth(text string, cause error, depth int, classifications ...errx.Classified) error`
  capture only when the cause has no trace.
- `ClassifyIf(cause error, classifications ...errx.Classified) error` and
  `ClassifyIfDepth(cause error, depth int, classifications ...errx.Classified) error`
  classify and capture only when needed.
- `HereIf(cause error) errx.Classified` and `HereIfDepth(cause error, depth int) errx.Classified`
  return `nil` when the cause already carries a trace.
- `Extract(err error) []Frame` returns the first trace in the error chain;
  `ExtractAll(err error) [][]Frame` returns all traces, outermost first.
- `HasTrace(err error) bool` reports whether the chain contains a non-empty trace.

### Constants

- `DefaultMaxDepth` is 32. Non-positive depth values use this default.
- `MaxDepth` is 256. Larger requested depths are clamped to this ceiling.

### Types

- `Frame` represents a stack frame with `File`, `Line` and `Function` fields.
- `Tracer` is implemented by errors that expose captured frames through `Frames() []Frame`.

## Performance Considerations

Stack trace capture has a small performance cost (~2-10µs per capture):
- Uses `runtime.Callers` to walk the stack
- Allocates a slice for program counters
- Frame resolution is done lazily on `Extract()`

**Recommendations:**
- Use per-error opt-in (`Here()`) in hot paths
- Use automatic capture (`stacktrace.Wrap()`) in application code
- Libraries should use core `errx`; applications add traces as needed

## Design Philosophy

This package follows Option 6 from the errx tracing design:

1. **Core stays minimal**: `errx` remains zero-dependency and fast
2. **Opt-in granularity**: Choose per-error or blanket trace capture
3. **Composable**: Traces are just another `Classified`, fitting existing patterns
4. **Library-friendly**: Libraries use `errx` core; applications add tracing where needed

## Examples

See the [package documentation](https://pkg.go.dev/github.com/go-extras/errx/stacktrace) for more examples.

## License

MIT License - see the [LICENSE](../LICENSE) file for details.

