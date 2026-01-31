package json_test

import (
	"errors"
	"fmt"

	"github.com/go-extras/errx"
	errxjson "github.com/go-extras/errx/json"
	"github.com/go-extras/errx/stacktrace"
)

var (
	ErrNotFound = errx.NewSentinel("not found")
	ErrDatabase = errx.NewSentinel("database")
)

// ExampleMarshal demonstrates basic JSON serialization of errx errors
func ExampleMarshal() {
	err := errx.Wrap("failed to fetch user", errors.New("connection timeout"), ErrNotFound)

	jsonBytes, _ := errxjson.Marshal(err)
	fmt.Println(string(jsonBytes))

	// Output:
	// {"message":"failed to fetch user: connection timeout","sentinels":["not found"],"cause":{"message":"connection timeout"}}
}

// ExampleMarshalIndent demonstrates pretty-printed JSON serialization
func ExampleMarshalIndent() {
	displayErr := errx.NewDisplayable("User not found")
	err := errx.Wrap("lookup failed", displayErr, ErrNotFound)

	jsonBytes, _ := errxjson.MarshalIndent(err, "", "  ")
	fmt.Println(string(jsonBytes))

	// Output:
	// {
	//   "message": "lookup failed: User not found",
	//   "display_text": "User not found",
	//   "sentinels": [
	//     "not found"
	//   ],
	//   "cause": {
	//     "message": "User not found",
	//     "display_text": "User not found"
	//   }
	// }
}

// ExampleToSerializedError demonstrates converting an error to a struct
func ExampleToSerializedError() {
	attrErr := errx.Attrs("user_id", 42)
	err := errx.Classify(errors.New("operation failed"), attrErr, ErrDatabase)

	serialized := errxjson.ToSerializedError(err)
	fmt.Printf("Message: %s\n", serialized.Message)
	fmt.Printf("Attributes: %d\n", len(serialized.Attributes))
	fmt.Printf("Sentinels: %v\n", serialized.Sentinels)

	// Output:
	// Message: operation failed
	// Attributes: 1
	// Sentinels: [database]
}

// ExampleWithMaxDepth demonstrates limiting error chain depth
func ExampleWithMaxDepth() {
	// Create a deep error chain
	err := errors.New("level 3")
	err = errx.Wrap("level 2", err)
	err = errx.Wrap("level 1", err)

	// Limit serialization to 2 levels
	jsonBytes, _ := errxjson.Marshal(err, errxjson.WithMaxDepth(2))
	fmt.Println(string(jsonBytes))

	// Output:
	// {"message":"level 1: level 2: level 3","cause":{"message":"level 2: level 3","cause":{"message":"(max depth reached)"}}}
}

// ExampleWithMaxStackFrames demonstrates limiting stack trace frames
func ExampleWithMaxStackFrames() {
	err := stacktrace.Wrap("operation failed", errors.New("base error"))

	// Limit to 3 stack frames
	serialized := errxjson.ToSerializedError(err, errxjson.WithMaxStackFrames(3))
	fmt.Printf("Stack frames: %d\n", len(serialized.StackTrace))
	fmt.Println("Has stack trace: true")

	// Output:
	// Stack frames: 3
	// Has stack trace: true
}

// ExampleWithIncludeStandardErrors demonstrates filtering error types
func ExampleWithIncludeStandardErrors() {
	// Mix of errx and standard errors
	stdErr := errors.New("standard error")
	err := errx.Wrap("wrapper", stdErr, ErrDatabase)

	// Exclude standard errors
	serialized := errxjson.ToSerializedError(err, errxjson.WithIncludeStandardErrors(false))

	// The standard error is skipped
	fmt.Printf("Has cause: %v\n", serialized.Cause != nil)
	fmt.Printf("Sentinels: %v\n", serialized.Sentinels)

	// Output:
	// Has cause: false
	// Sentinels: [database]
}

// Example_displayableError demonstrates serializing displayable errors
func Example_displayableError() {
	displayErr := errx.NewDisplayable("Invalid email address")
	err := errx.Wrap("validation failed", displayErr)

	jsonBytes, _ := errxjson.MarshalIndent(err, "", "  ")
	fmt.Println(string(jsonBytes))

	// Output:
	// {
	//   "message": "validation failed: Invalid email address",
	//   "display_text": "Invalid email address",
	//   "cause": {
	//     "message": "Invalid email address",
	//     "display_text": "Invalid email address"
	//   }
	// }
}

// Example_attributedError demonstrates serializing errors with attributes
func Example_attributedError() {
	attrErr := errx.Attrs(
		"user_id", 12345,
		"action", "delete",
		"resource", "account",
	)
	err := errx.Classify(errors.New("operation failed"), attrErr, ErrDatabase)

	jsonBytes, _ := errxjson.MarshalIndent(err, "", "  ")
	fmt.Println(string(jsonBytes))

	// Output:
	// {
	//   "message": "operation failed",
	//   "sentinels": [
	//     "database"
	//   ],
	//   "attributes": [
	//     {
	//       "key": "user_id",
	//       "value": 12345
	//     },
	//     {
	//       "key": "action",
	//       "value": "delete"
	//     },
	//     {
	//       "key": "resource",
	//       "value": "account"
	//     }
	//   ],
	//   "cause": {
	//     "message": "operation failed"
	//   }
	// }
}

// Example_complexError demonstrates serializing errors with all features
func Example_complexError() {
	// Build a rich error with all features
	baseErr := errors.New("connection timeout")
	displayErr := errx.NewDisplayable("Service temporarily unavailable")
	attrErr := errx.Attrs("retry_count", 3, "host", "localhost")

	err := stacktrace.Wrap("database query failed",
		baseErr, displayErr, attrErr, ErrDatabase)

	serialized := errxjson.ToSerializedError(err)

	fmt.Printf("Has display text: %v\n", serialized.DisplayText != "")
	fmt.Printf("Has attributes: %v\n", len(serialized.Attributes) > 0)
	fmt.Printf("Has stack trace: %v\n", len(serialized.StackTrace) > 0)
	fmt.Printf("Has sentinels: %v\n", len(serialized.Sentinels) > 0)

	// Output:
	// Has display text: true
	// Has attributes: true
	// Has stack trace: true
	// Has sentinels: true
}

// Example_errorChain demonstrates serializing error chains
func Example_errorChain() {
	err1 := errors.New("root cause")
	err2 := errx.Wrap("middle layer", err1, ErrDatabase)
	err3 := errx.Wrap("top layer", err2, ErrNotFound)

	serialized := errxjson.ToSerializedError(err3)

	// Walk the chain
	fmt.Println("Top:", serialized.Message)
	if serialized.Cause != nil {
		fmt.Println("Middle:", serialized.Cause.Message)
		if serialized.Cause.Cause != nil {
			fmt.Println("Root:", serialized.Cause.Cause.Message)
		}
	}

	// Output:
	// Top: top layer: middle layer: root cause
	// Middle: middle layer: root cause
	// Root: root cause
}

// Example_apiResponse demonstrates using JSON serialization for API responses
func Example_apiResponse() {
	// Simulate an error from the service layer
	displayErr := errx.NewDisplayable("User not found")
	attrErr := errx.Attrs("user_id", "12345")
	serviceErr := errx.Classify(displayErr, ErrNotFound, attrErr)

	// Serialize for API response
	jsonBytes, _ := errxjson.MarshalIndent(serviceErr, "", "  ")
	fmt.Println(string(jsonBytes))

	// Output:
	// {
	//   "message": "User not found",
	//   "display_text": "User not found",
	//   "sentinels": [
	//     "not found"
	//   ],
	//   "attributes": [
	//     {
	//       "key": "user_id",
	//       "value": "12345"
	//     }
	//   ],
	//   "cause": {
	//     "message": "User not found",
	//     "display_text": "User not found"
	//   }
	// }
}
