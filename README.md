# errx

Rich error handling with classification tags, displayable messages, and structured attributes for Go

[![CI](https://github.com/go-extras/errx/actions/workflows/go-test.yml/badge.svg?branch=master)](https://github.com/go-extras/errx/actions/workflows/go-test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-extras/errx.svg)](https://pkg.go.dev/github.com/go-extras/errx)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-extras/errx)](https://goreportcard.com/report/github.com/go-extras/errx)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.25-00ADD8?logo=go)](https://go.dev/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Overview

`errx` is a powerful error handling library for Go that extends the standard library's error handling with five key capabilities:

- **Classification Tags**: Categorize errors for programmatic checking without cluttering error messages
- **Displayable Errors**: Create user-safe messages that can be extracted from error chains
- **Structured Attributes**: Attach key-value metadata for logging and debugging
- **Stack Traces** (optional): Capture call stacks for debugging via the `stacktrace` subpackage
- **JSON Serialization** (optional): Serialize errors to JSON for API responses and logging via the `json` subpackage

The library is designed for developers building production systems that need sophisticated error handling, clear separation between internal and user-facing errors, and rich contextual information for debugging.

**Flexibility:** The core package uses a type-safe `Classified` interface for maximum safety and clarity. For codebases that prefer working with standard Go `error` interface, the `compat` subpackage provides compatible functions that accept any error type.

**Target audience:** Backend developers, API engineers, and application architects building robust systems with comprehensive error handling requirements.

## Features

- ✅ **Hierarchical error classification** with sentinel-based error checking
- ✅ **User-safe displayable messages** separate from internal error details
- ✅ **Structured attributes** for rich logging and debugging context
- ✅ **Optional stack traces** via the `stacktrace` subpackage
- ✅ **JSON serialization** via the `json` subpackage for API responses and logging
- ✅ **Standard error compatibility** via the `compat` subpackage for flexible integration
- ✅ **Zero dependencies** in core package (stacktrace and json use only Go stdlib)
- ✅ **Well-tested** with comprehensive test coverage
- ✅ **Simple API** designed for ease of use and composability
- ✅ **Compatible** with standard `errors.Is()` and `errors.As()`

## Requirements

- Go 1.25+ (tested on 1.25.x)

## Installation

Add the library to your Go module:

```bash
go get github.com/go-extras/errx@latest
```

## Quick Start

```go
package main

import (
    "errors"
    "fmt"
    "github.com/go-extras/errx"
)

// Define classification sentinels
var (
    ErrNotFound = errx.NewSentinel("resource not found")
    ErrInvalid  = errx.NewSentinel("invalid input")
)

func processOrder(orderID string) error {
    // Create a displayable error
    displayErr := errx.NewDisplayable("Order not found")
    
    // Wrap with context and classification
    return errx.Wrap("failed to process order", displayErr, ErrNotFound)
}

func main() {
    err := processOrder("12345")
    
    // Check error classification
    if errors.Is(err, ErrNotFound) {
        fmt.Println("Resource was not found")
    }
    
    // Extract displayable message
    if errx.IsDisplayable(err) {
        msg := errx.DisplayText(err)
        fmt.Println("User-safe message:", msg)
    }
    
    // Full error for logging
    fmt.Println("Full error:", err)
}
```

## Core Concepts

### Classification Sentinels

Classification sentinels let you identify error types without adding text to the error message chain.

```go
// Define sentinels
var (
    ErrDatabase   = errx.NewSentinel("database error")
    ErrNetwork    = errx.NewSentinel("network error")
    ErrValidation = errx.NewSentinel("validation error")
)

// Use Classify to classify an error without adding context
func fetchData() error {
    err := db.Execute("SELECT * FROM data")
    if err != nil {
        return errx.Classify(err, ErrDatabase)
    }
    return nil
}

// Use Wrap to add both context and classification
func getData(id string) error {
    err := fetchData()
    if err != nil {
        return errx.Wrap("failed to get data", err, ErrDatabase)
    }
    return nil
}

// Check the classification
err := getData("123")
if errors.Is(err, ErrDatabase) {
    // Handle database error
}
```

**When to use `Classify` vs `Wrap`:**
- Use `Classify` when the error message is already clear and you just need to classify it
- Use `Wrap` when you need to add contextual information about where/why the error occurred

### Hierarchical Sentinels

Create hierarchical error taxonomies by passing parent sentinels to `NewSentinel`:

```go
var (
    // Parent categories
    ErrRetryable    = errx.NewSentinel("retryable")
    ErrPermanent    = errx.NewSentinel("permanent")

    // Child sentinels with parents
    ErrTimeout      = errx.NewSentinel("timeout", ErrRetryable)
    ErrRateLimit    = errx.NewSentinel("rate limit", ErrRetryable)
    ErrNotFound     = errx.NewSentinel("not found", ErrPermanent)
    ErrForbidden    = errx.NewSentinel("forbidden", ErrPermanent)
)

func handleRequest() error {
    err := makeAPICall()
    
    // Check for specific error
    if errors.Is(err, ErrTimeout) {
        // Retry with backoff
        return retryWithBackoff()
    }
    
    // Check for general category
    if errors.Is(err, ErrRetryable) {
        // Any retryable error
        return retry()
    }
    
    // Check for permanent errors
    if errors.Is(err, ErrPermanent) {
        // Don't retry
        return err
    }
    
    return err
}
```

**Multiple Parent Sentinels:**

You can also create sentinels with multiple parents for multi-dimensional classification:

```go
var (
    // Classification dimensions
    ErrRetryable = errx.NewSentinel("retryable")
    ErrDatabase  = errx.NewSentinel("database")
    ErrNetwork   = errx.NewSentinel("network")

    // Sentinels with multiple parents
    ErrDatabaseTimeout = errx.NewSentinel("database timeout", ErrDatabase, ErrRetryable)
    ErrNetworkTimeout  = errx.NewSentinel("network timeout", ErrNetwork, ErrRetryable)
)

// Now you can check errors along multiple dimensions
err := query()
if errors.Is(err, ErrDatabase) {
    // Handle any database error
}
if errors.Is(err, ErrRetryable) {
    // Handle any retryable error (database or network)
}
```

### Displayable Messages

Separate user-safe messages from internal error details:

```go
func validateEmail(email string) error {
    if !strings.Contains(email, "@") {
        return errx.NewDisplayable("Please enter a valid email address")
    }
    return nil
}

func createAccount(email string) error {
    err := validateEmail(email)
    if err != nil {
        // Add internal context
        return errx.Wrap("account creation failed", err, ErrValidation)
    }
    return nil
}

// In your API handler
func handleCreateAccount(w http.ResponseWriter, r *http.Request) {
    err := createAccount(email)
    if err != nil {
        // Extract user-safe message
        userMsg := "An error occurred"
        if errx.IsDisplayable(err) {
            userMsg = errx.DisplayText(err)
        }

        // Log full error internally
        log.Error("account creation failed", "error", err)
        
        // Send safe message to user
        http.Error(w, userMsg, http.StatusBadRequest)
    }
}
```

### Structured Attributes

Attach key-value metadata for structured logging:

```go
func processPayment(userID string, amount float64) error {
    if amount < 0 {
        // Create attributed error
        attrErr := errx.Attrs(
            "user_id", userID,
            "amount", amount,
            "currency", "USD",
        )
        return errx.Wrap("payment validation failed", attrErr, ErrValidation)
    }
    return nil
}

// Extract attributes for logging
err := processPayment("user123", -50.0)
if errx.HasAttrs(err) {
    attrs := errx.ExtractAttrs(err)
    log.Error("payment failed", "error", err, "attributes", attrs)
}
```

#### Integration with slog

Convert `errx.AttrList` for seamless integration with structured logging. Two methods are provided:

**Option 1: `ToSlogAttrs()` - Most efficient (recommended)**

Use with `Logger.LogAttrs` for best performance and type safety:

```go
err := errx.Attrs("user_id", 123, "action", "delete")
attrs := errx.ExtractAttrs(err)

// Convert to []slog.Attr
slogAttrs := attrs.ToSlogAttrs()

// Use with LogAttrs (most efficient)
logger := slog.Default()
logger.LogAttrs(context.Background(), slog.LevelError, "operation failed", slogAttrs...)
```

**Option 2: `ToSlogArgs()` - Convenient**

Use with `Error`, `Info`, `Warn` methods for convenience:

```go
err := errx.Attrs("user_id", 123, "action", "delete")
attrs := errx.ExtractAttrs(err)

// Convert to []any
slogArgs := attrs.ToSlogArgs()

// Use with convenience methods
logger := slog.Default()
logger.Error("operation failed", slogArgs...)
```

### Stack Traces (Optional)

The `stacktrace` subpackage provides optional stack trace support while keeping the core `errx` package minimal and zero-dependency:

```go
import (
    "github.com/go-extras/errx"
    "github.com/go-extras/errx/stacktrace"
)

// Option 1: Per-error opt-in using Here()
err := errx.Wrap("operation failed", cause, ErrNotFound, stacktrace.Here())

// Option 2: Automatic capture with stacktrace.Wrap()
err := stacktrace.Wrap("operation failed", cause, ErrNotFound)

// Extract and use stack traces
frames := stacktrace.Extract(err)
if frames != nil {
    for _, frame := range frames {
        fmt.Printf("%s:%d %s\n", frame.File, frame.Line, frame.Function)
    }
}
```

**Key features:**
- **Opt-in**: Stack traces are only captured when explicitly requested
- **Zero overhead**: Core `errx` package remains dependency-free and fast
- **Composable**: Works seamlessly with all other `errx` features (sentinels, displayable, attributes)
- **Two usage patterns**: Per-error with `Here()` or automatic with `stacktrace.Wrap()`

See the [stacktrace package documentation](https://pkg.go.dev/github.com/go-extras/errx/stacktrace) for more details.

### JSON Serialization (json package)

The `json` subpackage provides JSON serialization capabilities for errx errors while maintaining the zero-dependency principle of the core package:

```go
import (
    "github.com/go-extras/errx"
    errxjson "github.com/go-extras/errx/json"
)

// Create a complex error with all features
displayErr := errx.NewDisplayable("Service temporarily unavailable")
attrErr := errx.Attrs("retry_count", 3, "host", "localhost")
err := errx.Wrap("database operation failed", displayErr, attrErr, ErrTimeout)

// Serialize to JSON
jsonBytes, _ := errxjson.Marshal(err)

// Pretty print
jsonBytes, _ := errxjson.MarshalIndent(err, "", "  ")
```

**Key features:**
- **Comprehensive serialization**: Handles all errx error types (sentinels, displayable, attributes, stack traces)
- **Zero dependencies**: Uses only Go's standard library `encoding/json`
- **Configurable**: Options for max depth, max stack frames, and filtering
- **Safe**: Includes circular reference detection and depth limits

**Configuration options:**

```go
// Limit error chain depth
jsonBytes, _ := errxjson.Marshal(err, errxjson.WithMaxDepth(16))

// Limit stack frames
jsonBytes, _ := errxjson.Marshal(err, errxjson.WithMaxStackFrames(10))

// Exclude standard errors
jsonBytes, _ := errxjson.Marshal(err, errxjson.WithIncludeStandardErrors(false))
```

See the [json package documentation](https://pkg.go.dev/github.com/go-extras/errx/json) for more details.

### Standard Error Compatibility (compat package)

The `compat` subpackage provides an alternative API that accepts standard Go `error` interface instead of requiring `errx.Classified` types. This is useful for:

- **Migration**: Easier transition from existing error handling code
- **Third-party integration**: Working with libraries that use standard errors
- **Flexibility**: Preferring standard error interface over type-safe classifications

```go
import (
    "errors"
    "github.com/go-extras/errx/compat"
)

// Define classification errors using standard errors
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

**Key features:**
- **Accepts any error type**: Works with `errors.New()`, third-party errors, etc.
- **Preserves error identity**: `errors.Is()` and `errors.As()` work correctly
- **Seamless integration**: Can mix standard errors with `errx.Classified` types
- **Same functionality**: Supports attributes, displayable errors, and all errx features

**Tradeoffs:**
- ✅ More flexible for codebases using standard error interface
- ✅ Easier migration path from existing code
- ⚠️ Less type safety - can accidentally pass non-classification errors
- ⚠️ Slightly more overhead due to additional wrapping layer

**When to use:**
- Use `compat` when migrating existing code or integrating with third-party libraries
- Use the main `errx` package for new code where type safety is preferred

See the [compat package documentation](https://pkg.go.dev/github.com/go-extras/errx/compat) for more details.

## Complete Example

```go
package main

import (
    "errors"
    "fmt"
    "github.com/go-extras/errx"
)

// Define error taxonomy
var (
    // Top-level categories
    ErrClient = errx.NewSentinel("client error")
    ErrServer = errx.NewSentinel("server error")

    // Specific errors
    ErrNotFound     = errx.NewSentinel("not found", ErrClient)
    ErrUnauthorized = errx.NewSentinel("unauthorized", ErrClient)
    ErrDatabase     = errx.NewSentinel("database", ErrServer)
)

// Data layer - adds attributes
func findUserInDB(userID string) error {
    // Simulate database error
    dbErr := errors.New("connection timeout")
    attrErr := errx.Attrs("user_id", userID, "operation", "select")
    return errx.Classify(dbErr, attrErr)
}

// Service layer - adds classification and context
func getUser(userID string) error {
    err := findUserInDB(userID)
    if err != nil {
        return errx.Wrap("failed to find user", err, ErrDatabase)
    }
    return nil
}

// API layer - adds displayable message
func handleGetUser(userID string) error {
    err := getUser(userID)
    if err != nil {
        displayErr := errx.NewDisplayable("User not found")
        return errx.Classify(err, displayErr, ErrNotFound)
    }
    return nil
}

func main() {
    err := handleGetUser("user123")
    
    if err != nil {
        // Determine status code from classification
        statusCode := 500
        switch {
        case errors.Is(err, ErrNotFound):
            statusCode = 404
        case errors.Is(err, ErrUnauthorized):
            statusCode = 401
        case errors.Is(err, ErrClient):
            statusCode = 400
        }
        
        // Get user-safe message
        userMsg := "An internal error occurred"
        if errx.IsDisplayable(err) {
            userMsg = errx.DisplayText(err)
        }
        
        // Extract attributes for logging
        if errx.HasAttrs(err) {
            attrs := errx.ExtractAttrs(err)
            fmt.Printf("Error: %v, Status: %d, Attrs: %v\n", 
                err, statusCode, attrs)
        }
        
        // Send response
        fmt.Printf("HTTP %d: %s\n", statusCode, userMsg)
    }
}
```

## Best Practices

### 1. Define Sentinels at Package Level

```go
package orders

var (
    ErrOrderNotFound = errx.NewSentinel("order not found")
    ErrOrderExpired  = errx.NewSentinel("order expired")
)
```

### 2. Add Displayable Messages at Domain Boundaries

Create displayable errors where you validate input or detect user-relevant conditions:

```go
func validateOrder(order Order) error {
    if order.Total < 0 {
        return errx.NewDisplayable("Order total cannot be negative")
    }
    return nil
}
```

### 3. Use Classify to Preserve Clear Messages

```go
// The validation error already has a clear message
err := validateOrder(order)
if err != nil {
    // Just add classification, don't wrap
    return errx.Classify(err, ErrInvalid)
}
```

### 4. Use Wrap to Add Context

```go
func processOrder(order Order) error {
    err := saveOrder(order)
    if err != nil {
        // Add context about what we were doing
        return errx.Wrap("failed to process order", err, ErrDatabase)
    }
    return nil
}
```

### 5. Check Sentinels from Specific to General

```go
switch {
case errors.Is(err, ErrDatabaseTimeout):
    // Handle specific timeout
case errors.Is(err, ErrDatabase):
    // Handle any database error
case errors.Is(err, ErrServer):
    // Handle any server error
}
```

### 6. Use Attributes for Structured Logging

```go
// Attach attributes at the point where context is available
if err != nil {
    return errx.Classify(
        err,
        errx.Attrs(
            "request_id", reqID,
            "user_id", userID,
            "operation", "create_order",
        ),
    )
}
```

### 7. Separate Internal and External Errors

```go
func apiHandler(w http.ResponseWriter, r *http.Request) {
    err := businessLogic()
    if err != nil {
        // Always log full internal error
        if errx.HasAttrs(err) {
            attrs := errx.ExtractAttrs(err)
            log.Error("operation failed", "error", err, "attrs", attrs)
        } else {
            log.Error("operation failed", "error", err)
        }
        
        // Only send displayable messages to users
        userMsg := "An error occurred"
        if errx.IsDisplayable(err) {
            userMsg = errx.DisplayText(err)
        }
        http.Error(w, userMsg, determineStatusCode(err))
    }
}
```

## Pattern: Combined Classification and Display

The most powerful pattern combines all three features:

```go
func authenticateUser(username, password string) error {
    user, err := findUser(username)
    if err != nil {
        // Create displayable error
        displayErr := errx.NewDisplayable("Invalid username or password")
        // Add internal context
        wrappedErr := errx.Wrap("authentication failed", displayErr, ErrUnauthorized)
        // Add debugging attributes
        attrErr := errx.Attrs("username", username, "reason", "user_not_found")
        return errx.Classify(wrappedErr, attrErr)
    }

    if !user.CheckPassword(password) {
        displayErr := errx.NewDisplayable("Invalid username or password")
        wrappedErr := errx.Wrap("password check failed", displayErr, ErrUnauthorized)
        attrErr := errx.Attrs("username", username, "reason", "wrong_password")
        return errx.Classify(wrappedErr, attrErr)
    }
    
    return nil
}

// Usage
err := authenticateUser("alice", "wrong")

// Check classification
errors.Is(err, ErrUnauthorized) // true

// Get displayable message
errx.DisplayText(err) // "Invalid username or password"

// Get full error
err.Error() // "authentication failed: password check failed: Invalid username or password"

// Get attributes
attrs := errx.ExtractAttrs(err)
// Convert to map if needed
attrMap := make(map[string]any)
for _, a := range attrs {
    attrMap[a.Key] = a.Value
} // map[username:alice reason:wrong_password]
```

## API Documentation

Full API documentation is available at [pkg.go.dev/github.com/go-extras/errx](https://pkg.go.dev/github.com/go-extras/errx).

### Core Functions

#### Error Creation

- **`NewSentinel(message string, parents ...error) error`**
  Creates a new sentinel error for classification. Supports hierarchical error taxonomies.

- **`NewDisplayable(message string) error`**
  Creates a user-safe displayable error message.

- **`Attrs(keyvals ...any) error`**
  Creates an error with structured key-value attributes.

- **`FromAttrMap(attrs AttrMap) error`**
  Creates an attributed error from a map of key-value pairs.

#### Error Wrapping

- **`Wrap(message string, err error, sentinels ...error) error`**
  Wraps an error with context and optional classification sentinels.

- **`Classify(err error, sentinels ...error) error`**
  Adds classification to an error without adding context to the message.

#### Error Inspection

- **`IsDisplayable(err error) bool`**
  Checks if an error chain contains a displayable message.

- **`DisplayText(err error) string`**
  Extracts the displayable message from an error chain.

- **`DisplayTextDefault(err error, def string) string`**
  Extracts the displayable message or returns a fallback string when no displayable error is present.

- **`HasAttrs(err error) bool`**
  Checks if an error chain contains structured attributes.

- **`ExtractAttrs(err error) AttrList`**
  Extracts all attributes from an error chain.

- **`(AttrList).ToSlogAttrs() []slog.Attr`**
  Converts extracted attributes to `[]slog.Attr` for use with `slog.Logger.LogAttrs`.

- **`(AttrList).ToSlogArgs() []any`**
  Converts extracted attributes to `[]any` for use with slog convenience methods like `Logger.Error`.

## Use Cases

- **API error handling**: Separate internal errors from user-facing messages
- **Microservices**: Classify errors for proper HTTP status code mapping
- **Structured logging**: Attach rich context to errors for debugging
- **Error monitoring**: Track error categories and patterns
- **Domain-driven design**: Create error taxonomies that match your domain

## Comparison with Standard Errors

```go
// Standard approach
err := errors.New("user not found")
wrapped := fmt.Errorf("failed to get user: %w", err)

// errx approach
displayErr := errx.NewDisplayable("User not found")
classifiedErr := errx.Wrap("failed to get user", displayErr, ErrNotFound)

// Benefits:
// - Programmatic checking: errors.Is(err, ErrNotFound)
// - Clean user messages: errx.DisplayText(err)
// - Full internal context: err.Error()
// - Structured metadata: errx.ExtractAttrs(err)
```

## Testing

Run the test suite:

```bash
# Run all tests
go test ./...

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -cover ./...
```

## Contributing

Contributions are welcome! Please:

- Open issues for bugs, feature requests, or questions
- Submit pull requests with clear descriptions and tests
- Follow the existing code style and conventions
- Ensure all tests pass and maintain test coverage

## License

MIT © 2026 Denis Voytyuk — see [LICENSE](LICENSE) for details.

## Acknowledgments

This library builds upon Go's standard `errors` package and is inspired by best practices from the Go community for error handling in production systems.
