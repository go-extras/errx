# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **`stacktrace` subpackage no longer uses `reflect` or `unsafe`** ([#29], [#35]) — `extractCarrierClassifications` now goes through the public `errx.CarrierClassifications` accessor instead of reflecting into the unexported `carrier.classifications` field, mirroring the migration already done in the `json` subpackage ([#17]). A future rename of the unexported `carrier` type can no longer silently drop classification extraction during stack-trace traversal. No exported APIs or behavior change.

## [1.3.0] - 2026-05-17

This release lands a large set of improvements across the core package and every subpackage: four JSON Marshal output bugs are fixed, the `stacktrace` and `json` subpackages get new APIs and perf polish, typed-nil handling is hardened, internal `reflect`/`unsafe` usage in the `json` subpackage is removed, and a batch of small additions makes errx easier to integrate with non-slog loggers and PII-sensitive log pipelines. No existing public signature or JSON output schema changes; previously-buggy JSON output for certain inputs now produces the correct result.

### Fixed

- **JSON Marshal `Classify` cause duplication** ([#14], [#21]) — `errxjson.Marshal` on `errx.Classify(cause, sentinels...)` no longer emits an empty duplicate `cause` node at every nesting level. Carriers are now peeled at recursion entry: their classifications are absorbed into the current level and the carrier itself is not emitted.

- **JSON Marshal false `"(circular reference)"` on shared causes** ([#14], [#21]) — DAG-shaped error chains (e.g. `errors.Join(errx.Wrap("a", shared), errx.Wrap("b", shared))`) now serialize fully. The visited-pointer tracker is per-path (push on entry, pop on exit) and multi-error branches each get an independent copy. True cycles on a single descent are still detected.

- **JSON Marshal sentinel leak across carrier levels** ([#14], [#21]) — Each level's `sentinels` array now contains only classifications attached at that user-level call. Sentinels from deeper levels no longer bleed upward.

- **JSON Marshal abort on non-serializable attribute** ([#14], [#21]) — A single attribute value of an unsupported type (e.g. `func()`) no longer aborts the entire Marshal. `serializeAttributes` validates each value with `json.Marshal` and falls back to `fmt.Sprintf("%+v", v)` on failure so the rest of the error report is preserved.

- **Typed-nil error identity in `internal/errptr`** ([#15], [#22]) — `errptr.Get` now returns `0` for typed-nil errors (interface with non-nil type but nil data), matching its behavior for plain `nil`. Prevents spurious `"(circular reference)"` reports when multiple typed-nils share a chain.

- **Panics on typed-nil `Classified` arguments** ([#15], [#22]) — `Wrap`, `Classify`, `ClassifyNew`, and `NewSentinel` now filter out both untyped-nil and typed-nil `Classified` values via an iface-data inspection helper. Previously a typed-nil sentinel slipped past `c != nil` and panicked later inside `errors.Is` when its receiver methods dereferenced nil state.

- **`HasAttrs` inconsistent with `ExtractAttrs`** ([#15], [#22]) — `HasAttrs` now returns `true` only when the chain has at least one non-empty attribute set, short-circuiting on the first non-empty find. The `if HasAttrs(err) { for _, a := range ExtractAttrs(err) {…} }` pattern no longer produces a confusing empty-loop scenario for `Attrs()` or `FromAttrMap(nil)`.

### Added

- **`fmt.Formatter` on stacktrace errors** ([#13], [#20]) — `fmt.Sprintf("%+v", err)` now renders the captured stack in the de-facto `pkg/errors` style (newline-separated `function\n\tFile:Line` entries). `%v` and `%s` still render `Error()` unchanged.

- **`stacktrace.ExtractAll(err) [][]Frame`** ([#13], [#20]) — Returns every captured trace in the chain, outermost first. Traverses both single-`Unwrap()` and multi-`Unwrap() []error` shapes and recognises the new `Tracer` interface.

- **`stacktrace.Tracer` interface** ([#13], [#20]) — `type Tracer interface { Frames() []Frame }`. Third-party errors with pre-captured traces (e.g. from a network boundary) are now picked up by `Extract` and `ExtractAll`.

- **Configurable capture depth** ([#13], [#20]) — New `stacktrace.HereDepth(depth) Classified`, `stacktrace.WrapDepth`, `stacktrace.ClassifyDepth`, `stacktrace.ClassifyNewDepth`, exported `stacktrace.DefaultMaxDepth` (32), and `stacktrace.MaxDepth` ceiling (256) so callers can opt into deeper traces without OOM risk. Default `Here()` stays at 32 frames.

- **`stacktrace.Wrap`/`Classify`/`ClassifyNew` benchmarks and `json/` benchmarks** ([#13], [#20]) — `BenchmarkHere`, `BenchmarkWrap`, `BenchmarkExtract_{Shallow,Deep}`, `BenchmarkExtractAll`, `BenchmarkHereDepth`, plus `BenchmarkMarshal_{Simple,DeepChain,MultiError,WithStackTrace,FreshStackTrace}`, all with `b.ReportAllocs()`.

- **`errx.CarrierClassifications(err) ([]Classified, bool)`** ([#17], [#24]) — Public accessor for the classifications attached to an internal carrier produced by `Classify`/`ClassifyNew`/`Wrap`. Replaces the previous `reflect`+`unsafe` peeking inside the `json` subpackage with a compile-time-checked API.

- **`AttrList.ToKVArgs() []any`** ([#18], [#25]) — Returns the flat `[k1, v1, k2, v2, …]` form expected by non-slog loggers such as zap's `SugaredLogger.Errorw`/`Infow` and logrus' positional-pair helpers.

- **Six new `json.Option`s** ([#18], [#25]):
  - `WithStackTrace(bool)` — toggle stack-trace inclusion entirely (previously `WithMaxStackFrames(0)` meant "unlimited", not "off").
  - `WithAttributes(bool)` and `WithSentinels(bool)` — symmetrical opt-outs for PII-sensitive log pipelines.
  - `WithAttributeValueTransformer(func(key string, v any) any)` — let callers redact, hash, or rewrite individual attribute values before serialization.
  - `WithStackTraceTrimPaths(prefix)` — strip a leading path prefix from each frame's `File` to compress output.
  - `WithMaxMessageBytes(int)` — UTF-8-safe truncation cap for long `Message` strings.

- **`compat.errorWrapper` now implements `Is`/`As`** ([#18], [#25]) — Standard `error` values used as classifications stay transparent to `errors.Is`/`errors.As`.

- **`DisplayTextOrEmpty(err) string`** ([#18], [#25]) — Returns `""` when no displayable is in the chain instead of leaking `err.Error()`. A safer default for HTTP response bodies than `DisplayText`.

- **`//go:fix inline` directive on `WithAttrs`** ([#18], [#25]) — Users on Go 1.24+ can migrate every call site with `go fix ./...`; gopls and `go fix` rewrite `errx.WithAttrs(...)` to `errx.Attrs(...)` automatically.

### Changed

- **`json` subpackage no longer uses `reflect` or `unsafe`** ([#17], [#24]) — Carrier inspection now goes through the new `errx.CarrierClassifications` helper. A future rename of the unexported `carrier` type can no longer silently drop sentinel extraction.

- **`json.WithMaxDepth(0)` is now "unlimited"** ([#17], [#24]) — Non-positive depth values disable the depth limit, matching the long-standing convention of `WithMaxStackFrames`. The godoc and tests cover both `0` and negative inputs.

- **README API reference rewritten** ([#12], [#19]) — All Core Functions signatures now match the actual code (`NewSentinel`, `NewDisplayable`, `Attrs`, `FromAttrMap`, `Wrap`, `Classify`, `ClassifyNew` all return `Classified` / accept `...Classified`). The Go version policy is documented explicitly: errx tracks the two most recent stable Go releases (`oldstable`/`stable`), matching CI. The CI badge no longer pins `?branch=master`.

- **`WithAttrs` deprecation message** ([#18], [#25]) — Now includes the `since v1.1.0` marker, a concrete before/after migration example, and a pointer to `go fix ./...` for automated migration.

- **`DisplayText` godoc** ([#18], [#25]) — Carries a prominent WARNING about the fallback leaking internal context when no displayable is in the chain, with cross-references to `DisplayTextDefault` and the new `DisplayTextOrEmpty` as the production-safe paths.

- **`Attrs` godoc** ([#18], [#25]) — Adds a concrete "all-strings drift / `!BADKEY`" pitfall example showing how `Attrs("user_id", "action", "delete")` parses into one valid pair plus one `!BADKEY` entry.

### Performance

- **Lazy frame resolution in `stacktrace`** ([#13], [#20]) — `(*traced).frames()` now caches via `sync.Once`, so `Error()`, `Extract()`, and the `json` subpackage's `serializeStackTrace` share the result instead of re-walking `runtime.CallersFrames` each time. `Error()` uses `len(t.pcs)` for the frame count, avoiding symbol resolution when only the string form is needed.

- **Exact-sized PC slice** ([#13], [#20]) — `captureStack` copies the captured PCs into a slice with `cap == len`, freeing the 32-word (~256 B on 64-bit) backing array for short traces.

- **`json` allocation cleanup** ([#17], [#24]) — `config` is passed by value (no per-Marshal heap allocation); `visited` and `seenSentinels` maps are allocated lazily so single-node errors and single-sentinel chains skip the allocation entirely.

- **`errx.Classify` zero-classification short-circuit** ([#18], [#25]) — `Classify(cause)` with no classifications (after typed-nil filtering) now returns `cause` unchanged via identity, mirroring the existing behavior in `Wrap`.

- **`AttrList.String()` uses `strings.Builder`** ([#18], [#25]) — Replaces the intermediate `[]string` + `strings.Join` allocation. Output is byte-identical.

- **`compat` conversions skip empty-slice allocation** ([#18], [#25]) — When the input is empty or all-nil, conversion helpers return `nil` instead of an empty `make([]errx.Classified, 0, 0)`.

- **`stacktrace.{Wrap,Classify,ClassifyNew}` defensive slice copy** ([#18], [#25]) — Copies the caller's classifications slice before appending the captured trace so a caller spreading a slice with spare capacity (`cls := make([]Classified, 0, 4); cls = append(cls, c); stacktrace.Wrap("x", err, cls...)`) is no longer mutated. Propagated to the new `*Depth` variants too.

- **Fast-path in `nonNilClassifications`** ([#15], [#22]) — When no nil/typed-nil entries are present (the common case), returns the input slice unchanged with zero allocations. A new slice is allocated only when a nil is actually detected.

### Tests

- **Fuzz tests** ([#16], [#23]) — `FuzzAttrs`, `FuzzExtractAttrs_Chain`, `FuzzExtractAttrs_MultiError`, `FuzzMarshal` with seeded corpora.

- **Race / concurrency** ([#16], [#23]) — `TestConcurrent_ReadOnly` exercises a shared errx error from 32 goroutines × 1000 iterations of `ExtractAttrs`/`DisplayText`/`IsDisplayable`, guarding the read-only-after-construction invariant under `-race`.

- **Deep / wide / diamond stress tests** ([#16], [#23]) — 200-deep `Wrap` chain, 150-wide classification list, 1200-attribute extraction, and diamond `*attributed` dedup. All gated by `testing.Short()`.

- **Gap-fillers** ([#16], [#23]) — Pinned tests for `(*sentinel).As` parent traversal, the deprecated `WithAttrs` alias, and the exact output (including the `"(empty attribute list)"` literal) of `Attr.String`, `AttrList.String`, and `attributed.Error`.

- **Test-quality cleanups** ([#16], [#23]) — Dead `TestMultiError` removed; `TestMarshal_MixedHashableUnhashable` now asserts the full chain shape; `TestDisplayText_WithMultipleDisplayables` pinned to a deterministic chain; slog examples decoupled from `slog.TextHandler` formatting via a custom example handler; `attrTestCases()` helper deduplicates slog test setup; `BenchmarkCombined_ErrorChain` hoists `errors.New` out of the hot loop and parameterises chain depth over {1, 4, 16, 64}.

[#12]: https://github.com/go-extras/errx/issues/12
[#13]: https://github.com/go-extras/errx/issues/13
[#14]: https://github.com/go-extras/errx/issues/14
[#15]: https://github.com/go-extras/errx/issues/15
[#16]: https://github.com/go-extras/errx/issues/16
[#17]: https://github.com/go-extras/errx/issues/17
[#18]: https://github.com/go-extras/errx/issues/18
[#19]: https://github.com/go-extras/errx/pull/19
[#20]: https://github.com/go-extras/errx/pull/20
[#21]: https://github.com/go-extras/errx/pull/21
[#22]: https://github.com/go-extras/errx/pull/22
[#23]: https://github.com/go-extras/errx/pull/23
[#24]: https://github.com/go-extras/errx/pull/24
[#25]: https://github.com/go-extras/errx/pull/25
[#29]: https://github.com/go-extras/errx/issues/29
[#35]: https://github.com/go-extras/errx/pull/35

## [1.2.1] - 2026-01-31

This release fixes a critical panic that occurred when marshaling errors containing unhashable types to JSON.

### Fixed

- **Fixed panic on unhashable error types** - Resolved a panic that occurred when marshaling errors with unhashable fields (maps, slices, functions) to JSON. The panic was caused by attempting to use errors as map keys for circular reference detection. The fix uses pointer-based identity tracking with `map[uintptr]bool` instead of `map[error]bool`.

- **Fixed panic on value-based errors** - Resolved a panic that occurred when processing errors with value receivers. The previous implementation used `reflect.ValueOf(err).UnsafePointer()` which only works for pointer types. The fix extracts the data pointer directly from the error interface structure, which works for both pointer-based and value-based errors.

### Changed

- **Simplified circular reference tracking** - Replaced wrapper structs (`visitedErrors` and `visitedErrorsTracker`) with plain `map[uintptr]bool` for circular reference detection, eliminating code duplication and simplifying the implementation.

- **Extracted pointer extraction logic** - Created `internal/errptr` package with comprehensive tests to provide a shared utility for extracting pointer identities from error interfaces, eliminating duplication between `json/json.go` and `attributed.go`.

### Technical Details

The key improvements include:

1. **Pointer-based identity tracking** - Using `map[uintptr]bool` instead of `map[error]bool` to avoid panics on unhashable error types (e.g., `validation.Errors` from go-ozzo/ozzo-validation).

2. **Safe pointer extraction** - Extracting the data pointer directly from the error interface structure using `unsafe.Pointer`, which works for both pointer-based and value-based errors.

3. **uintptr safety** - The use of `uintptr` as a map key is safe for this use case because the value is only used for identity comparison during a single operation, we never dereference the pointer, and the actual error values are kept alive by the call stack during traversal.

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

[1.3.0]: https://github.com/go-extras/errx/releases/tag/v1.3.0
[1.2.1]: https://github.com/go-extras/errx/releases/tag/v1.2.1
[1.2.0]: https://github.com/go-extras/errx/releases/tag/v1.2.0
[1.1.0]: https://github.com/go-extras/errx/releases/tag/v1.1.0
[1.0.0]: https://github.com/go-extras/errx/releases/tag/v1.0.0

