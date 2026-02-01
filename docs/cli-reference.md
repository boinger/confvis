# CLI Reference

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
| `--format` | `-f` | svg | Output format: `svg`, `json`, `text`, or `markdown` |
| `--badge-type` | | gauge | Badge type: `gauge`, `flat`, or `sparkline` |
| `--label` | | | Custom label for flat badge (defaults to report title) |
| `--history-file` | | | Path to history file for sparkline (JSON lines format) |
| `--history-count` | | 10 | Number of historical points to show in sparkline |
| `--width` | | 200 | Gauge width in pixels (gauge badge only) |
| `--height` | | 120 | Gauge height in pixels (gauge badge only) |
| `--style` | | github | Color scheme style (svg only) |
| `--dark` | | false | Use dark mode colors (svg only) |
| `--fail-under` | | 0 | Exit with code 1 if score is below this value |
| `--green-above` | | 75 | Score threshold for green color (overrides JSON) |
| `--yellow-above` | | 50 | Score threshold for yellow color (overrides JSON) |
| `--compare` | | | Path to baseline report JSON for comparison |
| `--fail-on-regression` | | false | Exit with code 1 if score decreased from baseline |

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
  Note: `baseline` and `delta` fields are only present when using `--compare`.
- **text**: Plain text score number (useful for scripting). With `--compare`, shows delta: `85 (+5)`
- **markdown**: Markdown-formatted report for PR comments or wiki pages:
  ```markdown
  ## Code Quality: 85% (PASS)

  | Factor | Score | Weight |
  |--------|------:|-------:|
  | Test Coverage | 92% | 30% |
  ```

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

# Output as markdown (for PR comments)
confvis gauge -c confidence.json -o - -f markdown

# Compare against baseline
confvis gauge -c confidence.json --compare baseline.json -o - -f json

# Fail if score regressed from baseline (CI/CD)
confvis gauge -c confidence.json --compare baseline.json -o badge.svg --fail-on-regression

# Use YAML input (auto-detected from extension)
confvis gauge -c confidence.yaml -o badge.svg

# YAML from stdin (requires explicit format)
cat confidence.yaml | confvis gauge -c - --input-format yaml -o badge.svg

# Sparkline badge showing score trend
confvis gauge -c confidence.json -o sparkline.svg --badge-type sparkline --history-file .confvis-history.jsonl

# Sparkline with custom history count
confvis gauge -c confidence.json -o sparkline.svg --badge-type sparkline --history-file .confvis-history.jsonl --history-count 20
```

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
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (invalid config, file not found, score below `--fail-under`, or regression with `--fail-on-regression`) |

## Examples

### CI/CD Usage

```bash
# Generate badge, fail build if config is invalid
confvis gauge -c confidence.json -o badge.svg || exit 1

# Fail if score drops below 75
confvis gauge -c confidence.json -o badge.svg --fail-under 75

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
