# External Sources

confvis can fetch metrics directly from external systems using the `confvis fetch` command. This eliminates the need to write custom scripts to transform metrics into the confvis JSON format.

## Available Sources

| Source | Description | Status |
|--------|-------------|--------|
| `sonarqube` | Code quality metrics from SonarQube | Available |
| `codecov` | Coverage metrics from Codecov | Available |
| `codeql` | Code scanning alerts from GitHub CodeQL | Available |
| `coveralls` | Coverage metrics from Coveralls | Available |
| `dependabot` | Vulnerability alerts from GitHub Dependabot | Available |
| `github-actions` | CI/CD workflow metrics from GitHub Actions | Available |
| `gitleaks` | Secret detection with GitLeaks | Available |
| `gosec` | Go security analysis with Gosec | Available |
| `grype` | Security vulnerability scanning with Grype | Available |
| `semgrep` | Static analysis findings from Semgrep | Available |
| `snyk` | Security vulnerability metrics from Snyk | Available |
| `trivy` | Local security vulnerability scanning with Trivy | Available |
| `trufflehog` | Secret detection with TruffleHog | Available |

## Usage

```bash
confvis fetch <source> -p <project> -o <output> [source-specific-flags]
```

Common flags for all sources:

| Flag | Short | Description |
|------|-------|-------------|
| `--project` | `-p` | Project identifier (required) |
| `--output` | `-o` | Output file, or `-` for stdout (required) |
| `--token` | `-t` | API token (or use environment variable) |
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
| `coverage` | Test Coverage | 20% | Direct percentage |
| `reliability_rating` | Reliability | 20% | A=100, B=75, C=50, D=25, E=0 |
| `security_rating` | Security | 20% | A=100, B=75, C=50, D=25, E=0 |
| `sqale_rating` | Maintainability | 20% | A=100, B=75, C=50, D=25, E=0 |
| `vulnerabilities` | Vulnerabilities | 10% | 0=100, 1-5=80, 6-10=60, 11-25=40, 26-50=20, 51+=0 |
| `bugs` | Bugs | 10% | 0=100, 1-5=80, 6-10=60, 11-25=40, 26-50=20, 51+=0 |
| `code_smells` | Code Smells | 5% | 0=100, 1-5=80, 6-10=60, 11-25=40, 26-50=20, 51+=0 |
| `duplicated_lines_density` | Duplication | 5% | 100 - percentage (linear inverse) |

### Example Output

```json
{
  "title": "MyProject",
  "score": 90,
  "threshold": 75,
  "source": "sonarqube",
  "generatedAt": "2026-02-01T15:30:00Z",
  "factors": [
    {"name": "Test Coverage", "score": 85, "weight": 20, "url": "https://sonar.example.com/..."},
    {"name": "Reliability", "score": 100, "weight": 20, "url": "..."},
    {"name": "Security", "score": 100, "weight": 20, "url": "..."},
    {"name": "Maintainability", "score": 75, "weight": 20, "url": "..."},
    {"name": "Vulnerabilities", "score": 100, "weight": 10, "url": "..."},
    {"name": "Bugs", "score": 80, "weight": 10, "url": "..."},
    {"name": "Code Smells", "score": 60, "weight": 5, "url": "..."},
    {"name": "Duplication", "score": 95, "weight": 5, "url": "..."}
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

## Codecov

Fetches code coverage metrics from Codecov.

### Configuration

| Flag | Environment Variable | Description |
|------|---------------------|-------------|
| `--token` | `CODECOV_TOKEN` | API token (required) |
| `--service` | | Git provider: `github`, `gitlab`, `bitbucket` (default: `github`) |

**Note:** The `--project` flag must be in `owner/repo` format (e.g., `myorg/myrepo`).

### Metric Mapping

| Codecov Metric | Factor Name | Weight |
|----------------|-------------|--------|
| `totals.coverage` | Code Coverage | 100% |

### Example Output

```json
{
  "title": "myorg/myrepo",
  "score": 83,
  "threshold": 75,
  "source": "codecov",
  "generatedAt": "2026-02-01T15:30:00Z",
  "factors": [
    {"name": "Code Coverage", "score": 83, "weight": 100, "url": "https://app.codecov.io/gh/myorg/myrepo"}
  ]
}
```

### Examples

```bash
# Basic usage
export CODECOV_TOKEN=xxx
confvis fetch codecov -p myorg/myrepo -o confidence.json

# GitLab project
confvis fetch codecov -p mygroup/myproject --service gitlab -o confidence.json

# Bitbucket project
confvis fetch codecov -p myteam/myrepo --service bitbucket -o confidence.json

# Pipe to badge generation
confvis fetch codecov -p myorg/myrepo -o - | confvis gauge -c - -o coverage-badge.svg
```

### Authentication

Generate an API token at [Codecov Settings](https://app.codecov.io/account/access).

## Dependabot

Fetches security vulnerability alerts from GitHub Dependabot.

### Configuration

| Flag | Environment Variable | Description |
|------|---------------------|-------------|
| `--token` | `DEPENDABOT_TOKEN` or `GITHUB_TOKEN` | GitHub token (required) |
| `--url` | `GITHUB_API_URL` | API URL for GitHub Enterprise |

**Note:** The `--project` flag must be in `owner/repo` format (e.g., `myorg/myrepo`).

### Metric Mapping

Vulnerability counts are converted to scores using severity-based penalties:

| Factor Name | Scoring Formula | Weight |
|-------------|-----------------|--------|
| Critical Vulnerabilities | 100 if 0, else max(0, 100 - count×25) | 40% |
| High Vulnerabilities | 100 if 0, else max(0, 100 - count×15) | 30% |
| Medium Vulnerabilities | 100 if 0, else max(0, 100 - count×5) | 20% |
| Low Vulnerabilities | 100 if 0, else max(0, 100 - count×2) | 10% |

For example:
- 0 critical issues = 100 points
- 2 critical issues = 100 - (2×25) = 50 points
- 4 high issues = 100 - (4×15) = 40 points

### Example Output

```json
{
  "title": "myorg/myrepo",
  "score": 90,
  "threshold": 75,
  "source": "dependabot",
  "generatedAt": "2026-02-01T15:30:00Z",
  "factors": [
    {"name": "Critical Vulnerabilities", "score": 100, "weight": 40, "description": "0 critical"},
    {"name": "High Vulnerabilities", "score": 85, "weight": 30, "description": "1 high"},
    {"name": "Medium Vulnerabilities", "score": 90, "weight": 20, "description": "2 medium"},
    {"name": "Low Vulnerabilities", "score": 96, "weight": 10, "description": "2 low"}
  ]
}
```

### Examples

```bash
# Basic usage with GITHUB_TOKEN
export GITHUB_TOKEN=xxx
confvis fetch dependabot -p myorg/myrepo -o dependabot.json

# Explicit token
confvis fetch dependabot -p myorg/myrepo --token ghp_xxx -o dependabot.json

# GitHub Enterprise
export GITHUB_API_URL=https://api.github.mycompany.com
confvis fetch dependabot -p myorg/myrepo -o dependabot.json

# Pipe to badge generation
confvis fetch dependabot -p myorg/myrepo -o - | confvis gauge -c - -o security-badge.svg
```

### Authentication

Requires a GitHub token with `security_events` scope (for private repos) or `public_repo` scope (for public repos). The token can be:

1. **Personal Access Token (Classic)**: Create at GitHub Settings > Developer Settings > Tokens
2. **Fine-grained Token**: Create with "Dependabot alerts" read permission
3. **GitHub Actions Token**: Use `${{ secrets.GITHUB_TOKEN }}` with `security-events: read` permission

## GitHub Actions

Fetches CI/CD workflow metrics from GitHub Actions.

### Configuration

| Flag | Environment Variable | Description |
|------|---------------------|-------------|
| `--token` | `GITHUB_TOKEN` | Personal access token or Actions token (required) |
| `--url` | `GITHUB_API_URL` | API URL for GitHub Enterprise |
| `--workflow` | | Filter by workflow file name or ID |
| `--event` | | Filter by trigger event (push, pull_request, etc.) |
| `--count` | | Number of recent runs to analyze (default: 20) |

**Note:** The `--project` flag must be in `owner/repo` format (e.g., `myorg/myrepo`).

### Metric Mapping

The success rate is calculated from the most recent completed workflow runs:

| Conclusion | Counted as Success |
|------------|-------------------|
| `success` | Yes |
| `neutral`, `skipped`, `cancelled`, `failure`, `timed_out` | No |

| Factor Name | Calculation | Weight |
|-------------|-------------|--------|
| Workflow Success Rate | (successful runs / total runs) × 100 | 100% |

### Example Output

```json
{
  "title": "myorg/myrepo",
  "score": 85,
  "threshold": 75,
  "source": "github-actions",
  "generatedAt": "2026-02-01T15:30:00Z",
  "factors": [
    {"name": "Workflow Success Rate", "score": 85, "weight": 100, "description": "17/20 successful runs", "url": "https://github.com/myorg/myrepo/actions"}
  ]
}
```

### Examples

```bash
# Basic usage - all workflows
export GITHUB_TOKEN=xxx
confvis fetch github-actions -p myorg/myrepo -o confidence.json

# Filter to specific workflow
confvis fetch github-actions -p myorg/myrepo --workflow ci.yml -o confidence.json

# Filter by event type
confvis fetch github-actions -p myorg/myrepo --event push -o confidence.json

# Analyze more runs for better accuracy
confvis fetch github-actions -p myorg/myrepo --count 50 -o confidence.json

# GitHub Enterprise
export GITHUB_API_URL=https://github.mycompany.com/api/v3
confvis fetch github-actions -p myorg/myrepo -o confidence.json
```

### Authentication

Use a Personal Access Token with `repo` scope, or the automatic `GITHUB_TOKEN` in GitHub Actions workflows.

## Grype

Scans containers and filesystems for security vulnerabilities using Grype.

Similar to Trivy, Grype runs locally and scans your codebase or container images for vulnerabilities.

### Configuration

| Flag | Environment Variable | Description |
|------|---------------------|-------------|
| `--grype-cmd` | `GRYPE_CMD` | Grype command (default: `grype`) |

**Note:** The `--project` flag specifies the target to scan - can be a path (`.`), container image (`alpine:latest`), or SBOM file.

### Prerequisites

Grype must be installed on the system. Install via:

```bash
# macOS
brew install grype

# Linux
curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin

# Docker (no installation required)
confvis fetch grype -p . --grype-cmd "docker run --rm -v $(pwd):/scan anchore/grype /scan" -o security.json
```

### Metric Mapping

Vulnerability counts are converted to scores using severity-based penalties (same as Trivy/Snyk):

| Factor Name | Scoring Formula | Weight |
|-------------|-----------------|--------|
| Critical Vulnerabilities | 100 if 0, else max(0, 100 - count×33) | 40% |
| High Vulnerabilities | 100 if 0, else max(0, 100 - count×20) | 30% |
| Medium Vulnerabilities | 100 if 0, else max(0, 100 - count×10) | 20% |
| Low Vulnerabilities | 100 if 0, else max(0, 100 - count×5) | 10% |

### Example Output

```json
{
  "title": "alpine:3.18",
  "score": 85,
  "threshold": 75,
  "source": "grype",
  "generatedAt": "2026-02-01T15:30:00Z",
  "factors": [
    {"name": "Critical Vulnerabilities", "score": 100, "weight": 40, "description": "0 critical"},
    {"name": "High Vulnerabilities", "score": 80, "weight": 30, "description": "1 high"},
    {"name": "Medium Vulnerabilities", "score": 70, "weight": 20, "description": "3 medium"},
    {"name": "Low Vulnerabilities", "score": 95, "weight": 10, "description": "1 low"}
  ]
}
```

### Examples

```bash
# Scan current directory
confvis fetch grype -p . -o security.json

# Scan a container image
confvis fetch grype -p alpine:3.18 -o security.json

# Scan with Docker (no local grype install needed)
confvis fetch grype -p . --grype-cmd "docker run --rm -v $(pwd):/scan anchore/grype /scan" -o security.json

# Custom title
confvis fetch grype -p . --title "Security Scan" -o security.json

# Pipe to badge generation
confvis fetch grype -p . -o - | confvis gauge -c - -o security-badge.svg
```

### CI/CD Integration

Grype is commonly used in CI/CD pipelines. Here's a GitHub Actions example:

```yaml
- name: Install Grype
  run: |
    curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin

- name: Fetch Grype metrics
  run: confvis fetch grype -p . -o security.json
```

## Semgrep

Scans code for security issues and bugs using Semgrep static analysis.

Semgrep can run locally or read output piped from an existing semgrep scan.

### Configuration

| Flag | Environment Variable | Description |
|------|---------------------|-------------|
| `--semgrep-cmd` | `SEMGREP_CMD` | Semgrep command (default: `semgrep`) |
| `--semgrep-config` | | Semgrep config/rules (default: `auto`) |
| `--from-stdin` | | Read semgrep JSON output from stdin |

**Note:** The `--project` flag specifies the path to scan.

### Prerequisites

Semgrep must be installed or piped output provided. Install via:

```bash
# macOS/Linux
pip install semgrep

# Or use Docker
confvis fetch semgrep -p . --semgrep-cmd "docker run --rm -v $(pwd):/src returntocorp/semgrep" -o semgrep.json
```

### Metric Mapping

Finding counts are converted to scores using severity-based penalties:

| Factor Name | Scoring Formula | Weight |
|-------------|-----------------|--------|
| Error Findings | 100 if 0, else max(0, 100 - count×20) | 40% |
| Warning Findings | 100 if 0, else max(0, 100 - count×10) | 35% |
| Info Findings | 100 if 0, else max(0, 100 - count×2) | 25% |

### Example Output

```json
{
  "title": "myapp",
  "score": 88,
  "threshold": 75,
  "source": "semgrep",
  "generatedAt": "2026-02-01T15:30:00Z",
  "factors": [
    {"name": "Error Findings", "score": 80, "weight": 40, "description": "1 errors"},
    {"name": "Warning Findings", "score": 90, "weight": 35, "description": "1 warnings"},
    {"name": "Info Findings", "score": 98, "weight": 25, "description": "1 info"}
  ]
}
```

### Examples

```bash
# Scan current directory (auto-detect rules)
confvis fetch semgrep -p . -o semgrep.json

# Use specific config/rules
confvis fetch semgrep -p . --semgrep-config "p/security-audit" -o semgrep.json

# Pipe from existing semgrep scan
semgrep --json --config auto . | confvis fetch semgrep --from-stdin -o semgrep.json

# Use Docker (no local installation)
confvis fetch semgrep -p . --semgrep-cmd "docker run --rm -v $(pwd):/src returntocorp/semgrep scan --json /src" -o semgrep.json

# Pipe to badge generation
confvis fetch semgrep -p . -o - | confvis gauge -c - -o semgrep-badge.svg
```

### CI/CD Integration

Semgrep is commonly used in CI/CD pipelines. Here's a GitHub Actions example:

```yaml
- name: Install Semgrep
  run: pip install semgrep

- name: Fetch Semgrep metrics
  run: confvis fetch semgrep -p . -o semgrep.json

# Or pipe from existing scan
- name: Run Semgrep and generate report
  run: |
    semgrep --json --config auto . | confvis fetch semgrep --from-stdin -o semgrep.json
```

## Snyk

Fetches security vulnerability metrics from Snyk.

### Configuration

| Flag | Environment Variable | Description |
|------|---------------------|-------------|
| `--token` | `SNYK_TOKEN` | API token (required) |
| `--org` | `SNYK_ORG_ID` | Organization ID (required) |
| `--url` | `SNYK_API_URL` | API URL (default: api.snyk.io) |

**Note:** The `--project` flag is the Snyk Project ID (UUID format).

### Metric Mapping

Vulnerability counts are converted to scores using severity-based penalties:

| Factor Name | Scoring Formula | Weight |
|-------------|-----------------|--------|
| Critical Vulnerabilities | 100 if 0, else max(0, 100 - count×33) | 40% |
| High Vulnerabilities | 100 if 0, else max(0, 100 - count×20) | 30% |
| Medium Vulnerabilities | 100 if 0, else max(0, 100 - count×10) | 20% |
| Low Vulnerabilities | 100 if 0, else max(0, 100 - count×5) | 10% |

For example:
- 0 critical issues = 100 points
- 2 high issues = 100 - (2×20) = 60 points
- 5 medium issues = 100 - (5×10) = 50 points

### Example Output

```json
{
  "title": "my-project",
  "score": 75,
  "threshold": 75,
  "source": "snyk",
  "generatedAt": "2026-02-01T15:30:00Z",
  "factors": [
    {"name": "Critical Vulnerabilities", "score": 100, "weight": 40, "description": "0 critical"},
    {"name": "High Vulnerabilities", "score": 60, "weight": 30, "description": "2 high"},
    {"name": "Medium Vulnerabilities", "score": 50, "weight": 20, "description": "5 medium"},
    {"name": "Low Vulnerabilities", "score": 55, "weight": 10, "description": "9 low"}
  ]
}
```

### Examples

```bash
# Basic usage
export SNYK_TOKEN=xxx
export SNYK_ORG_ID=my-org-id
confvis fetch snyk -p my-project-uuid -o confidence.json

# Explicit org ID
confvis fetch snyk --org my-org-id -p my-project-uuid -o confidence.json

# Custom API URL (Snyk Enterprise)
export SNYK_API_URL=https://api.eu.snyk.io
confvis fetch snyk --org my-org-id -p my-project-uuid -o confidence.json
```

### Finding Your IDs

1. **Organization ID**: Settings > General > Organization ID
2. **Project ID**: Navigate to project, copy UUID from URL: `app.snyk.io/org/{org}/project/{project-id}`

## CI/CD Integration

### GitHub Actions

> **Note:** The `fetch` command requires CLI installation. For workflows with pre-existing confidence.json files, use the [native GitHub Action](github-action.md) instead.

```yaml
name: Quality Badge

on: [push]

jobs:
  badge:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install confvis
        run: |
          curl -sSL "https://github.com/boinger/confvis/releases/latest/download/confvis_$(uname -s | tr '[:upper:]' '[:lower:]')_amd64.tar.gz" | tar xz -C /tmp
          sudo mv /tmp/confvis /usr/local/bin/

      - name: Fetch and generate
        env:
          SONARQUBE_URL: ${{ secrets.SONARQUBE_URL }}
          SONARQUBE_TOKEN: ${{ secrets.SONARQUBE_TOKEN }}
        run: |
          confvis fetch sonarqube -p ${{ github.repository }} -o confidence.json
          confvis gauge -c confidence.json -o badge.svg --fail-under 75
          # Or for CI gating without badge: confvis gate -c confidence.json --fail-under 75

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
    # Or for CI gating without badge: confvis gate -c confidence.json --fail-under 75
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
confvis fetch codecov -p myorg/myrepo -o coverage.json
confvis fetch snyk --org my-org -p project-id -o security.json
confvis fetch trivy -p . -o trivy.json

# Aggregate with weights
confvis aggregate \
  -c sonar.json:40 \
  -c coverage.json:25 \
  -c security.json:20 \
  -c trivy.json:15 \
  -o ./output
```

Or use process substitution for a single pipeline:

```bash
confvis aggregate \
  -c <(confvis fetch sonarqube -p api -o -):40 \
  -c <(confvis fetch codecov -p myorg/api -o -):25 \
  -c <(confvis fetch snyk --org my-org -p api-project -o -):20 \
  -c <(confvis fetch trivy -p . -o -):15 \
  -o ./output
```

## Trivy

Scans the local filesystem for security vulnerabilities using Trivy.

Unlike other sources that fetch from remote APIs, Trivy runs locally and scans your codebase directly. This is useful for CI/CD pipelines where you want to check for vulnerabilities before deploying.

### Configuration

| Flag | Environment Variable | Description |
|------|---------------------|-------------|
| `--trivy-cmd` | `TRIVY_CMD` | Trivy command (default: `trivy`) |

**Note:** The `--project` flag specifies the path to scan (default: `.`).

### Prerequisites

Trivy must be installed on the system. Install via:

```bash
# macOS
brew install trivy

# Linux (Ubuntu/Debian)
sudo apt-get install trivy

# Docker (no installation required)
confvis fetch trivy -p . --trivy-cmd "docker run --rm -v $(pwd):/scan aquasec/trivy fs /scan" -o security.json
```

### Metric Mapping

Vulnerability counts are converted to scores using severity-based penalties (same as Snyk):

| Factor Name | Scoring Formula | Weight |
|-------------|-----------------|--------|
| Critical Vulnerabilities | 100 if 0, else max(0, 100 - count×33) | 40% |
| High Vulnerabilities | 100 if 0, else max(0, 100 - count×20) | 30% |
| Medium Vulnerabilities | 100 if 0, else max(0, 100 - count×10) | 20% |
| Low Vulnerabilities | 100 if 0, else max(0, 100 - count×5) | 10% |

### Example Output

```json
{
  "title": "confvis",
  "score": 100,
  "threshold": 75,
  "source": "trivy",
  "generatedAt": "2026-02-01T15:30:00Z",
  "factors": [
    {"name": "Critical Vulnerabilities", "score": 100, "weight": 40, "description": "0 critical"},
    {"name": "High Vulnerabilities", "score": 100, "weight": 30, "description": "0 high"},
    {"name": "Medium Vulnerabilities", "score": 100, "weight": 20, "description": "0 medium"},
    {"name": "Low Vulnerabilities", "score": 100, "weight": 10, "description": "0 low"}
  ]
}
```

### Examples

```bash
# Scan current directory
confvis fetch trivy -p . -o security.json

# Scan a specific path
confvis fetch trivy -p ./cmd/myapp -o security.json

# Use Docker to run Trivy (no local installation needed)
confvis fetch trivy -p . --trivy-cmd "docker run --rm -v $(pwd):/scan aquasec/trivy fs /scan" -o security.json

# Custom title
confvis fetch trivy -p . --title "Security Scan" -o security.json

# Pipe to badge generation
confvis fetch trivy -p . -o - | confvis gauge -c - -o security-badge.svg
```

### CI/CD Integration

Trivy is commonly used in CI/CD pipelines. Here's a GitHub Actions example:

```yaml
- name: Install Trivy
  run: |
    curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin

- name: Fetch Trivy metrics
  run: confvis fetch trivy -p . -o security.json
```

## Scoring Methodology

### Vulnerability Penalty Formulas

Each vulnerability source calculates factor scores using:
`score = 100 - (count × penalty)` (minimum 0)

Penalty values differ by source:

| Source | Critical | High | Medium | Low | Rationale |
|--------|----------|------|--------|-----|-----------|
| Dependabot | 25 | 15 | 5 | 2 | Softer penalties—Dependabot alerts include advisories that may not be directly exploitable |
| Grype | 33 | 20 | 10 | 5 | Stricter penalties—direct vulnerability scanner |
| Trivy | 33 | 20 | 10 | 5 | Stricter penalties—direct vulnerability scanner |
| Snyk | 33 | 20 | 10 | 5 | Stricter penalties—direct vulnerability scanner |

### Aggregation Implications

When aggregating scores from multiple vulnerability sources, be aware that:

1. **Scores are not directly comparable** - A score of 75 from Dependabot represents a different security posture than 75 from Trivy
2. **Dependabot is more forgiving** - 3 critical vulnerabilities: Dependabot=25, Trivy=1
3. **Recommendation** - When aggregating, either:
   - Use sources with matching penalty schemes (Grype/Trivy/Snyk)
   - Apply different weights to account for the penalty difference
   - Use a single vulnerability source for consistency

## Adding New Sources

See [Architecture](architecture.md#adding-new-sources) for instructions on implementing new source modules.
