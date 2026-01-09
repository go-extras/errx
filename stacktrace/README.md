# errx/stacktrace

Optional stack trace support for errx errors.

## Overview

The `stacktrace` package extends `errx` with stack trace capabilities while keeping the core `errx` package minimal and zero-dependency. It provides two usage patterns:

1. **Per-error opt-in** using `Here()` as a `Classified`
2. **Automatic capture** using `stacktrace.Wrap()` and `stacktrace.Classify()`

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
err := stacktrace.Classify(cause, ErrRetryable)
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

## Integration with errx Features

Stack traces work seamlessly with all errx features:

```go
var ErrNotFound = errx.NewSentinel("not found")

// Combine stack traces with displayable errors and attributes
displayErr := errx.NewDisplayable("User not found")
attrErr := errx.WithAttrs("user_id", "12345", "action", "fetch")

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

- `Here() errx.Classified` - Captures the current stack trace as a Classified
- `Extract(err error) []Frame` - Extracts stack frames from an error chain
- `Wrap(text string, cause error, classifications ...errx.Classified) error` - Wraps with automatic trace
- `Classify(cause error, classifications ...errx.Classified) error` - Classifies with automatic trace

### Types

- `Frame` - Represents a single stack frame with `File`, `Line`, and `Function` fields

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

