# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[1.0.0]: https://github.com/go-extras/errx/releases/tag/v1.0.0

