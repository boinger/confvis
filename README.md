# confvis

[![Confidence](./badges/badge.svg)](./badges/dashboard/index.html)
[![CI](https://github.com/boinger/confvis/actions/workflows/ci.yml/badge.svg)](https://github.com/boinger/confvis/actions/workflows/ci.yml)
[![Coverage](https://codecov.io/gh/boinger/confvis/graph/badge.svg)](https://codecov.io/gh/boinger/confvis)

Generate visual confidence badges and dashboards from JSON metrics.

confvis transforms JSON confidence reports into SVG gauge badges and HTML dashboards, making it easy to visualize code quality, test coverage, security scores, or any metric you track.

## Installation

```bash
go install github.com/boinger/confvis/cmd/confvis@latest
```

Or build from source:

```bash
git clone https://github.com/boinger/confvis.git
cd confvis
go build -o confvis ./cmd/confvis
```

## Quick Start

1. Create a confidence report (JSON or YAML):

```json
{
  "title": "Code Quality",
  "score": 85,
  "threshold": 75,
  "factors": [
    {"name": "Test Coverage", "score": 92, "weight": 30},
    {"name": "Code Complexity", "score": 78, "weight": 25},
    {"name": "Documentation", "score": 88, "weight": 20},
    {"name": "Security Scan", "score": 80, "weight": 25}
  ]
}
```

Or in YAML:

```yaml
title: Code Quality
score: 85
threshold: 75
factors:
  - name: Test Coverage
    score: 92
    weight: 30
  - name: Code Complexity
    score: 78
    weight: 25
```

2. Generate visualizations:

```bash
# Generate both badge and dashboard
confvis generate -c confidence.json -o ./output
confvis generate -c confidence.yaml -o ./output  # YAML works too

# Generate just the gauge badge
confvis gauge -c confidence.json -o badge.svg
```

3. Embed in your README:

```markdown
![Confidence](./output/badge.svg)
```

## CI/CD Integration

Use `--fail-under` to enforce minimum scores, or `--fail-on-regression` to detect quality degradation:

```bash
# Fail the build if score drops below 75
confvis gauge -c confidence.json -o badge.svg --fail-under 75

# Fail if score regressed from baseline
confvis gauge -c confidence.json --compare baseline.json --fail-on-regression -o badge.svg

# Quiet mode for clean CI logs
confvis generate -c confidence.json -o ./output --fail-under 75 -q
```

Supports stdin/stdout for pipeline workflows:

```bash
# Pipe from another tool
metrics-tool export | confvis gauge -c - -o badge.svg

# Write directly to stdout
confvis gauge -c confidence.json -o - > badge.svg
```

## External Sources

confvis can fetch metrics directly from external systems:

```bash
# Fetch from SonarQube (code quality)
export SONARQUBE_URL=https://sonar.example.com
export SONARQUBE_TOKEN=squ_xxx
confvis fetch sonarqube -p myproject -o confidence.json

# Fetch from Codecov (coverage)
export CODECOV_TOKEN=xxx
confvis fetch codecov -p myorg/myrepo -o confidence.json

# Fetch from GitHub Actions (CI/CD)
export GITHUB_TOKEN=xxx
confvis fetch github-actions -p myorg/myrepo -o confidence.json

# Fetch from Snyk (security)
export SNYK_TOKEN=xxx
confvis fetch snyk --org my-org-id -p my-project-id -o confidence.json

# Fetch from Trivy (local security scan)
confvis fetch trivy -p . -o security.json

# Pipe directly to badge generation
confvis fetch sonarqube -p myproject -o - | confvis gauge -c - -o badge.svg
```

See [Sources Documentation](docs/sources.md) for details on available sources and their configuration.

## Commands

### `confvis fetch`

Fetch metrics from an external source.

```bash
confvis fetch <source> -p <project> -o <output> [source-specific-flags]
```

Supported sources: `sonarqube`, `codecov`, `github-actions`, `snyk`, `trivy`

### `confvis generate`

Generate both an SVG badge and HTML dashboard.

```bash
confvis generate -c confidence.json -o ./output [--dark]
```

Creates:
- `output/badge.svg` - SVG gauge badge
- `output/dashboard/index.html` - Interactive HTML dashboard

### `confvis gauge`

Generate a gauge badge in various formats.

```bash
confvis gauge -c confidence.json -o badge.svg [--format svg|json|text|markdown] [--badge-type gauge|flat] [--style github|minimal|corporate|high-contrast] [--dark]
```

Output formats:
- `svg` (default): SVG gauge badge image
- `json`: Score metadata as JSON
- `text`: Just the score number (for scripting)
- `markdown`: Markdown table for PR comments

Badge types:
- `gauge` (default): Semi-circular gauge
- `flat`: Shields.io-compatible rectangular badge
- `sparkline`: Trend line showing score history

Color styles: `github` (default), `minimal`, `corporate`, `high-contrast`

### `confvis aggregate`

Aggregate multiple reports into a single dashboard with weighted scores.

```bash
# Aggregate multiple reports
confvis aggregate -c api/confidence.json -c web/confidence.json -o ./output

# With custom weights
confvis aggregate -c api/confidence.json:60 -c web/confidence.json:40 -o ./output

# Using glob patterns (monorepo)
confvis aggregate -c "services/*/confidence.json" -o ./output
```

Creates:
- `output/badge.svg` - Aggregate SVG gauge badge
- `output/dashboard/index.html` - Multi-report dashboard with all components
- `output/<report-title>.svg` - Individual badges for each report

## JSON Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | Yes | Report title |
| `score` | int | No* | Overall score (0-100), auto-calculated if omitted |
| `threshold` | int | Yes | Minimum passing score |
| `description` | string | No | Report description |
| `thresholds` | object | No | Custom color thresholds (`greenAbove`, `yellowAbove`) |
| `factors` | array | No | Breakdown of contributing factors |

*Score is auto-calculated as a weighted average when omitted and factors are present.

Each factor:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Factor name |
| `score` | int | Yes | Factor score (0-100) |
| `weight` | int | Yes | Weight in overall calculation |
| `description` | string | No | Factor description |
| `url` | string | No | Link to detailed report (clickable in dashboard) |

## Documentation

- [Installation Guide](docs/installation.md)
- [CLI Reference](docs/cli-reference.md)
- [JSON Schema](docs/json-schema.md)
- [Integration Guide](docs/integration.md)
- [External Sources](docs/sources.md)
- [Architecture](docs/architecture.md)

## Examples

See the [examples/](examples/) directory for:
- GitHub Actions workflow
- Makefile integration
- Multi-source score aggregation

## License

MIT - see [LICENSE](LICENSE)
