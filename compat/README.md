# compat - Standard Error Interface Compatibility

The `compat` package provides compatibility functions that accept standard Go `error` interface instead of requiring `errx.Classified` types. This package is designed for users who prefer working with the standard error interface while still benefiting from errx's classification and wrapping capabilities.

## Why This Package Exists

The parent `errx` package uses the `Classified` interface for several important reasons:

1. **Type Safety**: Ensures only valid classification types can be attached to errors
2. **Sealed Interface Pattern**: Maintains controlled extensibility through the `IsClassified()` marker method
3. **API Stability**: Allows internal evolution without breaking existing code
4. **Clear Intent**: Makes it explicit that you're attaching metadata rather than wrapping arbitrary errors

However, some users prefer the flexibility of working with standard Go `error` interface. The `compat` package bridges this gap.

## Quick Start

```go
import (
    "errors"
    "github.com/go-extras/errx/compat"
)

// Define classification errors (can be any error type)
var (
    ErrNotFound   = errors.New("not found")
    ErrDatabase   = errors.New("database error")
    ErrValidation = errors.New("validation error")
)

// Use compat functions with standard errors
func fetchUser(id string) error {
    err := db.Query(id)
    if err != nil {
        return compat.Wrap("failed to fetch user", err, ErrNotFound, ErrDatabase)
    }
    return nil
}

// Check classifications using standard errors.Is
if errors.Is(err, ErrNotFound) {
    // Handle not found case
}
```

## API

### `compat.Wrap(text string, cause error, classifications ...error) error`

Wraps an error with additional context text and optional classifications. Accepts standard Go `error` interface for classifications.

```go
err := db.Query(id)
return compat.Wrap("failed to fetch user", err, ErrNotFound, ErrDatabase)
```

### `compat.Classify(cause error, classifications ...error) error`

Attaches classifications to an error without adding context text. Preserves the original error message.

```go
err := validateEmail(email)
return compat.Classify(err, ErrValidation)
```

## Mixing with errx Types

You can freely mix standard errors with `errx.Classified` types:

```go
displayable := errx.NewDisplayable("User not found")
attrErr := errx.WithAttrs("user_id", 123)

err := compat.Wrap("lookup failed", baseErr, ErrNotFound, displayable, attrErr)
```

## Stacktrace Integration

Since stacktrace functionality requires `errx.Classified` types, this package does NOT provide mirror functions for the stacktrace package. This is an intentional design decision.

If you need stack traces, you have two options:

1. **Use stacktrace.Here() explicitly**:
   ```go
   import "github.com/go-extras/errx/stacktrace"
   
   err := compat.Wrap("failed", cause, stacktrace.Here(), ErrDatabase)
   ```

2. **Use stacktrace package functions directly**:
   ```go
   err := stacktrace.Wrap("failed", cause, classification)
   ```

## Tradeoffs

### Advantages
- Works with any error type, including third-party errors
- More flexible for codebases that heavily use standard error interface
- Easier migration path from existing error handling code

### Disadvantages
- Less type safety - you can accidentally pass non-classification errors
- Slightly more overhead due to additional wrapping layer
- Less clear intent - harder to distinguish classification metadata from regular errors

## Examples

### Basic Classification

```go
var ErrNotFound = errors.New("not found")

err := db.Get(id)
if err != nil {
    return compat.Classify(err, ErrNotFound)
}
```

### Multiple Classifications

```go
var (
    ErrDatabase  = errors.New("database error")
    ErrRetryable = errors.New("retryable error")
)

err := db.Transaction()
return compat.Wrap("transaction failed", err, ErrDatabase, ErrRetryable)
```

### With Attributes

```go
attrErr := errx.WithAttrs("user_id", 123, "action", "delete")
err := compat.Wrap("operation failed", baseErr, ErrDatabase, attrErr)

// Later, extract attributes for logging
if errx.HasAttrs(err) {
    attrs := errx.ExtractAttrs(err)
    logger.Error("error occurred", "attrs", attrs)
}
```

### Chaining Calls

```go
err1 := compat.Classify(baseErr, ErrDatabase)
err2 := compat.Wrap("layer 2", err1, ErrRetryable)
err3 := compat.Wrap("layer 3", err2)

// All classifications are preserved
errors.Is(err3, ErrDatabase)  // true
errors.Is(err3, ErrRetryable) // true
```

## When to Use

Use the `compat` package when:
- You're migrating existing code that uses standard errors
- You prefer the flexibility of standard error interface
- You're integrating with third-party libraries that use standard errors
- You want to avoid the ceremony of creating `errx.Classified` types

Use the parent `errx` package when:
- You want maximum type safety
- You're building a new codebase from scratch
- You want clear distinction between classifications and regular errors
- You need the full power of the sealed interface pattern

## See Also

- [errx package documentation](https://pkg.go.dev/github.com/go-extras/errx)
- [errx/stacktrace package](https://pkg.go.dev/github.com/go-extras/errx/stacktrace)

