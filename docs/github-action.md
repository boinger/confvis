# GitHub Action

[![GitHub Marketplace](https://img.shields.io/badge/Marketplace-confvis-blue?logo=github)](https://github.com/marketplace/actions/confvis)

confvis provides a native GitHub Action for zero-install usage in your workflows.

## Quick Start

```yaml
- uses: boinger/confvis@v1
  with:
    config: confidence.json
    output: badge.svg
```

## Inputs

| Input | Description | Default | Required |
|-------|-------------|---------|----------|
| `config` | Path to confidence report (JSON/YAML) | - | Yes |
| `output` | Output path for badge/report | - | No (required except for `gate`) |
| `command` | Command: `gauge`, `gate`, `generate`, or `aggregate` | `gauge` | No |
| `format` | Output format: `svg`, `json`, `text`, `markdown`, `github-comment` | `svg` | No |
| `badge-type` | Badge type: `gauge`, `flat`, or `sparkline` | `gauge` | No |
| `style` | Color style: `github`, `minimal`, `corporate`, `high-contrast` | `github` | No |
| `dark` | Use dark mode colors | `false` | No |
| `fail-under` | Fail if score is below this threshold | `0` | No |
| `compare-baseline` | Compare against stored baseline | `false` | No |
| `fail-on-regression` | Fail if score decreased from baseline | `false` | No |
| `save-baseline` | Save current score as baseline | `false` | No |
| `create-check` | Create GitHub Check Run with results | `false` | No |
| `check-name` | Name for the GitHub Check Run | `Confidence Score` | No |
| `post-comment` | Post confidence report as PR comment | `false` | No |
| `comment-mode` | Comment mode: `create`, `update`, or `replace` | `update` | No |
| `history-auto` | Auto-manage sparkline history | `false` | No |
| `factor-threshold` | Per-factor thresholds, comma-separated `Name:threshold` (gate) | `''` | No |
| `compare` | Explicit baseline file path for comparison (gate) | `''` | No |
| `baseline-ref` | Git ref for baseline (gate) | `''` | No |
| `baseline-file` | File path for baseline (gate) | `''` | No |
| `input-format` | Input format: `auto`, `json`, or `yaml` (gate) | `auto` | No |
| `quiet` | Suppress non-error output (gate) | `false` | No |
| `version` | confvis version to use | `latest` | No |

## Outputs

| Output | Description |
|--------|-------------|
| `score` | The confidence score (0-100) |
| `passed` | Whether the score met the threshold (`true`/`false`) |
| `delta` | Score change from baseline (if `compare-baseline` enabled) |
| `gate_result` | Gate result: `pass` or `fail` (gate command only) |
| `gate_score` | Gate score (gate command only) |

## Examples

### Basic Badge Generation

```yaml
name: Generate Badge

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

### With Threshold Enforcement (gauge)

```yaml
- uses: boinger/confvis@v1
  with:
    config: confidence.json
    output: badge.svg
    fail-under: 75
```

### Quality Gate

Use the `gate` command for CI pass/fail gating without badge output:

```yaml
- uses: boinger/confvis@v1
  id: gate
  with:
    config: confidence.json
    command: gate
    fail-under: 75

- name: Use gate results
  run: |
    echo "Result: ${{ steps.gate.outputs.gate_result }}"
    echo "Score: ${{ steps.gate.outputs.gate_score }}"
```

### Gate with Regression Detection

```yaml
- uses: boinger/confvis@v1
  with:
    config: confidence.json
    command: gate
    compare-baseline: true
    fail-on-regression: true
```

### Gate with Per-Factor Thresholds

```yaml
- uses: boinger/confvis@v1
  with:
    config: confidence.json
    command: gate
    fail-under: 70
    factor-threshold: "Coverage:80,Security:90"
```

### Baseline Comparison and Regression Detection

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
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Required for baseline storage in git refs

      - uses: boinger/confvis@v1
        id: confidence
        with:
          config: confidence.json
          output: badge.svg
          compare-baseline: true
          fail-on-regression: true
          # Save baseline only on main branch
          save-baseline: ${{ github.ref == 'refs/heads/main' }}

      - name: Report score
        run: |
          echo "Score: ${{ steps.confidence.outputs.score }}%"
          echo "Passed: ${{ steps.confidence.outputs.passed }}"
          echo "Delta: ${{ steps.confidence.outputs.delta }}"
```

### Create GitHub Check Run

```yaml
- uses: boinger/confvis@v1
  with:
    config: confidence.json
    output: badge.svg
    create-check: true
    check-name: "Code Quality"
  permissions:
    checks: write
```

### Sparkline Badge with History

```yaml
- uses: boinger/confvis@v1
  with:
    config: confidence.json
    output: sparkline.svg
    badge-type: sparkline
    history-auto: true
```

### Generate Dashboard

```yaml
- uses: boinger/confvis@v1
  with:
    config: confidence.json
    output: ./output
    command: generate
```

### Different Badge Styles

```yaml
# Flat badge (Shields.io style)
- uses: boinger/confvis@v1
  with:
    config: confidence.json
    output: badge.svg
    badge-type: flat

# Dark mode
- uses: boinger/confvis@v1
  with:
    config: confidence.json
    output: badge.svg
    dark: true
    style: minimal

# High contrast for accessibility
- uses: boinger/confvis@v1
  with:
    config: confidence.json
    output: badge.svg
    style: high-contrast
```

### PR Comment with Score

Post a confidence report directly to the PR using the native `post-comment` feature:

```yaml
name: PR Confidence Report

on:
  pull_request:
    branches: [main]

jobs:
  report:
    runs-on: ubuntu-latest
    permissions:
      pull-requests: write
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: boinger/confvis@v1
        with:
          config: confidence.json
          output: badge.svg
          compare-baseline: true
          post-comment: true
          comment-mode: update  # update existing or create new
```

Comment modes:
- `create` - Always create a new comment
- `update` - Update existing confvis comment, or create if none (default)
- `replace` - Delete all previous confvis comments, then create new

### Fetch and Report

Combine with external sources:

```yaml
- name: Fetch from SonarQube
  run: |
    confvis fetch sonarqube -p myproject -o confidence.json
  env:
    SONARQUBE_URL: ${{ vars.SONARQUBE_URL }}
    SONARQUBE_TOKEN: ${{ secrets.SONARQUBE_TOKEN }}

- uses: boinger/confvis@v1
  with:
    config: confidence.json
    output: badge.svg
    create-check: true
```

### Complete CI/CD Pipeline

This example uses `gate` for fast CI pass/fail checking on PRs, and `gauge` for badge generation on main — complementary steps:

```yaml
name: Quality Pipeline

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Run tests with coverage
        run: |
          go test -coverprofile=coverage.out ./...
          # Generate confidence.json from coverage

  confidence:
    needs: test
    runs-on: ubuntu-latest
    permissions:
      checks: write
      pull-requests: write
      contents: write
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      # Quality gate: fast pass/fail check (PRs and main)
      - uses: boinger/confvis@v1
        id: gate
        with:
          config: confidence.json
          command: gate
          fail-under: 75
          compare-baseline: true
          fail-on-regression: ${{ github.event_name == 'pull_request' }}

      # Badge generation (main only, after gate passes)
      - uses: boinger/confvis@v1
        if: github.ref == 'refs/heads/main'
        id: badge
        with:
          config: confidence.json
          output: badges/badge.svg
          compare-baseline: true
          save-baseline: true
          create-check: true

      # PR feedback
      - uses: boinger/confvis@v1
        if: github.event_name == 'pull_request'
        with:
          config: confidence.json
          output: badge.svg
          post-comment: true

      - name: Commit badge (main only)
        if: github.ref == 'refs/heads/main'
        uses: stefanzweifel/git-auto-commit-action@v5
        with:
          commit_message: "chore: update confidence badge [skip ci]"
          file_pattern: "badges/*"
```

## Versioning

The action follows semantic versioning:

- `boinger/confvis@v1` - Latest v1.x.x (recommended)
- `boinger/confvis@v1.2.3` - Specific version
- `boinger/confvis@main` - Latest development (not recommended for production)

You can also specify the confvis binary version separately:

```yaml
- uses: boinger/confvis@v1
  with:
    config: confidence.json
    output: badge.svg
    version: v1.0.0  # Use specific confvis version
```

## Permissions

The action may require these permissions depending on features used:

| Feature | Permission | Reason |
|---------|------------|--------|
| `create-check: true` | `checks: write` | Create GitHub Check Runs |
| `save-baseline: true` | `contents: write` | Update git refs |
| `post-comment: true` | `pull-requests: write` | Post PR comments |

## Troubleshooting

### "Permission denied" when saving baseline

Ensure `contents: write` permission and that you've checked out with `fetch-depth: 0`:

```yaml
permissions:
  contents: write

steps:
  - uses: actions/checkout@v4
    with:
      fetch-depth: 0
```

### Check run not appearing

Ensure `checks: write` permission:

```yaml
permissions:
  checks: write
```

### Binary download fails

Check if you're using a valid version. The action downloads pre-built binaries from GitHub releases. Use `version: latest` (default) or a specific release tag.
