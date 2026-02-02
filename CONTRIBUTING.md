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

3. Install development tools:

```bash
make install-tools
```

This installs:
- `golangci-lint` - Linting
- `@commitlint/cli` - Commit message validation (requires npm)
- Pre-commit hooks

4. Build and test:

```bash
make build
make test
```

## Running Tests

Run the full test suite:

```bash
make test
```

Run with verbose output:

```bash
make test-verbose
```

Run with coverage and check threshold:

```bash
make coverage
```

Generate HTML coverage report:

```bash
make coverage-html
```

## Coverage Requirements

All contributions must maintain **80% or higher code coverage**. The CI pipeline will fail if coverage drops below this threshold.

Before submitting a PR, verify coverage locally:

```bash
make coverage
```

## Linting

We use golangci-lint with strict settings. Run:

```bash
make lint
```

All code must pass linting with **zero issues** (including info-level) before submission. No exceptions.

## Pre-commit Hooks

We use pre-commit to enforce quality standards. After running `make install-tools`, hooks are installed automatically.

To run all checks manually:

```bash
make pre-commit
```

Or using pre-commit directly:

```bash
pre-commit run --all-files
```

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Keep functions focused and small
- Add package comments for public packages
- Test public APIs

## Commit Message Format

We follow [Conventional Commits](https://www.conventionalcommits.org/). Every commit message must follow this format:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Types

| Type | Description |
|------|-------------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `style` | Code style (formatting, semicolons, etc) |
| `refactor` | Code refactoring (no functional changes) |
| `perf` | Performance improvement |
| `test` | Adding or updating tests |
| `build` | Build system or dependencies |
| `ci` | CI/CD configuration |
| `chore` | Maintenance tasks |
| `revert` | Reverting previous commit |

### Examples

```bash
feat(gauge): add sparkline visualization support
fix(cli): correct JSON output formatting for empty factors
docs: update README with new fetch sources
test(sources): add mock server tests for codecov client
build(deps): bump github.com/spf13/cobra to v1.10.2
ci: add SonarCloud analysis to workflow
```

### Rules

- First line must be under 100 characters
- Subject must be lowercase
- No period at the end of the subject line
- Use present tense ("add feature" not "added feature")
- Reference issues in footer when applicable: `Fixes #123`

The commitlint hook will validate your messages automatically.

## Pull Request Guidelines

1. Fork the repository
2. Create a feature branch from `main`
3. Make your changes
4. Ensure tests pass and linting is clean
5. Write clear commit messages following Conventional Commits
6. Submit a pull request

### PR Checklist

- [ ] Tests pass (`make test`)
- [ ] Linting passes with zero issues (`make lint`)
- [ ] Coverage meets 80% threshold (`make coverage`)
- [ ] New features have tests
- [ ] Commit messages follow Conventional Commits format
- [ ] Documentation updated if needed

## Project Structure

```
confvis/
├── cmd/confvis/        # Main entry point
├── internal/
│   ├── cli/            # Command-line interface (cobra)
│   ├── confidence/     # JSON parsing and types
│   ├── gauge/          # SVG gauge generation
│   ├── dashboard/      # HTML dashboard generation
│   ├── history/        # Sparkline history tracking
│   └── sources/        # External metric sources
│       ├── codecov/    # Codecov coverage metrics
│       ├── ghactions/  # GitHub Actions CI metrics
│       ├── snyk/       # Snyk vulnerability metrics
│       ├── sonarqube/  # SonarQube quality metrics
│       └── trivy/      # Trivy security scanning
├── testdata/           # Test fixtures
├── docs/               # Documentation
├── examples/           # Integration examples
└── badges/             # Auto-generated confidence badges
```

## Questions?

Open an issue for questions or discussion.
