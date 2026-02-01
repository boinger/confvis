# External Sources

confvis can fetch metrics directly from external systems using the `confvis fetch` command. This eliminates the need to write custom scripts to transform metrics into the confvis JSON format.

## Available Sources

| Source | Description | Status |
|--------|-------------|--------|
| `sonarqube` | Code quality metrics from SonarQube | Available |
| `codecov` | Coverage metrics from Codecov | Planned |
| `snyk` | Security vulnerability metrics | Planned |

## Usage

```bash
confvis fetch <source> -p <project> -o <output> [source-specific-flags]
```

Common flags for all sources:

| Flag | Short | Description |
|------|-------|-------------|
| `--project` | `-p` | Project identifier (required) |
| `--output` | `-o` | Output file, or `-` for stdout (required) |
| `--title` | | Custom report title |
| `--threshold` | | Pass/fail threshold (default: 75) |
| `--timeout` | | HTTP timeout in seconds (default: 30) |

## SonarQube

Fetches code quality metrics from a SonarQube server.

### Configuration

| Flag | Environment Variable | Description |
|------|---------------------|-------------|
| `--url` | `SONARQUBE_URL` | SonarQube server URL |
| `--token` | `SONARQUBE_TOKEN` | API token for authentication |
| `--branch` | | Branch to query (optional) |

### Metric Mapping

SonarQube metrics are converted to confidence factors:

| SonarQube Metric | Factor Name | Weight | Conversion |
|------------------|-------------|--------|------------|
| `coverage` | Test Coverage | 25% | Direct percentage |
| `reliability_rating` | Reliability | 25% | A=100, B=75, C=50, D=25, E=0 |
| `security_rating` | Security | 25% | A=100, B=75, C=50, D=25, E=0 |
| `sqale_rating` | Maintainability | 25% | A=100, B=75, C=50, D=25, E=0 |

### Example Output

```json
{
  "title": "MyProject",
  "score": 90,
  "threshold": 75,
  "source": "sonarqube",
  "generatedAt": "2026-02-01T15:30:00Z",
  "factors": [
    {"name": "Test Coverage", "score": 85, "weight": 25, "url": "https://sonar.example.com/..."},
    {"name": "Reliability", "score": 100, "weight": 25, "url": "..."},
    {"name": "Security", "score": 100, "weight": 25, "url": "..."},
    {"name": "Maintainability", "score": 75, "weight": 25, "url": "..."}
  ]
}
```

### Examples

```bash
# Basic usage with environment variables
export SONARQUBE_URL=https://sonar.example.com
export SONARQUBE_TOKEN=squ_xxxxxxxxxxxxxxxxxxxx
confvis fetch sonarqube -p myproject -o confidence.json

# Explicit URL and token
confvis fetch sonarqube \
  --url https://sonar.example.com \
  --token squ_xxx \
  -p myproject \
  -o confidence.json

# Fetch specific branch
confvis fetch sonarqube -p myproject --branch main -o confidence.json

# Custom title and stricter threshold
confvis fetch sonarqube -p myproject --title "API Server" --threshold 80 -o confidence.json

# Pipe directly to badge generation
confvis fetch sonarqube -p myproject -o - | confvis gauge -c - -o badge.svg

# Use in CI with verbose output
confvis fetch sonarqube -p myproject -o confidence.json -v
```

### Authentication

SonarQube supports two authentication methods:

1. **User Token** (recommended): Generate at User > My Account > Security > Tokens
2. **Project Token**: Generate at Project Settings > Analysis Tokens

Both use the same `--token` flag or `SONARQUBE_TOKEN` environment variable.

For public SonarQube projects, authentication may not be required.

## CI/CD Integration

### GitHub Actions

```yaml
name: Quality Badge

on: [push]

jobs:
  badge:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install confvis
        run: go install github.com/boinger/confvis/cmd/confvis@latest

      - name: Fetch SonarQube metrics
        env:
          SONARQUBE_URL: ${{ secrets.SONARQUBE_URL }}
          SONARQUBE_TOKEN: ${{ secrets.SONARQUBE_TOKEN }}
        run: |
          confvis fetch sonarqube -p ${{ github.repository }} -o confidence.json

      - name: Generate badge
        run: confvis gauge -c confidence.json -o badge.svg --fail-under 75

      - name: Upload badge
        uses: actions/upload-artifact@v4
        with:
          name: badge
          path: badge.svg
```

### GitLab CI

```yaml
quality-badge:
  image: golang:latest
  script:
    - go install github.com/boinger/confvis/cmd/confvis@latest
    - confvis fetch sonarqube -p $CI_PROJECT_PATH -o confidence.json
    - confvis gauge -c confidence.json -o badge.svg --fail-under 75
  artifacts:
    paths:
      - badge.svg
  variables:
    SONARQUBE_URL: $SONARQUBE_URL
    SONARQUBE_TOKEN: $SONARQUBE_TOKEN
```

## Combining Multiple Sources

Fetch from multiple sources and aggregate:

```bash
# Fetch from different sources
confvis fetch sonarqube -p myproject -o sonar.json

# Aggregate with other reports
confvis aggregate \
  -c sonar.json:60 \
  -c manual-review.json:40 \
  -o ./output
```

Or use process substitution for a single pipeline:

```bash
confvis aggregate \
  -c <(confvis fetch sonarqube -p api -o -):60 \
  -c <(confvis fetch sonarqube -p web -o -):40 \
  -o ./output
```

## Adding New Sources

See [Architecture](architecture.md#adding-new-sources) for instructions on implementing new source modules.
