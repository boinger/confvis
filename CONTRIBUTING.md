# Contributing to confvis

Thank you for your interest in contributing to confvis.

## Development Setup

1. Clone the repository:

```bash
git clone https://github.com/boinger/confvis.git
cd confvis
```

2. Ensure you have Go 1.21+ installed:

```bash
go version
```

3. Install dependencies:

```bash
go mod download
```

4. Build and test:

```bash
go build ./...
go test ./...
```

## Running Tests

Run the full test suite:

```bash
go test ./...
```

Run with verbose output:

```bash
go test -v ./...
```

Run with coverage:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Linting

We use golangci-lint. Install it and run:

```bash
golangci-lint run ./...
```

All code must pass linting with zero issues before submission.

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Keep functions focused and small
- Add package comments for public packages
- Test public APIs

## Pull Request Guidelines

1. Fork the repository
2. Create a feature branch from `main`
3. Make your changes
4. Ensure tests pass and linting is clean
5. Write clear commit messages
6. Submit a pull request

### Commit Messages

- Use present tense ("Add feature" not "Added feature")
- Keep the first line under 72 characters
- Reference issues when applicable

### PR Checklist

- [ ] Tests pass (`go test ./...`)
- [ ] Linting passes (`golangci-lint run ./...`)
- [ ] New features have tests
- [ ] Documentation updated if needed

## Project Structure

```
confvis/
├── cmd/confvis/        # Main entry point
├── internal/
│   ├── cli/            # Command-line interface (cobra)
│   ├── confidence/     # JSON parsing and types
│   ├── gauge/          # SVG gauge generation
│   └── dashboard/      # HTML dashboard generation
├── testdata/           # Test fixtures
├── docs/               # Documentation
└── examples/           # Integration examples
```

## Questions?

Open an issue for questions or discussion.
