# Maximum Dogfooding Infrastructure for confvis

> **First implementation step:** Copy this plan to `docs/IMPLEMENTATION_PLAN.md` for persistence.

Transform confvis into a shining example of code trustworthiness by having it measure itself.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        LOCAL DEVELOPMENT                         │
├─────────────────────────────────────────────────────────────────┤
│  docker-compose up                                               │
│  ├── SonarQube (localhost:9000) ─────► Fast quality feedback    │
│  └── Pre-commit hooks ───────────────► Lint + test + coverage   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                          CI (GitHub Actions)                     │
├─────────────────────────────────────────────────────────────────┤
│  On Push/PR:                                                     │
│  ├── Test + Coverage ────────────────► Codecov (cloud, free)    │
│  ├── Lint (golangci-lint) ───────────► Zero tolerance           │
│  ├── Quality Analysis ───────────────► SonarCloud (cloud, free) │
│  └── Security Scan ──────────────────► Trivy (self-hosted)      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                         DOGFOODING                               │
├─────────────────────────────────────────────────────────────────┤
│  confvis fetch codecov        ─► coverage.json                  │
│  confvis fetch sonarqube      ─► quality.json   (from SonarCloud)│
│  confvis fetch trivy          ─► security.json  (NEW SOURCE)     │
│  confvis fetch github-actions ─► ci.json                        │
│  confvis aggregate            ─► badges/badge.svg               │
└─────────────────────────────────────────────────────────────────┘
```

---

## Current State

| Aspect | Status |
|--------|--------|
| Tests | 191 passing |
| Coverage | ~44.5% (target: 80%) |
| CI/CD | None |
| Dogfooding | None |
| Linter config | None |
| Pre-commit hooks | None |
| Release automation | None |
| SECURITY.md | Missing |
| Trivy source | **Does not exist** (needs implementation)

---

## Phase 1: Foundation

### 1.1 Create `.golangci.yml`

Strict linting configuration with zero tolerance (per CLAUDE.md requirement).

```yaml
run:
  timeout: 5m
  tests: true

linters:
  enable-all: true
  disable:
    - exhaustruct
    - varnamelen
    - wrapcheck
    - depguard

linters-settings:
  gocyclo:
    min-complexity: 15
  funlen:
    lines: 100
    statements: 50
  lll:
    line-length: 120

issues:
  max-issues-per-linter: 0
  max-same-issues: 0
```

### 1.2 Create Root `Makefile`

```makefile
.PHONY: build test lint coverage pre-commit install-tools

build:
	go build -o confvis ./cmd/confvis

test:
	go test -coverprofile=coverage.out -covermode=atomic ./...

coverage: test
	@go tool cover -func=coverage.out | grep total
	@coverage=$$(go tool cover -func=coverage.out | grep total | awk '{gsub(/%/,""); print int($$3)}'); \
	if [ $$coverage -lt 80 ]; then echo "Coverage $$coverage% < 80%"; exit 1; fi

lint:
	golangci-lint run --config .golangci.yml ./...

pre-commit: lint test coverage

install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	npm install -g @commitlint/cli @commitlint/config-conventional
```

### 1.3 Create `.github/workflows/ci.yml`

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go: ['1.21', '1.22', '1.23']
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Needed for SonarCloud
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go }}
      - run: go test -coverprofile=coverage.out -covermode=atomic -json ./... > test-report.json
      - run: |
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{gsub(/%/,""); print int($3)}')
          echo "Coverage: $coverage%"
          if [ $coverage -lt 80 ]; then exit 1; fi
      - uses: codecov/codecov-action@v4
        if: matrix.go == '1.23'
        with:
          token: ${{ secrets.CODECOV_TOKEN }}
          files: coverage.out

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - uses: golangci/golangci-lint-action@v6
        with:
          version: latest
          args: --config .golangci.yml

  sonarcloud:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: go test -coverprofile=coverage.out -covermode=atomic ./...
      - uses: SonarSource/sonarcloud-github-action@master
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          SONAR_TOKEN: ${{ secrets.SONAR_TOKEN }}
```

### 1.4 Create `sonar-project.properties`

```properties
sonar.projectKey=boinger_confvis
sonar.organization=boinger
sonar.projectName=confvis

sonar.sources=.
sonar.exclusions=**/*_test.go,**/testdata/**
sonar.tests=.
sonar.test.inclusions=**/*_test.go

sonar.go.coverage.reportPaths=coverage.out
```

### 1.5 Create `.github/dependabot.yml`

```yaml
version: 2
updates:
  - package-ecosystem: gomod
    directory: "/"
    schedule:
      interval: weekly
    commit-message:
      prefix: "build(deps)"

  - package-ecosystem: github-actions
    directory: "/"
    schedule:
      interval: weekly
    commit-message:
      prefix: "ci(deps)"
```

---

## Phase 2: Quality Gates

### 2.1 Create `.pre-commit-config.yaml`

```yaml
repos:
  - repo: local
    hooks:
      - id: go-fmt
        name: go fmt
        entry: gofmt -w
        language: system
        types: [go]

      - id: golangci-lint
        name: golangci-lint
        entry: golangci-lint run --fix
        language: system
        types: [go]
        pass_filenames: false

      - id: go-test
        name: go test
        entry: bash -c 'go test ./...'
        language: system
        pass_filenames: false

  - repo: https://github.com/alessandrojcm/commitlint-pre-commit-hook
    rev: v9.13.0
    hooks:
      - id: commitlint
        stages: [commit-msg]
        additional_dependencies: ['@commitlint/config-conventional']
```

### 2.2 Create `commitlint.config.js`

```javascript
module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [2, 'always', [
      'feat', 'fix', 'docs', 'style', 'refactor',
      'perf', 'test', 'build', 'ci', 'chore', 'revert'
    ]],
    'subject-case': [2, 'always', 'lower-case'],
    'header-max-length': [2, 'always', 100]
  }
};
```

### 2.3 Update `CONTRIBUTING.md`

Add sections for:
- Conventional commit format with examples
- Pre-commit hook installation
- Coverage requirements (80%)
- Updated PR checklist

---

## Phase 3: Coverage Push (44% → 80%)

### Priority Order

| Package | Current | Target | Strategy |
|---------|---------|--------|----------|
| `cmd/confvis` | 0% | 50% | Add smoke test |
| `internal/cli` | 14% | 60% | Add unit tests for helpers |
| `internal/dashboard` | 30% | 70% | Test GenerateMulti, edge cases |
| `internal/gauge` | 37% | 75% | Test flat.go, sparkline.go |
| `internal/sources/*` | 60-75% | 80% | Error paths, mock servers |

### Key Test Files to Create/Extend

1. `cmd/confvis/main_test.go` - Entry point smoke test
2. `internal/cli/aggregate_test.go` - Unit tests for parseConfigsWithWeights, sanitizeFilename
3. `internal/cli/fetch_test.go` - More unit tests (currently integration-heavy)
4. `internal/gauge/flat_test.go` - Test GenerateFlat with all options
5. `internal/gauge/sparkline_test.go` - Test GenerateSparkline variations
6. `internal/dashboard/dashboard_test.go` - Test GenerateMulti

---

## Phase 4: Security + Trivy Source

### 4.1 Implement Trivy Source for confvis

**NEW**: Add a Trivy source module so confvis can dogfood its own security metrics.

```
internal/sources/trivy/
  trivy.go      # Source implementation
  client.go     # Trivy CLI wrapper (runs trivy fs .)
  types.go      # VulnerabilityReport, Vulnerability structs
  trivy_test.go # Unit tests
```

**Trivy CLI Integration:**
```bash
trivy fs --format json --scanners vuln . > trivy-report.json
```

**Metric Mapping (same as Snyk):**
| Factor Name | Scoring Formula | Weight |
|-------------|-----------------|--------|
| Critical Vulnerabilities | 100 if 0, else max(0, 100 - count×33) | 40 |
| High Vulnerabilities | 100 if 0, else max(0, 100 - count×20) | 30 |
| Medium Vulnerabilities | 100 if 0, else max(0, 100 - count×10) | 20 |
| Low Vulnerabilities | 100 if 0, else max(0, 100 - count×5) | 10 |

**CLI Usage:**
```bash
# Trivy must be installed or available via Docker
confvis fetch trivy -p . -o security.json
confvis fetch trivy -p . --trivy-cmd "docker run aquasec/trivy" -o security.json
```

**Environment Variables:**
| Variable | Description |
|----------|-------------|
| `TRIVY_CMD` | Path to trivy binary or docker command (default: `trivy`) |

### 4.2 Create `.github/workflows/security.yml`

```yaml
name: Security

on:
  push:
    branches: [main]
  pull_request:
  schedule:
    - cron: '0 6 * * 1'  # Weekly

jobs:
  codeql:
    runs-on: ubuntu-latest
    permissions:
      security-events: write
    steps:
      - uses: actions/checkout@v4
      - uses: github/codeql-action/init@v3
        with:
          languages: go
      - uses: github/codeql-action/autobuild@v3
      - uses: github/codeql-action/analyze@v3

  trivy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: 'fs'
          scan-ref: '.'
          format: 'sarif'
          output: 'trivy-results.sarif'
          severity: 'CRITICAL,HIGH'
      - name: Upload Trivy scan results
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: 'trivy-results.sarif'
```

### 4.3 Create `SECURITY.md`

- Supported versions table
- Vulnerability reporting process (email, not public issues)
- Response timeline (48h initial, severity-based fix timeline)
- Security measures in place (CodeQL, Trivy, Dependabot)

---

## Phase 5: Dogfooding

### 5.1 Create `.github/workflows/dogfood.yml`

```yaml
name: Confidence Badge

on:
  push:
    branches: [main]
  schedule:
    - cron: '0 8 * * *'  # Daily
  workflow_dispatch:

jobs:
  dogfood:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Build confvis
        run: go build -o confvis ./cmd/confvis

      - name: Install Trivy
        run: |
          curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin

      - name: Fetch Codecov metrics
        run: ./confvis fetch codecov -p boinger/confvis -o coverage.json
        env:
          CODECOV_TOKEN: ${{ secrets.CODECOV_TOKEN }}

      - name: Fetch SonarCloud metrics
        run: ./confvis fetch sonarqube -p boinger_confvis -o quality.json
        env:
          SONARQUBE_URL: https://sonarcloud.io
          SONARQUBE_TOKEN: ${{ secrets.SONAR_TOKEN }}

      - name: Fetch CI metrics
        run: ./confvis fetch github-actions -p boinger/confvis --workflow ci.yml --count 30 -o ci.json
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: Fetch Trivy security metrics
        run: ./confvis fetch trivy -p . -o security.json

      - name: Aggregate confidence
        run: |
          ./confvis aggregate \
            -c coverage.json:30 \
            -c quality.json:30 \
            -c ci.json:20 \
            -c security.json:20 \
            -o ./badges \
            --fail-under 75

      - name: Commit badge
        if: github.ref == 'refs/heads/main'
        uses: stefanzweifel/git-auto-commit-action@v5
        with:
          commit_message: "chore: update confidence badge [skip ci]"
          file_pattern: "badges/*"
```

### 5.2 Update `README.md`

Add badges at top:
```markdown
[![Confidence](./badges/badge.svg)](./badges/dashboard/index.html)
[![Coverage](https://codecov.io/gh/boinger/confvis/graph/badge.svg)](https://codecov.io/gh/boinger/confvis)
[![CI](https://github.com/boinger/confvis/actions/workflows/ci.yml/badge.svg)](https://github.com/boinger/confvis/actions)
```

---

## Phase 6: Release Automation

### 6.1 Create `.goreleaser.yaml`

```yaml
version: 2

builds:
  - main: ./cmd/confvis
    binary: confvis
    env:
      - CGO_ENABLED=0
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: checksums.txt

changelog:
  use: github-native
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore\(deps\):'
```

### 6.2 Create `.github/workflows/release.yml`

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

## Phase 7: Polish

1. Create `.github/PULL_REQUEST_TEMPLATE.md`
2. Create `.github/ISSUE_TEMPLATE/bug_report.md`
3. Create `.github/ISSUE_TEMPLATE/feature_request.md`
4. Create `.editorconfig` for consistent formatting
5. Final documentation pass

---

## Implementation Sequence

```
Week 1: Phase 1 (Foundation)
         - .golangci.yml, Makefile, ci.yml, dependabot.yml
   ↓
Week 2: Phase 2 (Quality Gates)
         - pre-commit, commitlint, CONTRIBUTING.md update
   ↓
Week 3: Phase 3a (Coverage Push - Start)
         - cmd/confvis tests, cli unit tests
   ↓
Week 4: Phase 4 (Security + Trivy Source) ◄── CRITICAL PATH
         - Implement internal/sources/trivy/*
         - security.yml, SECURITY.md
   ↓
Week 5: Phase 3b (Coverage Push - Complete)
         - Finish coverage to 80%+
         - Trivy source tests included
   ↓
Week 6: Phase 5 (Dogfooding)
         - dogfood.yml workflow
         - README badges
         - SonarCloud setup
   ↓
Week 7: Phase 6 (Release Automation)
         - .goreleaser.yaml, release.yml
   ↓
Week 8: Phase 7 (Polish)
         - Templates, .editorconfig, final docs
```

**Note:** Phase 4 (Trivy source) must complete before Phase 5 (Dogfooding) since the dogfooding workflow depends on `confvis fetch trivy`.

---

## Files to Create/Modify

### New Files - Infrastructure
- `.golangci.yml`
- `Makefile`
- `docker-compose.yml` (local SonarQube)
- `.github/workflows/ci.yml`
- `.github/workflows/security.yml`
- `.github/workflows/dogfood.yml`
- `.github/workflows/release.yml`
- `.github/dependabot.yml`
- `.github/PULL_REQUEST_TEMPLATE.md`
- `.github/ISSUE_TEMPLATE/bug_report.md`
- `.github/ISSUE_TEMPLATE/feature_request.md`
- `.pre-commit-config.yaml`
- `commitlint.config.js`
- `.goreleaser.yaml`
- `.editorconfig`
- `SECURITY.md`
- `sonar-project.properties` (SonarQube config)
- `badges/` directory (auto-generated)

### New Files - Trivy Source (Phase 4)
- `internal/sources/trivy/trivy.go`
- `internal/sources/trivy/client.go`
- `internal/sources/trivy/types.go`
- `internal/sources/trivy/trivy_test.go`
- `internal/cli/fetch.go` (add trivy import)

### Modified Files
- `CONTRIBUTING.md` - Add conventional commits, pre-commit, coverage sections
- `README.md` - Add badges, update with new development workflow
- `docs/sources.md` - Add Trivy source documentation
- `docs/cli-reference.md` - Add Trivy fetch documentation
- Various `*_test.go` files - Coverage improvements

---

## Verification

After each phase, verify with:

```bash
# Phase 1
make lint && make test && make coverage

# Phase 2
pre-commit run --all-files
echo "feat: test" | npx commitlint

# Phase 3
go test -coverprofile=cov.out ./...
go tool cover -func=cov.out | grep total  # Should show ≥80%

# Phase 4
# Check GitHub Security tab for CodeQL results
# Check Snyk dashboard for vulnerability scan

# Phase 5
./confvis aggregate -c coverage.json:40 -c ci.json:30 -c security.json:30 -o ./badges
ls badges/  # Should contain badge.svg and dashboard/

# Phase 6
git tag v0.1.0 && git push --tags
# Check GitHub Releases for binaries

# Phase 7
# Manual review of all documentation
```

---

## Service Setup Required

| Service | Action | Free Tier |
|---------|--------|-----------|
| Codecov | Connect repo at codecov.io | Unlimited (public OSS) |
| SonarCloud | Connect repo at sonarcloud.io | Unlimited (public OSS) |
| Trivy | No account needed | Fully open source |
| GitHub CodeQL | Enable in Security tab | Included |

**Note:** All services are 100% free for public open source repositories.

### Secrets to Configure

- `CODECOV_TOKEN` - From Codecov settings → Repository → Settings
- `SONAR_TOKEN` - From SonarCloud → My Account → Security → Generate Token

### Local Development Setup

For local SonarQube feedback (optional but recommended):

```yaml
# docker-compose.yml
services:
  sonarqube:
    image: sonarqube:lts-community
    ports:
      - "9000:9000"
    volumes:
      - sonarqube_data:/opt/sonarqube/data
      - sonarqube_logs:/opt/sonarqube/logs
      - sonarqube_extensions:/opt/sonarqube/extensions

volumes:
  sonarqube_data:
  sonarqube_logs:
  sonarqube_extensions:
```

```bash
# Start local SonarQube
docker-compose up -d

# Run local analysis (after sonar-scanner is installed)
sonar-scanner -Dsonar.host.url=http://localhost:9000

# Or use confvis to fetch from local instance
confvis fetch sonarqube -p confvis --url http://localhost:9000 -o quality.json
```
