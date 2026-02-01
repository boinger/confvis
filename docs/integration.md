# Integration Guide

This guide covers common integration patterns for confvis.

## GitHub Actions

Automatically update confidence badges on push:

```yaml
name: Update Confidence Badge

on:
  push:
    branches: [main]

jobs:
  badge:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Install confvis
        run: go install github.com/boinger/confvis/cmd/confvis@latest

      - name: Generate badge
        run: confvis generate -c confidence.json -o ./badges --fail-under 75 -q

      - name: Commit badge
        uses: stefanzweifel/git-auto-commit-action@v5
        with:
          commit_message: "Update confidence badge"
          file_pattern: "badges/*"
```

## Makefile Integration

Add targets to your Makefile:

```makefile
.PHONY: confidence confidence-check confidence-ci

# Generate badge and dashboard
confidence:
	confvis generate -c confidence.json -o ./badges

# Validate confidence.json (useful in CI)
confidence-check:
	confvis gauge -c confidence.json -o /dev/null
	@echo "Confidence check passed"

# CI gate: fail if score drops below threshold
confidence-ci:
	confvis generate -c confidence.json -o ./badges --fail-under 75 -q
```

## Aggregating Multiple Sources

Create a script that combines metrics from multiple tools into a single report.
With auto-calculate, you only need to specify factors - the overall score is computed automatically:

```bash
#!/bin/bash
# aggregate.sh - Combine metrics into confidence.json

# Get test coverage
COVERAGE=$(go test -cover ./... 2>&1 | grep -oP 'coverage: \K[0-9.]+' | head -1)
COVERAGE=${COVERAGE%.*}  # Remove decimal

# Get linting score (example: count warnings)
LINT_ISSUES=$(golangci-lint run ./... 2>&1 | grep -c "^" || echo 0)
LINT_SCORE=$((100 - LINT_ISSUES * 5))
[ $LINT_SCORE -lt 0 ] && LINT_SCORE=0

# Generate JSON - score is auto-calculated from weighted factors
cat > confidence.json << EOF
{
  "title": "Code Quality",
  "threshold": 75,
  "factors": [
    {"name": "Test Coverage", "score": $COVERAGE, "weight": 50},
    {"name": "Lint Score", "score": $LINT_SCORE, "weight": 50}
  ]
}
EOF

# Show the calculated score
confvis gauge -c confidence.json -o - -f text
```

## Embedding in README

Reference the generated badge in your README:

```markdown
# My Project

![Confidence](./badges/badge.svg)

...
```

For GitHub, the badge will render inline. For other platforms, you may need to use a raw URL or host the badge elsewhere.

## Pre-commit Hook

Validate the confidence report before committing:

```bash
#!/bin/sh
# .git/hooks/pre-commit

if [ -f confidence.json ]; then
    # Validate JSON and optionally enforce minimum score
    confvis gauge -c confidence.json -o /dev/null --fail-under 70 -q || {
        echo "Error: confidence.json is invalid or score too low"
        exit 1
    }
fi
```

## Integration with Coverage Tools

### Go Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# Extract percentage
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print int($3)}')

# Update confidence.json with jq
jq --argjson cov "$COVERAGE" '.factors[0].score = $cov' confidence.json > tmp.json
mv tmp.json confidence.json
```

### Jest (JavaScript)

```bash
# Run Jest with coverage
npm test -- --coverage --coverageReporters=json-summary

# Extract from coverage-summary.json
COVERAGE=$(jq '.total.lines.pct | floor' coverage/coverage-summary.json)
```

## Styling Options

Generate badges in different color schemes to match your project:

```bash
# Clean, minimal style
confvis gauge -c confidence.json -o badge.svg --style minimal

# Professional look
confvis gauge -c confidence.json -o badge.svg --style corporate

# Accessibility-focused
confvis gauge -c confidence.json -o badge.svg --style high-contrast

# Dark mode variants
confvis gauge -c confidence.json -o badge.svg --style minimal --dark
```

## Dashboard Hosting

The generated dashboard (`dashboard/index.html`) is a self-contained HTML file. Options for hosting:

1. **GitHub Pages**: Push to a `gh-pages` branch or `docs/` folder
2. **Static hosting**: Any static file host (Netlify, Vercel, S3)
3. **Local**: Open directly in a browser for development

The dashboard requires no JavaScript or external dependencies.
