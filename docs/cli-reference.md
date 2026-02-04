# CLI Reference

> **Tip:** For GitHub Actions workflows, consider using the [native GitHub Action](github-action.md) instead of the CLI for simpler configuration.

## Configuration File

confvis supports a YAML configuration file to set default values for flags. This eliminates the need to specify common flags repeatedly.

### Config File Locations

Config files are searched in the following order (first found wins):

1. `.confvis.yaml` in current directory
2. `.confvis.yml` in current directory
3. `~/.config/confvis/config.yaml`

### Precedence

Configuration values are applied in this order (later overrides earlier):

1. **Defaults** - Built-in default values
2. **Config file** - Values from `.confvis.yaml`
3. **Environment variables** - `CONFVIS_<SECTION>_<KEY>` format
4. **Command-line flags** - Explicit flag values

### Sample Config

```yaml
# Gauge command defaults
gauge:
  width: 200
  height: 120
  style: github        # github, minimal, corporate, high-contrast
  dark: false
  fail_under: 75
  badge_type: gauge    # gauge, flat, sparkline
  history_file: .confvis-history.jsonl
  history_ref: refs/confvis/history  # git ref for history storage
  history_auto: true   # auto-detect: git ref if in repo, else file
  history_count: 10
  green_above: 75
  yellow_above: 50
  compare_baseline: true  # auto-compare against stored baseline
  baseline_ref: refs/confvis/baseline  # git ref for baseline storage
  # Per-factor thresholds - fail if individual factors drop below
  factor_thresholds:
    Test Coverage: 80
    Security Scan: 90
    Code Complexity: 70

# Baseline command defaults
baseline:
  ref: refs/confvis/baseline

# Fetch command defaults
fetch:
  timeout: 30
  threshold: 75

# Source-specific defaults
sources:
  codecov:
    service: github    # github, gitlab, bitbucket
  snyk:
    org: my-org-id
  sonarqube:
    url: https://sonar.example.com
```

### Environment Variables

All config values can be overridden with environment variables using the format:
`CONFVIS_<SECTION>_<KEY>`

Examples:
- `CONFVIS_GAUGE_STYLE=minimal`
- `CONFVIS_GAUGE_FAIL_UNDER=80`
- `CONFVIS_FETCH_TIMEOUT=60`

---

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--verbose` | `-v` | Enable verbose output |
| `--quiet` | `-q` | Suppress non-error output (overrides `-v`) |
| `--help` | `-h` | Show help |

## Commands

### `confvis generate`

Generate both an SVG badge and HTML dashboard from a confidence report.

```bash
confvis generate -c <config> -o <output-dir> [flags]
```

#### Required Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to confidence report (JSON/YAML), or `-` for stdin |
| `--input-format` | | Input format: `auto`, `json`, or `yaml` (auto-detects from extension) |
| `--output` | `-o` | Output directory path |

#### Optional Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--dark` | false | Use dark mode colors |
| `--fail-under` | 0 | Exit with code 1 if score is below this value |

#### Output

Creates:
- `<output>/badge.svg` - SVG gauge badge
- `<output>/dashboard/index.html` - HTML dashboard

#### Examples

```bash
# Basic usage
confvis generate -c confidence.json -o ./output

# With dark mode
confvis generate -c confidence.json -o ./output --dark

# Verbose output
confvis generate -c confidence.json -o ./output -v

# Read from stdin
cat confidence.json | confvis generate -c - -o ./output

# Fail if score below threshold (CI/CD)
confvis generate -c confidence.json -o ./output --fail-under 75

# Quiet mode (no output on success)
confvis generate -c confidence.json -o ./output -q
```

---

### `confvis gauge`

Generate an SVG gauge badge from a confidence report.

```bash
confvis gauge -c <config> -o <output-file> [flags]
```

#### Required Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to confidence report (JSON/YAML), or `-` for stdin |
| `--output` | `-o` | Output SVG file path, or `-` for stdout |

#### Optional Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--input-format` | | auto | Input format: `auto`, `json`, or `yaml` (auto-detects from extension) |
| `--format` | `-f` | svg | Output format: `svg`, `json`, `text`, `markdown`, or `github-comment` |
| `--badge-type` | | gauge | Badge type: `gauge`, `flat`, or `sparkline` |
| `--label` | | | Custom label for flat badge (defaults to report title) |
| `--icon` | | | SVG path data for flat badge icon |
| `--history-file` | | | Path to history file for sparkline (JSON lines format) |
| `--history-ref` | | | Git ref for storing history (e.g., `refs/confvis/history`) |
| `--history-auto` | | false | Auto-detect history storage: git ref if in repo, else file |
| `--history-count` | | 10 | Number of historical points to show in sparkline |
| `--width` | | 200 | Gauge width in pixels (gauge badge only) |
| `--height` | | 120 | Gauge height in pixels (gauge badge only) |
| `--style` | | github | Color scheme style (svg only) |
| `--dark` | | false | Use dark mode colors (svg only) |
| `--fail-under` | | 0 | Exit with code 1 if score is below this value |
| `--green-above` | | 75 | Score threshold for green color (overrides JSON) |
| `--yellow-above` | | 50 | Score threshold for yellow color (overrides JSON) |
| `--compare` | | | Path to baseline report JSON for comparison |
| `--compare-baseline` | | false | Auto-fetch baseline from ref/file and compare |
| `--baseline-ref` | | refs/confvis/baseline | Git ref for baseline storage |
| `--baseline-file` | | | File path fallback for non-git repos |
| `--fail-on-regression` | | false | Exit with code 1 if score decreased from baseline |
| `--factor-threshold` | | | Per-factor threshold in `Name:threshold` format (can be repeated) |

#### Output Formats

- **svg** (default): SVG gauge badge image
- **json**: JSON object with score data and metadata:
  ```json
  {
    "title": "string",
    "score": 85,
    "threshold": 75,
    "passed": true,
    "version": "string (if present)",
    "generatedAt": "string (if present)",
    "source": "string (if present)",
    "baseline": 80,
    "delta": 5
  }
  ```
  Note: `baseline` and `delta` fields are only present when using `--compare` or `--compare-baseline`.
- **text**: Plain text score number (useful for scripting). With `--compare`, shows delta: `85 (+5)`
- **markdown**: Markdown-formatted report for PR comments or wiki pages:
  ```markdown
  ## Code Quality: 85% (PASS)

  | Factor | Score | Weight |
  |--------|------:|-------:|
  | Test Coverage | 92% | 30% |
  ```
- **github-comment**: GitHub-flavored markdown optimized for PR comments:
  ```markdown
  ## Confidence Report: My Project

  | Metric | Value |
  |--------|-------|
  | Score | **85%** :white_check_mark: |
  | Threshold | 75% |
  | Status | Passed |
  | Change | +5 :arrow_up: |

  <details>
  <summary>Factor Breakdown</summary>

  | Factor | Score | Weight | Description |
  |--------|-------|--------|-------------|
  | Coverage | 90 | 30 | 90% coverage |
  | Security | 80 | 30 | 0 vulnerabilities |

  </details>

  ---
  <sub>Generated by confvis v1.0.0</sub>
  ```
  Features: emoji status indicators, collapsible factor breakdown, baseline delta with directional arrows.

#### Color Styles

Available `--style` options:

| Style | Description |
|-------|-------------|
| `github` | GitHub-inspired colors (default) |
| `minimal` | Clean, subtle color scheme |
| `corporate` | Professional, muted colors |
| `high-contrast` | Accessibility-focused high contrast |

All styles support `--dark` mode.

#### Examples

```bash
# Basic usage
confvis gauge -c confidence.json -o badge.svg

# Custom dimensions
confvis gauge -c confidence.json -o badge.svg --width 300 --height 180

# Dark mode
confvis gauge -c confidence.json -o badge.svg --dark

# Read from stdin
cat confidence.json | confvis gauge -c - -o badge.svg

# Write to stdout
confvis gauge -c confidence.json -o -

# Pipe through (stdin to stdout)
cat confidence.json | confvis gauge -c - -o - > badge.svg

# Fail if score below threshold (CI/CD)
confvis gauge -c confidence.json -o badge.svg --fail-under 75

# Quiet mode (no output on success)
confvis gauge -c confidence.json -o badge.svg -q

# Output as JSON
confvis gauge -c confidence.json -o - -f json

# Output just the score (for scripting)
SCORE=$(confvis gauge -c confidence.json -o - -f text)

# Custom color thresholds (stricter)
confvis gauge -c confidence.json -o badge.svg --green-above 90 --yellow-above 70

# Different color styles
confvis gauge -c confidence.json -o badge.svg --style minimal
confvis gauge -c confidence.json -o badge.svg --style corporate --dark
confvis gauge -c confidence.json -o badge.svg --style high-contrast

# Shields.io-style flat badge
confvis gauge -c confidence.json -o badge.svg --badge-type flat
confvis gauge -c confidence.json -o badge.svg --badge-type flat --label "Quality"

# Flat badge with custom icon (SVG path data)
confvis gauge -c confidence.json -o badge.svg --badge-type flat --icon "M7 1 A6 6 0 1 1 4 1.8 L7 7 Z"

# Output as markdown (for PR comments)
confvis gauge -c confidence.json -o - -f markdown

# Output as GitHub-flavored markdown with emoji and collapsible sections
confvis gauge -c confidence.json -o - -f github-comment

# GitHub comment with baseline comparison showing delta
confvis gauge -c confidence.json --compare baseline.json -o - -f github-comment

# Compare against baseline
confvis gauge -c confidence.json --compare baseline.json -o - -f json

# Fail if score regressed from baseline (CI/CD)
confvis gauge -c confidence.json --compare baseline.json -o badge.svg --fail-on-regression

# Use YAML input (auto-detected from extension)
confvis gauge -c confidence.yaml -o badge.svg

# YAML from stdin (requires explicit format)
cat confidence.yaml | confvis gauge -c - --input-format yaml -o badge.svg

# Sparkline badge showing score trend (file-based history)
confvis gauge -c confidence.json -o sparkline.svg --badge-type sparkline --history-file .confvis-history.jsonl

# Sparkline with custom history count
confvis gauge -c confidence.json -o sparkline.svg --badge-type sparkline --history-file .confvis-history.jsonl --history-count 20

# Sparkline with git ref storage (no file needed, persists in git)
confvis gauge -c confidence.json -o sparkline.svg --badge-type sparkline --history-ref refs/confvis/history

# Sparkline with auto-detection (uses git ref if in repo, else file)
confvis gauge -c confidence.json -o sparkline.svg --badge-type sparkline --history-auto

# Compare against stored baseline (auto-fetched from git ref)
confvis gauge -c confidence.json --compare-baseline -o badge.svg

# Compare against baseline with regression check
confvis gauge -c confidence.json --compare-baseline --fail-on-regression -o badge.svg

# Compare against baseline stored in file
confvis gauge -c confidence.json --compare-baseline --baseline-file baseline.json -o badge.svg

# Enforce per-factor threshold (fail if Coverage factor < 80)
confvis gauge -c confidence.json -o badge.svg --factor-threshold "Test Coverage:80"

# Multiple per-factor thresholds
confvis gauge -c confidence.json -o badge.svg \
  --factor-threshold "Test Coverage:80" \
  --factor-threshold "Security Scan:90"

# Per-factor thresholds combined with overall threshold
confvis gauge -c confidence.json -o badge.svg --fail-under 75 \
  --factor-threshold "Coverage:80" --factor-threshold "Security:90"
```

---

### `confvis baseline`

Manage confidence baselines for regression detection. Baselines are saved confidence scores
that can be used to detect regressions in CI/CD pipelines.

```bash
confvis baseline <command> [flags]
```

#### Subcommands

##### `confvis baseline save`

Save current confidence report as baseline.

```bash
confvis baseline save -c <config> [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--config` | `-c` | | Path to confidence report (required) |
| `--ref` | | refs/confvis/baseline | Git ref for storage |
| `--file` | | | File path alternative to git ref |
| `--dry-run` | | false | Preview what would be saved without writing |

##### `confvis baseline show`

Display the current baseline.

```bash
confvis baseline show [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--ref` | refs/confvis/baseline | Git ref to read from |
| `--file` | | File path alternative to git ref |
| `--format` | text | Output format: `text` or `json` |

#### Baseline Storage

By default, baselines are stored in git refs (`refs/confvis/baseline`), which keeps them
out of your working tree but accessible across branches. Use `--file` for non-git repos.

#### Examples

```bash
# Save baseline to default git ref
confvis baseline save -c confidence.json

# Save to custom git ref
confvis baseline save -c confidence.json --ref refs/confvis/prod-baseline

# Save to file instead of git ref
confvis baseline save -c confidence.json --file baseline.json

# Show baseline
confvis baseline show

# Show baseline as JSON
confvis baseline show --format json

# Show baseline from file
confvis baseline show --file baseline.json

# Preview what would be saved (dry-run)
confvis baseline save -c confidence.json --dry-run
```

#### CI/CD Usage

```yaml
# Save baseline on main branch merges
- name: Save baseline
  if: github.ref == 'refs/heads/main'
  run: confvis baseline save -c confidence.json

# Compare against baseline on PRs
- name: Check for regressions
  run: confvis gauge -c confidence.json --compare-baseline --fail-on-regression -o badge.svg
```

---

### `confvis fetch`

Fetch metrics from an external source and generate a confidence report.

```bash
confvis fetch <source> -p <project> -o <output> [flags]
```

#### Available Sources

| Source | Description |
|--------|-------------|
| `sonarqube` | Code quality metrics from SonarQube |
| `codecov` | Coverage metrics from Codecov |
| `github-actions` | CI/CD workflow metrics from GitHub Actions |
| `snyk` | Vulnerability metrics from Snyk |
| `trivy` | Local security vulnerability scanning |

#### Required Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--project` | `-p` | Project key/identifier |
| `--output` | `-o` | Output file path, or `-` for stdout |

#### Common Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--url` | `-u` | | Source server URL (or use environment variable) |
| `--token` | `-t` | | API token (or use environment variable) |
| `--branch` | `-b` | | Branch to query (SonarQube only) |
| `--title` | | | Report title (defaults to project name) |
| `--threshold` | | 75 | Pass/fail threshold |
| `--timeout` | | 30 | HTTP timeout in seconds |

#### Source-Specific Flags

| Flag | Source | Default | Description |
|------|--------|---------|-------------|
| `--service` | codecov | github | Git provider (github, gitlab, bitbucket) |
| `--workflow` | github-actions | | Workflow file or ID to filter |
| `--event` | github-actions | | Trigger event to filter (push, pull_request) |
| `--count` | github-actions | 20 | Number of recent runs to analyze |
| `--org` | snyk | | Organization ID (required for Snyk) |
| `--trivy-cmd` | trivy | trivy | Trivy command to execute |

#### Environment Variables

| Variable | Source | Description |
|----------|--------|-------------|
| `SONARQUBE_URL` | sonarqube | Server URL |
| `SONARQUBE_TOKEN` | sonarqube | API token |
| `CODECOV_TOKEN` | codecov | API token |
| `DEPENDABOT_TOKEN` | dependabot | API token (fallback: GITHUB_TOKEN) |
| `GITHUB_TOKEN` | dependabot, github-actions | Personal access token |
| `GITHUB_API_URL` | dependabot, github-actions | API URL (for Enterprise) |
| `SNYK_TOKEN` | snyk | API token |
| `SNYK_ORG_ID` | snyk | Organization ID |
| `SNYK_API_URL` | snyk | API URL |
| `TRIVY_CMD` | trivy | Trivy command to execute |

#### Examples

```bash
# Fetch from SonarQube
export SONARQUBE_URL=https://sonar.example.com
export SONARQUBE_TOKEN=squ_xxx
confvis fetch sonarqube -p myproject -o confidence.json

# Fetch from Codecov (project is owner/repo)
export CODECOV_TOKEN=xxx
confvis fetch codecov -p myorg/myrepo -o confidence.json

# Fetch from Dependabot (GitHub vulnerability alerts)
export GITHUB_TOKEN=xxx
confvis fetch dependabot -p myorg/myrepo -o dependabot.json

# Fetch from GitHub Actions with workflow filter
export GITHUB_TOKEN=xxx
confvis fetch github-actions -p myorg/myrepo --workflow ci.yml --count 20 -o confidence.json

# Fetch from Snyk
export SNYK_TOKEN=xxx
confvis fetch snyk --org my-org-id -p my-project-id -o confidence.json

# Fetch from Trivy (local scan)
confvis fetch trivy -p . -o security.json

# Pipe directly to gauge for badge generation
confvis fetch sonarqube -p myproject -o - | confvis gauge -c - -o badge.svg

# Verbose output shows score details
confvis fetch sonarqube -p myproject -o confidence.json -v
```

See [Sources Documentation](sources.md) for detailed information on each source.

---

### `confvis aggregate`

Aggregate multiple confidence reports into a single dashboard with an overall score.

```bash
confvis aggregate -c <config>[:weight] [-c <config>[:weight] ...] -o <output-dir> [flags]
```

#### Required Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to confidence report (JSON/YAML), or glob pattern. Can be repeated. Optional weight suffix (e.g., `path:80`) |
| `--output` | `-o` | Output directory path |

#### Optional Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--badge-type` | gauge | Badge type: `gauge` or `flat` |
| `--icon` | | SVG path data for flat badge icon |
| `--label` | | Custom label for flat badge (defaults to 'Aggregate') |
| `--dark` | false | Use dark mode colors |
| `--fail-under` | 0 | Exit with code 1 if aggregate score is below this value |

#### Config Format

Each `-c` flag accepts a path with optional weight:

- `path` - Use default weight of 100
- `path:weight` - Use specified weight (e.g., `api.json:60`)
- `glob` - Expand glob pattern (e.g., `services/*/confidence.json`)
- `glob:weight` - All matched files use same weight

#### Output

Creates:
- `<output>/badge.svg` - Aggregate SVG gauge badge
- `<output>/dashboard/index.html` - Multi-report HTML dashboard
- `<output>/<report-title>.svg` - Individual badges for each report

#### Examples

```bash
# Basic aggregation of two reports
confvis aggregate -c api/confidence.json -c web/confidence.json -o ./output

# With custom weights (API counts more)
confvis aggregate -c api/confidence.json:60 -c web/confidence.json:40 -o ./output

# Using glob pattern for monorepo
confvis aggregate -c "services/*/confidence.json" -o ./output

# Multiple glob patterns
confvis aggregate -c "backend/*.json" -c "frontend/*.json:50" -o ./output

# Dark mode
confvis aggregate -c api/confidence.json -c web/confidence.json -o ./output --dark

# CI/CD with threshold
confvis aggregate -c "services/*/confidence.json" -o ./output --fail-under 75

# Verbose output showing weights
confvis aggregate -c api.json:60 -c web.json:40 -o ./output -v

# Flat badge with icon and custom label
confvis aggregate -c api.json -c web.json -o ./output --badge-type flat --icon "M7 1 A6 6 0 1 1 4 1.8 L7 7 Z" --label "My Project"
```

---

### `confvis check`

Create check runs on CI platforms directly from confidence reports.

```bash
confvis check <platform> [flags]
```

#### Available Platforms

| Platform | Description |
|----------|-------------|
| `github` | Create a GitHub Check Run |

#### `confvis check github`

Create a GitHub Check Run for your confidence score.

```bash
confvis check github -c <config> [flags]
```

##### Required Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to confidence report (JSON/YAML), or `-` for stdin |

##### Optional Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--owner` | | Repository owner (auto-detected in GitHub Actions) |
| `--repo` | | Repository name (auto-detected in GitHub Actions) |
| `--sha` | | Commit SHA (auto-detected in GitHub Actions) |
| `--name` | Confidence Score | Check run name |
| `--token` | | GitHub token (or `GITHUB_TOKEN` env var) |
| `--api-url` | | GitHub API URL (auto-detected in GitHub Actions) |
| `--auto-detect` | true | Auto-detect values from GitHub Actions environment |
| `--dry-run` | false | Preview check run without creating it |

##### Environment Variables

| Variable | Description |
|----------|-------------|
| `GITHUB_TOKEN` | GitHub API token (required) |
| `GITHUB_REPOSITORY` | Repository in `owner/repo` format |
| `GITHUB_SHA` | Commit SHA |
| `GITHUB_API_URL` | API URL (for GitHub Enterprise) |

##### Examples

```bash
# Auto-detect from GitHub Actions environment
confvis check github -c confidence.json

# Explicit options
confvis check github -c confidence.json \
  --owner myorg --repo myrepo --sha abc123 \
  --token $GITHUB_TOKEN

# Custom check name
confvis check github -c confidence.json --name "Code Quality"

# Pipe from fetch
confvis fetch sonarqube -p myproject -o - | confvis check github -c -

# With verbose output
confvis check github -c confidence.json -v

# Preview what would be posted (dry-run)
confvis check github -c confidence.json --dry-run
```

##### GitHub Actions Integration

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
          go-version: '1.21'

      - name: Install confvis
        run: go install github.com/boinger/confvis/cmd/confvis@latest

      - name: Create confidence check
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: confvis check github -c confidence.json
```

##### Config File Support

```yaml
# .confvis.yaml
check:
  github:
    name: "Code Quality"
    api_url: https://api.github.example.com  # For GitHub Enterprise
```

---

---

### `confvis comment`

Post confidence reports as comments on pull requests.

```bash
confvis comment <platform> [flags]
```

#### Available Platforms

| Platform | Description |
|----------|-------------|
| `github` | Post comment to GitHub PR |

#### `confvis comment github`

Post confidence report as a comment on a GitHub pull request.

```bash
confvis comment github -c <config> [flags]
```

##### Required Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to confidence report (JSON/YAML), or `-` for stdin |

##### Optional Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--owner` | | Repository owner (auto-detected in GitHub Actions) |
| `--repo` | | Repository name (auto-detected in GitHub Actions) |
| `--pr` | | Pull request number (auto-detected in GitHub Actions) |
| `--token` | | GitHub token (or `GITHUB_TOKEN` env var) |
| `--api-url` | | GitHub API URL (auto-detected in GitHub Actions) |
| `--mode` | update | Comment mode: `create`, `update`, or `replace` |
| `--auto-detect` | true | Auto-detect values from GitHub Actions environment |
| `--dry-run` | false | Preview comment without posting |

##### Comment Modes

| Mode | Behavior |
|------|----------|
| `create` | Always create a new comment |
| `update` | Update existing confvis comment, or create if none (default) |
| `replace` | Delete all previous confvis comments, then create new |

Comments are identified by a hidden HTML marker (`<!-- confvis-comment -->`), allowing
the `update` and `replace` modes to find existing confvis comments.

##### Environment Variables

| Variable | Description |
|----------|-------------|
| `GITHUB_TOKEN` | GitHub API token (required) |
| `GITHUB_REPOSITORY` | Repository in `owner/repo` format |
| `GITHUB_EVENT_PATH` | Path to event JSON (for PR number detection) |
| `GITHUB_API_URL` | API URL (for GitHub Enterprise) |

##### Examples

```bash
# Auto-detect from GitHub Actions environment
confvis comment github -c confidence.json

# Explicit options
confvis comment github -c confidence.json \
  --repo owner/repo --pr 123 \
  --token $GITHUB_TOKEN

# Always create new comment
confvis comment github -c confidence.json --mode create

# Update existing or create new (default)
confvis comment github -c confidence.json --mode update

# Delete all previous confvis comments, then create new
confvis comment github -c confidence.json --mode replace

# Preview without posting
confvis comment github -c confidence.json --dry-run

# Pipe from fetch
confvis fetch sonarqube -p myproject -o - | confvis comment github -c -
```

##### GitHub Actions Integration

```yaml
name: PR Confidence Comment

on:
  pull_request:
    branches: [main]

jobs:
  confidence:
    runs-on: ubuntu-latest
    permissions:
      pull-requests: write  # Required for posting comments
    steps:
      - uses: actions/checkout@v4

      - name: Generate confidence report
        # ... your report generation steps ...

      - name: Post PR comment
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: confvis comment github -c confidence.json
```

##### Config File Support

```yaml
# .confvis.yaml
comment:
  github:
    mode: update
    api_url: https://api.github.example.com  # For GitHub Enterprise
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (invalid config, file not found, score below `--fail-under`, regression with `--fail-on-regression`, or factor below `--factor-threshold`) |

## Examples

### CI/CD Usage

```bash
# Generate badge, fail build if config is invalid
confvis gauge -c confidence.json -o badge.svg || exit 1

# Fail if score drops below 75
confvis gauge -c confidence.json -o badge.svg --fail-under 75

# Fail if specific factors drop below thresholds (quality gates)
confvis gauge -c confidence.json -o badge.svg \
  --factor-threshold "Test Coverage:80" \
  --factor-threshold "Security Scan:90"

# Fail if score regressed from main branch baseline
confvis gauge -c confidence.json --compare main-baseline.json --fail-on-regression -o badge.svg

# Quiet mode for clean CI logs
confvis generate -c confidence.json -o ./output --fail-under 75 -q

# Pipe from another tool
metrics-tool export | confvis gauge -c - -o badge.svg --fail-under 80

# Generate markdown for PR comment
confvis gauge -c confidence.json --compare main-baseline.json -o - -f markdown >> pr-comment.md

# Generate shields.io-compatible badge for README
confvis gauge -c confidence.json -o badge.svg --badge-type flat --label "Quality"
```

### Verbose Debugging

```bash
confvis generate -c confidence.json -o ./output -v
# Output:
# Generating output for "Code Quality" (score: 85, threshold: 75)
# Wrote badge to ./output/badge.svg
# Wrote dashboard to ./output/dashboard/index.html
```
