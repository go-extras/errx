package json

// Option is a function that configures the JSON serialization behavior.
type Option func(*config)

// WithMaxDepth sets the maximum depth for traversing error chains.
// This prevents issues with deeply nested or potentially circular error chains.
// The default is 32.
//
// When the depth limit is reached, the serialized error will have a message
// of "(max depth reached)" and no further unwrapping will occur.
//
// Example:
//
//	jsonBytes, err := json.Marshal(err, json.WithMaxDepth(10))
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
