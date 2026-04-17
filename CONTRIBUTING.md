# Contributing to Nexus (gore ORM)

Thank you for your interest in contributing to Nexus!

## Code of Conduct

This project adheres to a [code of conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

### Prerequisites

- Go 1.22 or later
- MySQL 8.0+ (for integration tests)
- PostgreSQL 14+ (for integration tests)

### Setting Up Development Environment

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/Nexus.git
   cd Nexus
   ```

3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/hexiefeng/Nexus.git
   ```

4. Install development tools:
   ```bash
   go install golang.org/x/tools/cmd/goimports@latest
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   go install honnef.co/go/tools/cmd/staticcheck@latest
   ```

5. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Workflow

### 1. Code Style

- Run `go fmt ./...` before committing
- Run `goimports -w .` to organize imports
- Follow [Effective Go](https://go.dev/doc/effective_go) conventions
- Run golangci-lint: `golangci-lint run ./...`

### 2. Writing Code

```bash
# Run unit tests (no external dependencies)
go test -tags=unit ./...

# Run integration tests (requires MySQL)
go test -tags=integration ./...

# Run all tests with coverage
go test -cover ./...
```

### 3. Commit Messages

Follow conventional commits format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `ci`: CI/CD changes
- `chore`: Build process or auxiliary tool changes

Examples:
```
feat(api): add transaction support

Adds Transaction() method to DbContext for explicit transaction
management with automatic rollback on error.

Fixes #123
```

### 4. Testing Requirements

- All new features must include unit tests
- Bug fixes must include a test case that reproduces the bug
- Integration tests required for database-specific functionality
- Run full test suite before submitting PR:
  ```bash
  go test -race -cover ./...
  ```

### 5. Documentation

- Update `README.md` if adding new features
- Add godoc comments for public APIs
- Update `docs/` for architecture/design changes
- Add examples in `testcode/` for new functionality

## Pull Request Process

### Before Submitting

1. Ensure all tests pass locally
2. Update documentation if needed
3. Run linting: `golangci-lint run ./...`
4. Keep commits atomic and well-described

### PR Description Template

```markdown
## Summary
Brief description of changes

## Motivation
Why is this change required? What problem does it solve?

## Changes
- List of specific changes made
- Files modified

## Testing
How was this tested?

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex logic
- [ ] Documentation updated
- [ ] Tests added/updated
- [ ] All tests pass
```

### Review Process

1. Maintainers will review within 48 hours
2. Address feedback by pushing new commits
3. Once approved, maintainers will merge

## Reporting Issues

### Bug Reports

Include:
- Go version (`go version`)
- Operating system
- Minimal reproducible example
- Full error message
- Expected vs actual behavior

### Feature Requests

- Describe the feature and its motivation
- Suggest implementation approach
- Consider backward compatibility

## License

By contributing to Nexus, you agree that your contributions will be licensed under the same license as the project.
