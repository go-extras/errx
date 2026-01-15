# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `ToSlogAttrs()` method for converting `errx.Attrs` to `[]slog.Attr` for efficient slog integration
- `ToSlogArgs()` method for converting `errx.Attrs` to `[]any` for convenient slog integration

#### Compat Subpackage
- New `compat` subpackage for standard error interface compatibility
- `compat.Wrap()` - Wrap errors accepting standard `error` interface for classifications
- `compat.Classify()` - Classify errors accepting standard `error` interface for classifications
- Internal conversion from standard errors to `errx.Classified` while preserving error identity
- Full compatibility with all errx features (sentinels, displayable errors, attributes)
- Comprehensive documentation explaining design decisions and tradeoffs
- 16 unit tests + 8 example tests
- Package README with usage examples and guidance

## [1.0.0] - TBD

### Added

#### Core Package
- `NewSentinel()` - Create classification sentinels for programmatic error checking
- `Wrap()` - Wrap errors with context and optional classifications
- `Classify()` - Attach classifications to errors without adding context
- Hierarchical sentinel support with multiple parent sentinels
- `NewDisplayable()` - Create user-safe displayable error messages
- `IsDisplayable()` - Check if error chain contains displayable message
- `DisplayText()` - Extract displayable message from error chain
- `DisplayTextDefault()` - Extract displayable message with fallback
- `WithAttrs()` - Create errors with structured key-value attributes
- `FromAttrMap()` - Create attributed errors from maps
- `HasAttrs()` - Check if error contains attributes
- `ExtractAttrs()` - Extract all attributes from error chain
- `Classified` interface for extensibility by external packages
- Support for Go 1.20+ multi-error unwrapping
- Full compatibility with `errors.Is()` and `errors.As()`

#### Stacktrace Subpackage
- `Here()` - Capture stack trace at current location
- `Wrap()` - Wrap errors with automatic stack trace capture
- `Classify()` - Classify errors with automatic stack trace capture
- `Extract()` - Extract stack frames from error chain
- `Format()` - Format stack traces for display
- Zero-dependency implementation using only Go stdlib

#### JSON Subpackage
- `Marshal()` - Serialize errx errors to JSON
- `MarshalIndent()` - Serialize with pretty-printing
- `ToSerializedError()` - Convert to structured format before serialization
- `WithMaxDepth()` - Configure maximum error chain depth
- `WithMaxStackFrames()` - Configure maximum stack frames
- `WithIncludeStandardErrors()` - Control standard error inclusion
- Comprehensive serialization of sentinels, displayable errors, attributes, and stack traces
- Circular reference detection
- Multi-error support

### Documentation
- Comprehensive README with examples and best practices
- Package-level documentation for all packages
- 15 runnable example tests
- Subpackage READMEs (stacktrace, json)
- WARP.md for AI assistant guidance

### Testing
- 93.1% test coverage in core package
- 85.4% test coverage in json subpackage
- 83.8% test coverage in stacktrace subpackage
- Comprehensive benchmark suite
- Race detection enabled in CI
- External implementation tests

### Infrastructure
- GitHub Actions CI/CD pipeline
- golangci-lint with strict configuration
- govulncheck security scanning
- Issue templates (bug report, feature request, question)
- Pull request template
- Contributing guidelines

[Unreleased]: https://github.com/go-extras/errx/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/go-extras/errx/releases/tag/v1.0.0

