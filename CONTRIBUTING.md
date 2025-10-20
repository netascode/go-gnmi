# Contributing to go-gnmi

Thank you for your interest in contributing to go-gnmi! This document provides guidelines for contributing to the project.

## Code of Conduct

This project adheres to a [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates. Include:

- Clear title and description with steps to reproduce
- Expected vs actual behavior with code samples
- Go version, OS information, device type/version
- gNMI capabilities supported by the device

### Suggesting Features

Feature suggestions are welcome! Please:

- Check existing feature requests to avoid duplicates
- Provide clear use cases and explain why it's useful
- Consider gNMI specification compliance and device compatibility
- Consider implementation complexity and security implications

### Pull Requests

1. Fork the repository and create a branch from `main`
2. Make your changes following the coding guidelines below
3. Add tests for any new functionality
4. Add SPDX headers to all new Go files (see Headers section)
5. Ensure all checks pass: `make test`, `make lint`, `make security`
6. Update documentation as needed
7. Write clear commit messages following conventional commits
8. Submit a pull request

## Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR-USERNAME/go-gnmi.git
cd go-gnmi

# Install dependencies
go mod download

# Run all checks
make verify
```

**Prerequisites**: Go 1.24+, golangci-lint, Make (optional)

## Coding Guidelines

### Go Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go) and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `go fmt` for formatting
- Write clear comments for exported functions
- Follow gNMI terminology from the specification

### File Headers

All Go source files must include SPDX license identifier and copyright notice:

```go
// SPDX-License-Identifier: MPL-2.0
// Copyright (c) 2025 Daniel Schmidt

package gnmi
```

Use provided scripts: `bash scripts/add-headers.sh` or `bash scripts/check-headers.sh`

### Testing

- Write unit tests for all new functionality
- Use table-driven tests with edge cases and error conditions
- Test with mock gNMI targets for unit tests
- Add integration tests for device compatibility
- Run race detector: `go test -race`

### Security Testing

Test all code for security vulnerabilities:

- **Lock management**: Ensure defer unlock is always used
- **Input validation**: Test malformed JSON, oversized payloads, invalid paths
- **Error handling**: Verify retry logic doesn't leak credentials or sensitive data
- **TLS security**: Verify certificate verification enforced by default
- **Concurrent access**: Test thread safety with race detector

### Documentation

- Document all exported functions, types, and constants
- Include usage examples in godoc comments
- Update README.md for significant changes
- Document gNMI capabilities required and security implications

## gNMI Specification Compliance

This project aims for full gNMI specification compliance:

- **gNMI Specification**: All operations (Get, Set, Capabilities) must comply
- **gRPC Transport**: Transport layer compliance required

When implementing gNMI operations, reference relevant specification sections in code comments.

## Review Process

All PRs must meet these requirements:

1. At least one approval
2. CI passes (tests, lint, coverage, security)
3. Code follows style guidelines
4. SPDX headers present in all Go files
5. Documentation updated
6. Security implications considered
7. Breaking changes discussed (requires major version bump)

## Questions?

- Open an issue for questions
- Start a discussion in GitHub Discussions
- Reach out to maintainers

## License

By contributing, you agree that your contributions will be licensed under the Mozilla Public License Version 2.0.
