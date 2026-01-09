# errx/json

JSON serialization for errx errors.

## Overview

The `errx/json` package provides JSON serialization capabilities for errx errors while maintaining the zero-dependency principle of the core errx package. It serializes all errx error types including classification sentinels, displayable errors, attributed errors, and stack traces.

## Installation

```bash
go get github.com/go-extras/errx/json
```

## Usage

### Basic Serialization

```go
import (
    "github.com/go-extras/errx"
    errxjson "github.com/go-extras/errx/json"
)

err := errx.Wrap("failed to fetch user", cause, ErrNotFound)
jsonBytes, _ := errxjson.Marshal(err)
```

### Pretty Printing

```go
jsonBytes, _ := errxjson.MarshalIndent(err, "", "  ")
```

### Convert to Struct

```go
serialized := errxjson.ToSerializedError(err)
// Manipulate the struct before serializing
jsonBytes, _ := json.Marshal(serialized)
```

## Configuration Options

### WithMaxDepth

Limit the depth of error chain traversal to prevent issues with deeply nested or circular error chains.

```go
jsonBytes, _ := errxjson.Marshal(err, errxjson.WithMaxDepth(16))
```

### WithMaxStackFrames

Limit the number of stack frames included in the serialized output to reduce JSON size.

```go
jsonBytes, _ := errxjson.Marshal(err, errxjson.WithMaxStackFrames(10))
```

### WithIncludeStandardErrors

Control whether standard (non-errx) errors in the error chain are included.

```go
// Only include errx errors
jsonBytes, _ := errxjson.Marshal(err, errxjson.WithIncludeStandardErrors(false))
```

## JSON Structure

The serialized error has the following structure:

```json
{
  "message": "error message from Error()",
  "display_text": "user-facing message (if displayable error present)",
  "sentinels": ["list", "of", "sentinel", "texts"],
  "attributes": [
    {"key": "user_id", "value": 123},
    {"key": "action", "value": "delete"}
  ],
  "stack_trace": [
    {
      "file": "/path/to/file.go",
      "line": 42,
      "function": "package.FunctionName"
    }
  ],
  "cause": {
    "message": "wrapped error"
  },
  "causes": [
    {"message": "error 1"},
    {"message": "error 2"}
  ]
}
```

Fields are omitted if empty (using `omitempty` tags).

## Examples

### Displayable Error

```go
displayErr := errx.NewDisplayable("Invalid email address")
err := errx.Wrap("validation failed", displayErr)

jsonBytes, _ := errxjson.MarshalIndent(err, "", "  ")
// {
//   "message": "validation failed: Invalid email address",
//   "display_text": "Invalid email address",
//   "cause": {
//     "message": "Invalid email address",
//     "display_text": "Invalid email address"
//   }
// }
```

### Error with Attributes

```go
attrErr := errx.WithAttrs("user_id", 123, "action", "delete")
err := errx.Classify(baseErr, attrErr, ErrDatabase)

jsonBytes, _ := errxjson.Marshal(err)
```

### Error with Stack Trace

```go
import "github.com/go-extras/errx/stacktrace"

err := stacktrace.Wrap("operation failed", baseErr, ErrDatabase)
jsonBytes, _ := errxjson.Marshal(err)
// Stack trace will be included in the "stack_trace" field
```

### Complex Error

```go
baseErr := errors.New("connection timeout")
displayErr := errx.NewDisplayable("Service temporarily unavailable")
attrErr := errx.WithAttrs("retry_count", 3, "host", "localhost")

err := stacktrace.Wrap("database query failed",
    baseErr, displayErr, attrErr, ErrTimeout)

jsonBytes, _ := errxjson.MarshalIndent(err, "", "  ")
```

## Use Cases

### API Error Responses

```go
func handleError(w http.ResponseWriter, err error) {
    jsonBytes, _ := errxjson.Marshal(err)
    w.Header().Set("Content-Type", "application/json")
    w.Write(jsonBytes)
}
```

### Structured Logging

```go
serialized := errxjson.ToSerializedError(err)
slog.Error("operation failed",
    "error_message", serialized.Message,
    "display_text", serialized.DisplayText,
    "attributes", serialized.Attributes)
```

### Error Persistence

```go
// Store error in database
jsonBytes, _ := errxjson.Marshal(err)
db.SaveError(jsonBytes)

// Later, read and inspect
var serialized errxjson.SerializedError
json.Unmarshal(jsonBytes, &serialized)
```

## Design Principles

1. **Zero Dependencies**: Uses only Go's standard library `encoding/json`
2. **No Deserialization**: Only provides serialization (one-way), not deserialization
3. **Comprehensive Coverage**: Handles all errx error types and standard errors
4. **Safety First**: Includes circular reference detection and depth limits
5. **Flexible Configuration**: Options for customizing serialization behavior

## Limitations

- **No Deserialization**: This package does not provide deserialization (JSON to error) functionality. Errors are runtime constructs and cannot be meaningfully reconstructed from JSON.
- **Sentinel Hierarchy**: Parent sentinels in hierarchical relationships are not serialized - only direct sentinels are included.
- **Attribute Ordering**: The order of attributes in the JSON output is stable for a given error but should not be relied upon for semantic meaning.
