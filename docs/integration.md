# Integration Guide

This guide covers common integration patterns for confvis.

## GitHub Actions

### Using the Native Action (Recommended)

The simplest way to use confvis in GitHub Actions - no installation required:

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

      - uses: boinger/confvis@v1
        with:
          config: confidence.json
          output: badges/badge.svg

      - name: Commit badge
        uses: stefanzweifel/git-auto-commit-action@v5
        with:
          commit_message: "Update confidence badge"
          file_pattern: "badges/*"
```

See [GitHub Action Documentation](github-action.md) for all options including baseline comparison, check creation, and sparkline badges.

### Using CLI Installation

If you need more control or want to run multiple commands:

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
          go-version: '1.23'

      - name: Install confvis
        run: go install github.com/boinger/confvis/cmd/confvis@latest

      - name: Generate badge
        run: confvis generate -c confidence.json -o ./badges --fail-under 75 -q

      - name: Quality gate (CI-only alternative)
        id: gate
        run: confvis gate -c confidence.json --fail-under 75 -q
        # Gate outputs are available as steps.gate.outputs.gate_result and steps.gate.outputs.gate_score

      - name: Commit badge
        uses: stefanzweifel/git-auto-commit-action@v5
        with:
          commit_message: "Update confidence badge"
          file_pattern: "badges/*"
```

## Pull Request Comments

Post confidence reports directly to PRs using the `github-comment` format:

```yaml
name: PR Confidence Report

on:
  pull_request:
    branches: [main]

jobs:
  confidence:
    runs-on: ubuntu-latest
    permissions:
      pull-requests: write
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Install confvis
        run: go install github.com/boinger/confvis/cmd/confvis@latest

      - name: Generate confidence report
        id: report
        run: |
          # Generate GitHub-flavored markdown for PR comment
          # Uses stored baseline for comparison (shows delta in comment)
          confvis gauge -c confidence.json --compare-baseline -o report.md --format github-comment

      - name: Post PR comment
        uses: peter-evans/create-or-update-comment@v4
        with:
          issue-number: ${{ github.event.pull_request.number }}
          body-path: report.md
```

The `github-comment` format produces output with:
- Emoji status indicators (:white_check_mark: / :x:)
- Collapsible factor breakdown using `<details>`
- Delta arrows when comparing against baseline (:arrow_up: / :arrow_down:)
- Clean formatting optimized for GitHub's markdown renderer

## GitHub Checks Integration

Create native GitHub Check Runs that appear directly in the PR checks tab:

```yaml
name: Confidence Check

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  confidence:
    runs-on: ubuntu-latest
    permissions:
      checks: write  # Required for creating check runs
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Install confvis
        run: go install github.com/boinger/confvis/cmd/confvis@latest

      - name: Create confidence check
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: confvis check github -c confidence.json
```

The check run shows:
- Pass/fail status based on threshold
- Score percentage and threshold
- Factor breakdown table

### Combined with Fetching

Fetch metrics from external sources and create a check in one pipeline:

```yaml
- name: Fetch and create check
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    SONARQUBE_URL: ${{ vars.SONARQUBE_URL }}
    SONARQUBE_TOKEN: ${{ secrets.SONARQUBE_TOKEN }}
  run: |
    confvis fetch sonarqube -p myproject -o confidence.json
    confvis check github -c confidence.json --name "SonarQube Quality"
```

### Custom Check Names

Use different names for different aspects:

```yaml
- name: Create coverage check
  run: confvis check github -c coverage.json --name "Test Coverage"

- name: Create security check
  run: confvis check github -c security.json --name "Security Scan"
```

## Baseline Management for Regression Detection

Baselines let you track score changes across commits. On merge to main, save the current score; on PRs, compare against it.

### Basic Workflow

```yaml
name: Confidence with Baseline

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  confidence:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Install confvis
        run: go install github.com/boinger/confvis/cmd/confvis@latest

      - name: Save baseline (main only)
        if: github.ref == 'refs/heads/main'
        run: confvis baseline save -c confidence.json

      - name: Quality gate (CI-only, no badge)
        id: gate
        run: confvis gate -c confidence.json --compare-baseline --fail-on-regression
        # Outputs: steps.gate.outputs.gate_result (pass|fail), steps.gate.outputs.gate_score

      - name: Generate badge with comparison
        run: |
          confvis gauge -c confidence.json \
            --compare-baseline \
            --fail-on-regression \
            -o badge.svg

      - name: Show baseline info
        run: confvis baseline show || echo "No baseline yet"
```

### How Baselines Work

Baselines are stored in git refs (`refs/confvis/baseline`) by default, which:
- Keeps them out of your working tree
- Doesn't create commits or pollute history
- Works across branches (all branches can read/write the same ref)
- Survives branch switches and rebases

For non-git repos or when you need file-based storage:

```bash
# Save to file
confvis baseline save -c confidence.json --file baseline.json

# Compare against file
confvis gauge -c confidence.json --compare-baseline --baseline-file baseline.json -o badge.svg
```

### Viewing Baseline Info

```bash
# Show current baseline
confvis baseline show
# Output: Baseline: 85% (saved 2024-01-15T10:30:00Z, commit abc1234, branch main)

# Show as JSON
confvis baseline show --format json
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

# CI gate: fail if score drops below threshold (no badge output)
confidence-ci:
	confvis gate -c confidence.json --fail-under 75 -q
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
    confvis gate -c confidence.json --fail-under 70 -q || {
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
