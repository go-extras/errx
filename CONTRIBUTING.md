# Contributing to errx

Thank you for your interest in contributing to errx! This document provides guidelines and instructions for contributing.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for all contributors.

## How to Contribute

### Reporting Bugs

Before creating a bug report:
1. Check the [existing issues](https://github.com/go-extras/errx/issues) to avoid duplicates
2. Ensure you're using the latest version of errx
3. Verify the issue is reproducible

When creating a bug report, use the bug report template and include:
- Clear description of the issue
- Minimal reproducible example
- Expected vs actual behavior
- Environment details (Go version, OS, architecture)

### Suggesting Features

Feature requests are welcome! Use the feature request template and include:
- Clear description of the proposed feature
- Motivation and use cases
- Proposed API design (if applicable)
- Whether you're willing to implement it

### Asking Questions

For usage questions or clarifications, use the question template or start a discussion.

## Development Process

### Setting Up Your Environment

1. **Fork and clone the repository:**
   ```bash
   git clone https://github.com/YOUR_USERNAME/errx.git
   cd errx
   ```

2. **Ensure you have Go 1.25+ installed:**
   ```bash
   go version
   ```

3. **Install development tools:**
   ```bash
   # Install golangci-lint
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   
   # Install govulncheck
   go install golang.org/x/vuln/cmd/govulncheck@latest
   ```

### Making Changes

1. **Create a feature branch:**
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/your-bug-fix
   ```

2. **Make your changes following our coding standards (see below)**

3. **Write or update tests:**
   - All new code must have tests
   - Maintain or improve test coverage
   - Tests should be clear and focused

4. **Run the test suite:**
   ```bash
   # Run all tests
   go test ./...
   
   # Run with race detection
   go test -race ./...
   
   # Check coverage
   go test -cover ./...
   ```

5. **Run linters:**
   ```bash
   # Format code
   go fmt ./...
   
   # Run golangci-lint
   golangci-lint run
   
   # Run go vet
   go vet ./...
   ```

6. **Run security checks:**
   ```bash
   govulncheck ./...
   ```

### Submitting Changes

1. **Commit your changes:**
   ```bash
   git add .
   git commit -m "Brief description of changes"
   ```
   
   Write clear commit messages:
   - Use present tense ("Add feature" not "Added feature")
   - Be concise but descriptive
   - Reference issues when applicable (#123)

2. **Push to your fork:**
   ```bash
   git push origin feature/your-feature-name
   ```

3. **Create a Pull Request:**
   - Use the PR template
   - Fill out all relevant sections
   - Link related issues
   - Ensure all CI checks pass

## Coding Standards

### Go Style Guidelines

- Follow standard Go conventions and idioms
- Use `gofmt` for formatting
- Follow the [Effective Go](https://golang.org/doc/effective_go) guidelines
- Keep functions focused and concise (max 240 lines, 160 statements)
- Limit cyclomatic complexity (max 21)
- Limit cognitive complexity (max 30)

### Documentation

- All exported functions, types, and constants must have godoc comments
- Comments should start with the name of the thing being described
- Include examples for complex functionality
- Update README.md for user-facing changes
- Add entries to CHANGELOG.md for all changes

### Testing Requirements

- **Unit tests:** All new code must have unit tests
- **Test coverage:** Maintain or improve existing coverage (aim for >90%)
- **Table-driven tests:** Use table-driven tests for multiple scenarios
- **Edge cases:** Test nil values, empty inputs, and boundary conditions
- **Examples:** Add example tests for new public APIs
- **Benchmarks:** Add benchmarks for performance-critical code

Example test structure:
```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"case1", "input1", "output1"},
        {"case2", "input2", "output2"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Feature(tt.input)
            if result != tt.expected {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### API Design Principles

When proposing new APIs, consider:

1. **Simplicity:** Keep the API surface small and focused
2. **Consistency:** Follow existing patterns in the codebase
3. **Composability:** Features should work well together
4. **Backwards compatibility:** Avoid breaking changes in v1.x
5. **Zero dependencies:** Core package must remain dependency-free
6. **Performance:** Minimize allocations and overhead
7. **Go idioms:** Work naturally with `errors.Is`, `errors.As`, etc.

### Breaking Changes

Breaking changes are **not allowed** in v1.x releases. If you believe a breaking change is necessary:

1. Open an issue to discuss the change first
2. Explain why it's necessary and what it improves
3. Propose a migration path for existing users
4. Consider if it can be done in a backwards-compatible way

Breaking changes will only be considered for v2.0.0 or later.

## Pull Request Process

1. **Before submitting:**
   - Ensure all tests pass locally
   - Run linters and fix all issues
   - Update documentation
   - Add CHANGELOG entry
   - Rebase on latest master if needed

2. **PR review:**
   - Maintainers will review your PR
   - Address feedback and requested changes
   - Keep the PR focused on a single concern
   - Be patient and respectful during review

3. **After approval:**
   - Maintainers will merge your PR
   - Your contribution will be included in the next release

## Release Process

Releases are managed by maintainers and follow semantic versioning:

- **MAJOR** (v2.0.0): Breaking changes
- **MINOR** (v1.1.0): New features, backwards compatible
- **PATCH** (v1.0.1): Bug fixes, backwards compatible

## Getting Help

- **Questions:** Open a question issue or discussion
- **Bugs:** Open a bug report with reproduction steps
- **Features:** Open a feature request with use cases
- **Security:** Email security issues privately to maintainers

## Recognition

Contributors will be recognized in:
- Git commit history
- GitHub contributors page
- Release notes (for significant contributions)

## License

By contributing to errx, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to errx! 🎉

