# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - 2026-01-31

This release adds a new convenience function `ClassifyNew` for creating and classifying errors in a single step.

### Added

- **New `ClassifyNew()` function** - Added `ClassifyNew(text string, classifications ...Classified) error` function to create a new error and immediately classify it with one or more classifications. This convenience function makes the code more concise and readable:
  ```go
  // Before
  err := errx.Classify(errors.New("some error"), ErrNotFound, ErrDatabase)

  // After
  err := errx.ClassifyNew("some error", ErrNotFound, ErrDatabase)
  ```
  This also eliminates the need to import the `errors` package in many cases.

- **compat.ClassifyNew()** - Added `compat.ClassifyNew(text string, classifications ...error) error` function that accepts standard Go `error` interface for classifications, maintaining compatibility with existing error types.

- **stacktrace.ClassifyNew()** - Added `stacktrace.ClassifyNew(text string, classifications ...errx.Classified) error` function that automatically captures stack traces at the call site while creating and classifying errors.

### Testing

- Added 15 comprehensive unit tests across all three packages (errx, compat, stacktrace)
- Added 6 example tests with output verification demonstrating usage patterns
- All tests pass with 100% success rate

## [1.1.0] - 2026-01-31

This release refactors the attribute API to improve naming consistency and clarity. The `Attrs` type has been renamed to `AttrList` to avoid confusion with the new `Attrs()` function, which provides a more concise API for creating attributed errors.

### Breaking Changes

- **Renamed `Attrs` type to `AttrList`** - The type alias for `[]Attr` has been renamed from `Attrs` to `AttrList`. This is a breaking change for code that directly references the `Attrs` type. Users should update their code to use `AttrList` instead:
  ```go
  // Before
  var attrs errx.Attrs = errx.ExtractAttrs(err)

  // After
  var attrs errx.AttrList = errx.ExtractAttrs(err)
  ```
  Note: Most users are not affected by this change as the type is typically used implicitly through `ExtractAttrs()` return values.

### Added

- **New `Attrs()` function** - Added a new `Attrs(attrs ...any) Classified` function as the primary API for creating attributed errors. This provides a more concise and intuitive name compared to `WithAttrs()`:
  ```go
  // New recommended approach
  attrErr := errx.Attrs("user_id", 123, "action", "delete")
  return errx.Wrap("operation failed", baseErr, attrErr)
  ```

### Deprecated

- **Deprecated `WithAttrs()` function** - The `WithAttrs()` function is now deprecated in favor of the new `Attrs()` function. `WithAttrs()` will continue to work for backward compatibility, but users are encouraged to migrate to `Attrs()`:
  ```go
  // Deprecated
  attrErr := errx.WithAttrs("user_id", 123)

  // Recommended
  attrErr := errx.Attrs("user_id", 123)
  ```

## [1.0.0] - 2026-01-15

**First stable release** of errx - a rich error handling library for Go with classification tags, displayable messages, and structured attributes.

This release provides a complete, production-ready error handling solution with comprehensive features for building robust Go applications. The library is designed for developers building production systems that need sophisticated error handling, clear separation between internal and user-facing errors, and rich contextual information for debugging.

### Core Features

#### Error Classification
- **Sentinel-based classification** - Create error sentinels with `NewSentinel()` for programmatic error checking
- **Hierarchical sentinels** - Support for parent sentinels to build error taxonomies
- **Wrap and Classify** - `Wrap()` adds context and classification; `Classify()` adds classification without context
- **Standard library compatibility** - Full support for `errors.Is()` and `errors.As()`
- **Extensible interface** - `Classified` interface allows external packages to implement custom error types

#### Displayable Messages
- **User-safe messages** - `NewDisplayable()` creates messages safe to show to end users
- **Message extraction** - `DisplayText()` and `DisplayTextDefault()` extract displayable messages from error chains
- **Separation of concerns** - Keep internal error details separate from user-facing messages

#### Structured Attributes
- **Key-value metadata** - `WithAttrs()` attaches structured attributes to errors
- **Map support** - `FromAttrMap()` creates attributed errors from maps
- **Attribute extraction** - `ExtractAttrs()` retrieves all attributes from error chain
- **Logging integration** - `ToSlogAttrs()` and `ToSlogArgs()` for seamless slog integration

### Subpackages

#### stacktrace - Optional Stack Trace Support
- **Opt-in stack traces** - `Here()` captures stack trace at specific locations
- **Automatic capture** - `stacktrace.Wrap()` and `stacktrace.Classify()` with automatic stack trace capture
- **Stack extraction** - `Extract()` retrieves stack frames from error chain
- **Zero dependencies** - Uses only Go standard library

#### json - JSON Serialization
- **Comprehensive serialization** - `Marshal()` and `MarshalIndent()` for JSON output
- **Configurable options** - Control depth, stack frames, and standard error inclusion
- **Safe serialization** - Circular reference detection and depth limits
- **Zero dependencies** - Uses only Go standard library

#### compat - Standard Error Interface Compatibility
- **Standard error support** - `compat.Wrap()` and `compat.Classify()` accept any `error` type
- **Migration friendly** - Easier transition from existing error handling code
- **Full feature support** - Works with all errx features (sentinels, displayable, attributes)
- **Preserved identity** - Maintains `errors.Is()` and `errors.As()` compatibility

### Documentation & Testing

- **Comprehensive documentation** - Detailed README with examples and best practices
- **Package documentation** - Complete API documentation for all packages
- **Example tests** - 15+ runnable examples demonstrating all features
- **High test coverage** - 93.1% (core), 85.4% (json), 83.8% (stacktrace), 100% (compat)
- **Benchmark suite** - Performance benchmarks for all major operations
- **Contributing guide** - Clear guidelines for contributors

### Infrastructure

- **CI/CD pipeline** - GitHub Actions with comprehensive testing
- **Code quality** - golangci-lint with strict configuration
- **Security scanning** - govulncheck for vulnerability detection
- **Issue templates** - Bug reports, feature requests, and questions
- **Pull request template** - Standardized PR process

[1.2.0]: https://github.com/go-extras/errx/releases/tag/v1.2.0
[1.1.0]: https://github.com/go-extras/errx/releases/tag/v1.1.0
[1.0.0]: https://github.com/go-extras/errx/releases/tag/v1.0.0

