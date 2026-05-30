# errx

Rich, idiomatic error handling for Go — classification you can check with `errors.Is`, user-safe messages you can show, and structured attributes you can log, all in one composable call.

[![CI](https://github.com/go-extras/errx/actions/workflows/go-test.yml/badge.svg)](https://github.com/go-extras/errx/actions/workflows/go-test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/go-extras/errx.svg)](https://pkg.go.dev/github.com/go-extras/errx)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-extras/errx)](https://goreportcard.com/report/github.com/go-extras/errx)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.25-00ADD8?logo=go)](https://go.dev/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

```go
return errx.Wrap("authentication failed", cause,
    errx.NewDisplayable("Invalid username or password"), // safe to show the user
    errx.Attrs("username", username, "reason", "user_not_found"), // safe to log
    ErrUnauthorized, // safe to branch on with errors.Is
)
```

One call attaches everything. The classification, the user-facing message, and the
structured attributes all travel with the error and stay extractable anywhere up the
stack — without ever leaking into `err.Error()`.

## Why errx

Most Go error libraries do one thing well: `pkg/errors` wraps, `eris` serializes,
`errorx` classifies, the `multierr` family aggregates. errx is the only one that
combines **hierarchical `errors.Is` classification**, **user-safe displayable
messages**, and **structured attributes** in a **zero-dependency core** — with stack
traces and JSON as opt-in subpackages you only pay for when you import them.

| Capability | **errx** | [pkg/errors][pkgerrors] | [cockroachdb/errors][crdb] | [joomcode/errorx][errorx] | [morikuni/failure][failure] | [rotisserie/eris][eris] |
|---|:--:|:--:|:--:|:--:|:--:|:--:|
| Hierarchical classification (`errors.Is`) | ✅ sentinels | ❌ | ❌ | ✅ traits | ⚠️ codes | ❌ |
| User-safe display messages | ✅ | ❌ | ⚠️ hints | ❌ | ⚠️ | ❌ |
| Structured attributes (key/value) | ✅ | ❌ | ⚠️ strings | ✅ props | ✅ context | ❌ |
| Zero-dependency / pay-for-what-you-use core | ✅ | ✅ | ❌ heavy | ⚠️ traces in core | ✅ | ⚠️ traces forced |
| `slog` integration | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| JSON serialization | ✅ | ❌ | ⚠️ protobuf | ❌ | ❌ | ✅ |
| Multi-error root (`errors.Join`-style) | ✅ `Join`<sup>2</sup> | ❌ | ✅ `Join` | ❌ | ❌ | ❌ |
| Cross-service round-trip (decode) | ⚠️ serialize-only<sup>1</sup> | ❌ | ✅ wire codec | ❌ | ❌ | ⚠️ one-way |
| Maintenance status | ✅ active | ⚠️ [unmaintained](#migrating-from-pkgerrors) | ✅ active | ✅ active | ⚠️ since 2022 | ⚠️ pre-1.0<sup>3</sup> |

<sup>1</sup> errx serializes errors to JSON (including multi-error branches) but does not
yet decode them back into an error value on a receiving service. If wire-portable errors
across service boundaries are your primary need, [cockroachdb/errors][crdb] is
purpose-built for that. (Tracked for analysis in [#37](https://github.com/go-extras/errx/issues/37).)

<sup>2</sup> The standard library's `errors.Join` (Go 1.20+) already provides an
aggregation root for free; `errx.Join` adds the ability to *classify and annotate the
group* (`errx.Classify(errx.Join(…), ErrBatch, errx.Attrs(…))`) — see
[Aggregating errors](#aggregating-errors). For pure aggregation, reach for stdlib
`errors.Join` or the [multierr][umultierr] family.

<sup>3</sup> [eris][eris] is actively maintained (CI/dependabot run), but its last
functional release is v0.5.4 (2022) and it remains pre-1.0.

### When another library is the better fit

We'd rather be honest than oversell — pick the right tool:

- **[cockroachdb/errors][crdb]** when you need errors encoded/decoded across services
  over the wire, with redaction designed for Sentry. It's heavier by design because that
  problem is hard.
- **[go.uber.org/multierr][umultierr] / [hashicorp/go-multierror][hmultierr]** when your
  problem is *aggregating* many errors into one. That's an orthogonal niche — pair it
  with errx, don't replace it (and see [`errx.Join`](#aggregating-errors) below if you
  want aggregation that carries classifications).
- **[joomcode/errorx][errorx]** — the closest library to errx — when you want stack
  traces captured automatically in the core with no opt-in step. errx differs by keeping
  traces opt-in, by offering a dedicated *user-safe message* channel errorx lacks, and by
  leaning on stdlib `errors.Is` instead of a bespoke trait-matching API.
- **stdlib `errors` + `fmt.Errorf("%w", …)`** when all you need is wrapping plus
  `errors.Is`. errx stays fully compatible, so you can reach for it later without a
  rewrite.

## Requirements

- Go 1.25+ (tracks the two most recent stable Go releases per the Go team's support policy)

errx follows the [Go team's release policy](https://go.dev/doc/devel/release) and
officially supports the two most recent stable Go releases. CI verifies the library on
both the current `stable` and `oldstable` toolchains; older versions may work but are not
tested. The `go` directive in `go.mod` declares the minimum language version required to
build the module.

## Installation

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

// Define classification sentinels once, at package level.
var (
    ErrNotFound = errx.NewSentinel("resource not found")
    ErrInvalid  = errx.NewSentinel("invalid input")
)

func processOrder(orderID string, cause error) error {
    // Attach context, a user-safe message, attributes, and a classification —
    // all in one idiomatic variadic call.
    return errx.Wrap("failed to process order", cause,
        errx.NewDisplayable("Order not found"),
        errx.Attrs("order_id", orderID),
        ErrNotFound,
    )
}

func main() {
    err := processOrder("12345", errors.New("no rows in result set"))

    // Branch on the classification — the sentinel text never pollutes the message.
    if errors.Is(err, ErrNotFound) {
        fmt.Println("resource was not found")
    }

    // Extract the user-safe message (empty string if none — never leaks internals).
    if msg := errx.DisplayTextOrEmpty(err); msg != "" {
        fmt.Println("user message:", msg) // "Order not found"
    }

    // Pull structured attributes for logging.
    attrs := errx.ExtractAttrs(err) // [{order_id 12345}]

    // The full internal chain is still there for your logs.
    fmt.Println("full error:", err) // "failed to process order: no rows in result set"
    _ = attrs
}
```

For creating a brand-new error and classifying it in one step (no `cause` to wrap), use
`ClassifyNew`:

```go
err := errx.ClassifyNew("user record missing", ErrNotFound, ErrDatabase)
// equivalent to: errx.Classify(errors.New("user record missing"), ErrNotFound, ErrDatabase)
```

## Core Concepts

errx gives you three building blocks. Each is a `Classified` value, so they all compose
as variadic arguments to `Wrap`, `Classify`, and `ClassifyNew`.

### Classification sentinels

Sentinels identify error *types* without adding text to the error message. Check them with
the standard `errors.Is`.

```go
var (
    ErrDatabase   = errx.NewSentinel("database error")
    ErrValidation = errx.NewSentinel("validation error")
)

func fetchData() error {
    if err := db.Execute("SELECT …"); err != nil {
        return errx.Classify(err, ErrDatabase) // classify, no extra context text
    }
    return nil
}

if errors.Is(err, ErrDatabase) {
    // handle database error
}
```

**`Classify` vs `Wrap`:** use `Classify` when the message is already clear and you only
need to tag it; use `Wrap` when you also want to add context about where/why it happened.

#### Hierarchical sentinels

Pass parent sentinels to build a taxonomy. A child matches itself *and* every parent.

```go
var (
    ErrRetryable = errx.NewSentinel("retryable")
    ErrPermanent = errx.NewSentinel("permanent")

    ErrTimeout   = errx.NewSentinel("timeout", ErrRetryable)
    ErrRateLimit = errx.NewSentinel("rate limit", ErrRetryable)
    ErrNotFound  = errx.NewSentinel("not found", ErrPermanent)
)

switch {
case errors.Is(err, ErrTimeout):
    return retryWithBackoff() // specific
case errors.Is(err, ErrRetryable):
    return retry()            // any retryable error
case errors.Is(err, ErrPermanent):
    return err                // don't retry
}
```

Sentinels can have **multiple parents** for multi-dimensional classification:

```go
var (
    ErrRetryable       = errx.NewSentinel("retryable")
    ErrDatabase        = errx.NewSentinel("database")
    ErrDatabaseTimeout = errx.NewSentinel("database timeout", ErrDatabase, ErrRetryable)
)

errors.Is(err, ErrDatabase)  // true for any database error
errors.Is(err, ErrRetryable) // true for any retryable error
```

### Displayable messages

Keep user-safe messages separate from internal detail, and extract them anywhere up the
chain.

```go
func validateEmail(email string) error {
    if !strings.Contains(email, "@") {
        return errx.NewDisplayable("Please enter a valid email address")
    }
    return nil
}
```

Three ways to read it back, by how cautious you need to be:

```go
errx.IsDisplayable(err)                  // does the chain carry a user-safe message?
errx.DisplayText(err)                     // the message, or err.Error() on a miss
errx.DisplayTextDefault(err, "Oops")      // the message, or your fallback on a miss
errx.DisplayTextOrEmpty(err)              // the message, or "" on a miss (never leaks internals)
```

> **Security note:** `DisplayText` falls back to the *full* `err.Error()` when no
> displayable message is present, which can leak internal context (DB errors, paths) to
> users. For untrusted, user-facing surfaces prefer `DisplayTextOrEmpty` (pick a localized
> fallback at the response layer) or `DisplayTextDefault`.

### Structured attributes

Attach key/value metadata for logging without cluttering the message.

```go
func processPayment(userID string, amount float64) error {
    if amount < 0 {
        return errx.ClassifyNew("payment validation failed",
            errx.NewDisplayable("Amount must be positive"),
            errx.Attrs("user_id", userID, "amount", amount, "currency", "USD"),
            ErrValidation,
        )
    }
    return nil
}

attrs := errx.ExtractAttrs(err) // AttrList: [{user_id …}, {amount …}, …]
```

#### Integration with `slog`

`AttrList` converts directly for structured logging:

```go
attrs := errx.ExtractAttrs(err)

// Most efficient — use with LogAttrs:
logger.LogAttrs(ctx, slog.LevelError, "operation failed", attrs.ToSlogAttrs()...)

// Convenient — use with Error/Info/Warn:
logger.Error("operation failed", attrs.ToSlogArgs()...)
```

For loggers that take flat `key, value, …` variadics (zap's `SugaredLogger`, logrus, etc.)
use `ToKVArgs`:

```go
sugar.Errorw("operation failed", attrs.ToKVArgs()...)
```

## The idiomatic combined pattern

The whole point of errx: classification, a user-safe message, and attributes attached in a
single call. There is **no builder and no intermediate variables** — `NewDisplayable`,
`Attrs`, and sentinels are all `Classified`, so they slot straight into the variadic.

```go
func authenticateUser(username, password string) error {
    user, err := findUser(username)
    if err != nil {
        // We have a cause to wrap → Wrap.
        return errx.Wrap("authentication failed", err,
            errx.NewDisplayable("Invalid username or password"),
            errx.Attrs("username", username, "reason", "user_not_found"),
            ErrUnauthorized,
        )
    }

    if !user.CheckPassword(password) {
        // No underlying error → ClassifyNew (Wrap would return nil on a nil cause).
        return errx.ClassifyNew("password check failed",
            errx.NewDisplayable("Invalid username or password"),
            errx.Attrs("username", username, "reason", "wrong_password"),
            ErrUnauthorized,
        )
    }
    return nil
}
```

**The one rule to remember:**

| You have… | Use | Because |
|---|---|---|
| an existing `cause` to wrap | `errx.Wrap("ctx", cause, …)` | adds context over the cause |
| no cause — you're creating the error | `errx.ClassifyNew("ctx", …)` | `Wrap` returns `nil` when `cause == nil` |

Reading it all back:

```go
errors.Is(err, ErrUnauthorized)  // true
errx.DisplayTextOrEmpty(err)     // "Invalid username or password"
err.Error()                       // "authentication failed: <internal cause>"
errx.ExtractAttrs(err)            // [{username …}, {reason …}]
```

## Aggregating errors

errx works transparently with multi-error trees. The attribute and displayable helpers
traverse `errors.Join` branches out of the box, so you can compose with the standard
library directly:

```go
err := errx.Classify(errors.Join(item1Err, item2Err), ErrBatch)
errors.Is(err, ErrBatch)   // true
errx.ExtractAttrs(err)     // attributes from every branch
```

For convenience, `errx.Join` is an aggregation root that mirrors `errors.Join` (it drops
nil entries and returns `nil` if every entry is nil) but lives in the errx namespace so
you can classify the group with the existing API:

```go
var ErrBatch = errx.NewSentinel("batch failed")

err := errx.Classify(errx.Join(item1Err, item2Err), ErrBatch, errx.Attrs("count", 2))
```

`errx.Join` is intentionally orthogonal to classification — keep `Join` for aggregation and
reach for `Classify`/`Wrap` to tag the result, exactly as above. The JSON subpackage
serializes joined members into the `causes` field automatically.

## Optional subpackages

The core stays zero-dependency. These subpackages add capabilities only when you import
them.

### Stack traces — `stacktrace`

```go
import "github.com/go-extras/errx/stacktrace"

// Per-error opt-in:
err := errx.Wrap("operation failed", cause, ErrNotFound, stacktrace.Here())

// Or capture automatically:
err := stacktrace.Wrap("operation failed", cause, ErrNotFound)
err := stacktrace.ClassifyNew("user record missing", ErrNotFound)

// Extract frames (the primary access path):
for _, frame := range stacktrace.Extract(err) {
    fmt.Printf("%s:%d %s\n", frame.File, frame.Line, frame.Function)
}

// Or print pkg/errors-style with %+v:
fmt.Printf("%+v\n", err)
// operation failed: user record missing
// main.fetchUser
//     /app/user.go:42
// main.main
//     /app/main.go:17
```

Traces are **opt-in** so the core package stays dependency-free and fast, and they compose
with every other feature. Errors that carry a trace implement `fmt.Formatter`: `%+v` prints
the message followed by the captured frames (de-facto `pkg/errors` style), while `%v` and
`%s` print the message only. This makes errx a **drop-in target for code migrating off
`pkg/errors`**, where `log.Printf("%+v", err)` is the standard way to surface stack traces —
no logging changes required. Errors without a trace (plain `errx.Wrap`/`Classify`) print the
message only under `%+v`, exactly as before.

See the [stacktrace docs](https://pkg.go.dev/github.com/go-extras/errx/stacktrace).

### JSON serialization — `json`

```go
import errxjson "github.com/go-extras/errx/json"

jsonBytes, _ := errxjson.Marshal(err)
pretty, _   := errxjson.MarshalIndent(err, "", "  ")
```

It serializes the whole picture — message, display text, sentinels, attributes, stack
frames, and multi-error `causes` — using only `encoding/json`, with circular-reference and
depth protection. Options let you shape and **redact** the output, which matters for
PII-sensitive pipelines:

```go
// Redact sensitive attribute values during serialization:
redact := func(key string, v any) any {
    if key == "password" || key == "token" {
        return "<redacted>"
    }
    return v
}
jsonBytes, _ := errxjson.Marshal(err,
    errxjson.WithAttributeValueTransformer(redact),
    errxjson.WithMaxDepth(16),            // bound chain depth
    errxjson.WithMaxStackFrames(10),      // bound stack frames
    errxjson.WithStackTraceTrimTop(2),    // drop innermost (top) frames
    errxjson.WithStackFrameFilter(        // keep only frames you care about
        func(f stacktrace.Frame) bool { return !strings.HasPrefix(f.Function, "runtime.") }),
    errxjson.WithStackTrace(false),       // drop stack traces entirely
    errxjson.WithAttributes(false),       // drop attributes entirely
    errxjson.WithSentinels(false),        // drop sentinel text
    errxjson.WithStackTraceTrimPaths("/build/"), // strip build-host path prefixes
    errxjson.WithMaxMessageBytes(1024),   // UTF-8-safe message truncation
)
```

See the [json docs](https://pkg.go.dev/github.com/go-extras/errx/json).

### Standard-error compatibility — `compat`

The `compat` subpackage mirrors the core API but accepts plain `error` values instead of
`Classified`, easing migration and third-party integration.

```go
import "github.com/go-extras/errx/compat"

var ErrNotFound = errors.New("not found") // plain stdlib errors

err := compat.Wrap("failed to fetch user", cause, ErrNotFound)
errors.Is(err, ErrNotFound) // true
```

**Tradeoffs:** more flexible and easier to migrate to, but less type-safe (you can pass a
non-classification error by accident) and slightly heavier. Use `compat` when migrating or
integrating; use the core package for new, type-safe code. See the
[compat docs](https://pkg.go.dev/github.com/go-extras/errx/compat).

## Migrating from `pkg/errors`

[`pkg/errors`][pkgerrors] is **unmaintained** — and not even formally archived, which is
arguably worse: the repo still accepts issues and pull requests, but the last release was
v0.9.1 in January 2020, dozens of issues and PRs sit unanswered, its own maintainer has
publicly asked for someone else to take it over (explaining it had gone years without upkeep
and that he no longer uses it, [pkg/errors#245][pkgerrors-245]), and its test suite no longer
passes cleanly on current Go toolchains ([pkg/errors#249][pkgerrors-249]). The
wrap-plus-stack-trace niche it filled is exactly what errx covers, so moving off it is
low-risk — and there's a tool that does the rewrite for you.

**[errx-migrate][errx-migrate]** is a type-aware AST rewriter that converts a
`pkg/errors`-based codebase to errx **safely and idempotently**. It uses full type
information to resolve which `errors` identifier actually refers to `pkg/errors` (handling
aliases, dot-imports, and shadowing), so it never rewrites stdlib `errors` calls by mistake,
and it leaves `gofmt`-clean output with explicit `// TODO(errx-migrate)` markers wherever a
rewrite can't be proven safe.

```bash
go install github.com/go-extras/errx-migrate/cmd/errx-migrate@latest

errx-migrate -diff ./...          # preview the migration as a unified diff (no writes)
errx-migrate -w ./...             # apply it in place (stack-preserving, the default)
errx-migrate -w -no-stack ./...   # lean mode: zero-dependency core + stdlib errors/fmt
```

By default it runs in **stack-preserving** mode, mapping the stack-capturing calls onto the
[`errx/stacktrace`](#stack-traces--stacktrace) subpackage so runtime behavior — including
`%+v` stack rendering — is unchanged. Pass `-no-stack` for **lean** mode, which drops
implicit stack capture and targets the zero-dependency errx core plus stdlib `errors`/`fmt`
(an explicit `errors.WithStack` is still honored with a real stack).

| `pkg/errors` | stack-preserving (default) | lean (`-no-stack`) |
|---|---|---|
| `errors.New` / `errors.Errorf` | `stacktrace.ClassifyNew(…)` | stdlib `errors.New` / `fmt.Errorf` |
| `errors.Wrap(err, msg)` | `stacktrace.Wrap(msg, err)` | `errx.Wrap(msg, err)` |
| `errors.Wrapf(err, f, …)` | `stacktrace.Wrap(fmt.Sprintf(f, …), err)` | `errx.Wrap(fmt.Sprintf(f, …), err)` |
| `errors.WithMessage(err, msg)` | `errx.Wrap(msg, err)` | `errx.Wrap(msg, err)` |
| `errors.WithStack(err)` | `stacktrace.Classify(err)` | `stacktrace.Classify(err)` |
| `errors.Is` / `As` / `Unwrap` | stdlib `errors.*` | stdlib `errors.*` |

Note the **argument-order swap** — `pkg/errors.Wrap(err, msg)` becomes `errx.Wrap(msg, err)`
— which the rewriter handles for you; nil semantics match (`Wrap(nil, …)` ⇒ `nil` on both
sides). The full mapping, the conservative fallbacks, and an analyzer mode that runs under
`go vet` / `gopls` / golangci-lint are documented in the
[errx-migrate README][errx-migrate].

Prefer to migrate by hand, incrementally? Start with the
[`compat`](#standard-error-compatibility--compat) subpackage so you can pass the plain
sentinel `error` values you already have, then adopt the type-safe core where it helps.

## Complete example: an HTTP handler

```go
package main

import (
    "errors"
    "log/slog"
    "net/http"

    "github.com/go-extras/errx"
)

var (
    ErrClient       = errx.NewSentinel("client error")
    ErrServer       = errx.NewSentinel("server error")
    ErrNotFound     = errx.NewSentinel("not found", ErrClient)
    ErrUnauthorized = errx.NewSentinel("unauthorized", ErrClient)
    ErrDatabase     = errx.NewSentinel("database", ErrServer)
)

// Data layer: classify + attach attributes in one call.
func findUserInDB(userID string) error {
    cause := errors.New("connection timeout")
    return errx.Wrap("query failed", cause,
        errx.Attrs("user_id", userID, "operation", "select"),
        ErrDatabase,
    )
}

// API layer: add the user-facing message and a precise classification.
func getUser(userID string) error {
    if err := findUserInDB(userID); err != nil {
        return errx.Wrap("failed to get user", err,
            errx.NewDisplayable("User not found"),
            ErrNotFound,
        )
    }
    return nil
}

func handleGetUser(w http.ResponseWriter, r *http.Request, logger *slog.Logger) {
    err := getUser(r.PathValue("id"))
    if err == nil {
        return
    }

    // Map classification → HTTP status, specific before general.
    status := http.StatusInternalServerError
    switch {
    case errors.Is(err, ErrNotFound):
        status = http.StatusNotFound
    case errors.Is(err, ErrUnauthorized):
        status = http.StatusUnauthorized
    case errors.Is(err, ErrClient):
        status = http.StatusBadRequest
    }

    // Log the full internal error with structured attributes.
    logger.LogAttrs(r.Context(), slog.LevelError, "request failed",
        errx.ExtractAttrs(err).ToSlogAttrs()...)

    // Send only the user-safe message — fall back without leaking internals.
    msg := errx.DisplayTextOrEmpty(err)
    if msg == "" {
        msg = "An error occurred"
    }
    http.Error(w, msg, status)
}
```

## Best practices

1. **Define sentinels at package level** — they're identity-based; create them once.
2. **Add displayable messages at domain boundaries** — where you validate input or detect
   user-relevant conditions.
3. **`Classify` to preserve a clear message; `Wrap` to add context.**
4. **`Wrap` when you have a cause, `ClassifyNew` when you're creating the error** (see the
   table above).
5. **Check sentinels specific → general** in `switch` statements.
6. **Use `DisplayTextOrEmpty` on user-facing surfaces** to avoid leaking internal detail.
7. **Log attributes with `slog`** via `ToSlogAttrs`/`ToSlogArgs`/`ToKVArgs`.

## API reference

Full docs: [pkg.go.dev/github.com/go-extras/errx](https://pkg.go.dev/github.com/go-extras/errx).
The core package uses the type-safe `Classified` interface; see [`compat`](#standard-error-compatibility--compat)
when you need to pass plain `error` values.

### Creation

- **`NewSentinel(text string, parents ...Classified) Classified`** — a classification
  sentinel, optionally hierarchical.
- **`NewDisplayable(message string) Classified`** — a user-safe message.
- **`Attrs(attrs ...any) Classified`** — structured attributes from alternating key/value
  pairs, `Attr` structs, or `[]Attr`/`AttrList` slices.
- **`FromAttrMap(attrs AttrMap) Classified`** — attributes from a map (ordering is
  non-deterministic).

### Wrapping & aggregation

- **`Wrap(text string, cause error, classifications ...Classified) error`** — add context
  and classifications. Returns `nil` if `cause` is `nil`.
- **`Classify(cause error, classifications ...Classified) error`** — attach classifications
  with no extra context. Returns `nil` if `cause` is `nil`.
- **`ClassifyNew(text string, classifications ...Classified) error`** — create and classify
  in one step.
- **`Join(errs ...error) error`** — aggregate multiple errors into one (`errors.Join`-style,
  with `Unwrap() []error`). Drops nil entries; returns `nil` if all are nil.

### Inspection

- **`IsDisplayable(err error) bool`** / **`DisplayText(err error) string`**
- **`DisplayTextDefault(err error, def string) string`** — fallback on a miss.
- **`DisplayTextOrEmpty(err error) string`** — `""` on a miss; safe for user surfaces.
- **`HasAttrs(err error) bool`** / **`ExtractAttrs(err error) AttrList`**
- **`(AttrList).ToSlogAttrs() []slog.Attr`** / **`(AttrList).ToSlogArgs() []any`** /
  **`(AttrList).ToKVArgs() []any`**
- **`CarrierClassifications(err error) ([]Classified, bool)`** — inspect the classifications
  attached at one level of the chain (used by the json subpackage; treat the slice as
  read-only).

## Comparison with standard errors

```go
// Standard approach
err := errors.New("user not found")
wrapped := fmt.Errorf("failed to get user: %w", err)

// errx approach — one call, three extra capabilities
err := errx.Wrap("failed to get user", cause,
    errx.NewDisplayable("User not found"),
    errx.Attrs("user_id", id),
    ErrNotFound,
)
// errors.Is(err, ErrNotFound) • errx.DisplayTextOrEmpty(err) •
// errx.ExtractAttrs(err) • err.Error() still has full internal context
```

Migrating from `pkg/errors`? [errx-migrate](#migrating-from-pkgerrors) automates the whole
rewrite. Coming from stdlib — or migrating by hand — start with the
[`compat`](#standard-error-compatibility--compat) subpackage so you can pass the plain
sentinel `error` values you already have, then adopt the type-safe core where it helps.

## Use cases

- **API error handling** — separate internal errors from user-facing messages.
- **Microservices** — classify errors for HTTP status mapping.
- **Structured logging** — attach rich context for debugging.
- **Error monitoring** — track categories and patterns.
- **Domain-driven design** — error taxonomies that match your domain.

## Testing

```bash
go test ./...            # all tests
go test -race ./...      # with race detection
go test -cover ./...     # with coverage
```

## Contributing

Contributions are welcome! Please open issues for bugs, feature requests, or questions, and
submit pull requests with clear descriptions, tests, and the existing code style.

## License

MIT © 2026 Denis Voytyuk — see [LICENSE](LICENSE) for details.

## Acknowledgments

This library builds on Go's standard `errors` package and the Go community's error-handling
best practices.

<!-- alternative libraries -->
[pkgerrors]: https://github.com/pkg/errors
[pkgerrors-245]: https://github.com/pkg/errors/issues/245
[pkgerrors-249]: https://github.com/pkg/errors/issues/249
[errx-migrate]: https://github.com/go-extras/errx-migrate
[crdb]: https://github.com/cockroachdb/errors
[errorx]: https://github.com/joomcode/errorx
[failure]: https://github.com/morikuni/failure
[eris]: https://github.com/rotisserie/eris
[umultierr]: https://github.com/uber-go/multierr
[hmultierr]: https://github.com/hashicorp/go-multierror
