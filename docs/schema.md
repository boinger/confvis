# Schema Reference

confvis reads confidence reports in JSON or YAML format. This document describes the schema.

## Schema Overview

```json
{
  "title": "string (required)",
  "score": "integer (0-100, auto-calculated if omitted)",
  "threshold": "integer (required, 0-100)",
  "description": "string (optional)",
  "thresholds": {
    "greenAbove": "integer (optional, 0-100, default 75)",
    "yellowAbove": "integer (optional, 0-100, default 50)"
  },
  "factors": [
    {
      "name": "string (required)",
      "score": "integer (required, 0-100)",
      "weight": "integer (required)",
      "description": "string (optional)",
      "url": "string (optional)"
    }
  ],
  "version": "string (optional)",
  "generatedAt": "string (optional, ISO 8601 timestamp)",
  "source": "string (optional)",
  "passLabel": "string (optional, default 'PASS')",
  "failLabel": "string (optional, default 'FAIL')"
}
```

## Field Reference

### Report Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | Yes | Report title, displayed in dashboard header |
| `score` | integer | No* | Overall confidence score (0-100). Auto-calculated from factors if omitted |
| `threshold` | integer | Yes | Minimum passing score. Score >= threshold = PASS |
| `description` | string | No | Detailed report description |
| `thresholds` | object | No | Custom color thresholds (see below) |
| `factors` | array | No | Contributing factors breakdown |
| `version` | string | No | Report version identifier |
| `generatedAt` | string | No | ISO 8601 timestamp when report was generated |
| `source` | string | No | Origin of the report (e.g., "ci-pipeline", "manual") |
| `passLabel` | string | No | Custom label for passing status (default: "PASS") |
| `failLabel` | string | No | Custom label for failing status (default: "FAIL") |

*Score is auto-calculated as a weighted average if omitted and factors are present.

### Color Thresholds Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `greenAbove` | integer | 75 | Minimum score for green (success) color |
| `yellowAbove` | integer | 50 | Minimum score for yellow (warning) color |

Note: `greenAbove` must be >= `yellowAbove`. Scores below `yellowAbove` display in red.

### Factor Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Factor name (e.g., "Test Coverage") |
| `score` | integer | Yes | Factor score (0-100) |
| `weight` | integer | Yes | Relative weight in calculations |
| `threshold` | integer | No | Minimum acceptable score for this factor (0-100). When set, `confvis gauge` fails if factor score is below this threshold |
| `description` | string | No | Explanation of this factor |
| `url` | string | No | Link to detailed report (clickable in dashboard and markdown output) |

## Examples

### Minimal Report

```json
{
  "title": "Build Status",
  "score": 80,
  "threshold": 70
}
```

### Full Report with Factors

```json
{
  "title": "Code Quality Report",
  "score": 85,
  "threshold": 75,
  "description": "Overall code quality assessment for the project.",
  "factors": [
    {
      "name": "Test Coverage",
      "score": 92,
      "weight": 30,
      "description": "Percentage of code covered by tests"
    },
    {
      "name": "Code Complexity",
      "score": 78,
      "weight": 25,
      "description": "Cyclomatic complexity within acceptable range"
    },
    {
      "name": "Documentation",
      "score": 88,
      "weight": 20,
      "description": "API documentation completeness"
    },
    {
      "name": "Security Scan",
      "score": 80,
      "weight": 25,
      "description": "No critical vulnerabilities detected"
    }
  ]
}
```

### Report with Auto-Calculated Score

When `score` is omitted but factors are present, the score is automatically
calculated as a weighted average:

```json
{
  "title": "Auto-Calculated Report",
  "threshold": 75,
  "factors": [
    {"name": "Test Coverage", "score": 80, "weight": 50},
    {"name": "Lint Score", "score": 60, "weight": 50}
  ]
}
```

This calculates to: (80×50 + 60×50) / 100 = **70**

### Report with Custom Color Thresholds

```json
{
  "title": "High Standards Report",
  "score": 85,
  "threshold": 80,
  "thresholds": {
    "greenAbove": 90,
    "yellowAbove": 70
  }
}
```

With these thresholds, score 85 displays in yellow (warning) instead of green.

### Report with Metadata

```json
{
  "title": "CI Pipeline Report",
  "score": 92,
  "threshold": 80,
  "version": "1.2.0",
  "generatedAt": "2024-01-15T10:30:00Z",
  "source": "github-actions"
}
```

Metadata fields are included in JSON output format (`--format json`).

### Report with Custom Labels

```json
{
  "title": "Security Audit",
  "score": 95,
  "threshold": 90,
  "passLabel": "COMPLIANT",
  "failLabel": "NON-COMPLIANT"
}
```

Custom labels replace the default "PASS"/"FAIL" text in the gauge.

### Factor with URL

```json
{
  "title": "Code Quality",
  "score": 88,
  "threshold": 75,
  "factors": [
    {
      "name": "Test Coverage",
      "score": 92,
      "weight": 50,
      "description": "Unit and integration test coverage",
      "url": "https://codecov.io/gh/org/repo"
    },
    {
      "name": "Static Analysis",
      "score": 84,
      "weight": 50,
      "description": "SonarQube quality gate results",
      "url": "https://sonarcloud.io/project/overview?id=org_repo"
    }
  ]
}
```

Factor URLs can link to detailed reports or external tools.

### Factors with Per-Factor Thresholds

```json
{
  "title": "Quality Gates",
  "threshold": 75,
  "factors": [
    {
      "name": "Test Coverage",
      "score": 85,
      "weight": 40,
      "threshold": 80,
      "description": "Must maintain 80% coverage"
    },
    {
      "name": "Security Scan",
      "score": 95,
      "weight": 35,
      "threshold": 90,
      "description": "Security requires 90% minimum"
    },
    {
      "name": "Documentation",
      "score": 70,
      "weight": 25,
      "description": "No threshold - informational only"
    }
  ]
}
```

Per-factor thresholds allow you to enforce minimum scores on individual factors,
independent of the aggregate score. The build fails if any factor's score drops
below its threshold, even if the overall score passes.

Thresholds can also be set via CLI (`--factor-threshold "Coverage:80"`) or config
file (`gauge.factor_thresholds`). CLI takes precedence over config, which takes
precedence over the factor's own threshold field.

## YAML Format

All examples above can be written in YAML. confvis auto-detects the format based on file extension (`.yaml` or `.yml`) or content.

### Minimal Report

```yaml
title: Build Status
score: 80
threshold: 70
```

### Full Report with Factors

```yaml
title: Code Quality Report
score: 85
threshold: 75
description: Overall code quality assessment for the project.
factors:
  - name: Test Coverage
    score: 92
    weight: 30
    description: Percentage of code covered by tests
  - name: Code Complexity
    score: 78
    weight: 25
    description: Cyclomatic complexity within acceptable range
  - name: Documentation
    score: 88
    weight: 20
    description: API documentation completeness
  - name: Security Scan
    score: 80
    weight: 25
    description: No critical vulnerabilities detected
```

### Report with Custom Thresholds

```yaml
title: High Standards Report
score: 85
threshold: 80
thresholds:
  greenAbove: 90
  yellowAbove: 70
```

### Factors with Per-Factor Thresholds

```yaml
title: Quality Gates
threshold: 75
factors:
  - name: Test Coverage
    score: 85
    weight: 40
    threshold: 80
    description: Must maintain 80% coverage
  - name: Security Scan
    score: 95
    weight: 35
    threshold: 90
    description: Security requires 90% minimum
  - name: Documentation
    score: 70
    weight: 25
    description: No threshold - informational only
```

## Score Thresholds

The gauge uses color coding based on score. Default thresholds:

| Score Range | Color | Meaning |
|-------------|-------|---------|
| 75-100 | Green | Good |
| 50-74 | Yellow | Warning |
| 0-49 | Red | Critical |

You can customize these thresholds in the JSON or via CLI flags (`--green-above`, `--yellow-above`).

The pass/fail indicator is based solely on `score >= threshold` (separate from color thresholds).

## Validation

- `score` must be between 0 and 100
- `threshold` must be between 0 and 100
- `title` cannot be empty
- Factor `name` cannot be empty
- Factor `score` must be between 0 and 100
- Factor `weight` must be between 0 and 100
- Factor `threshold` (if specified) must be between 0 and 100
- `thresholds.greenAbove` must be between 0 and 100
- `thresholds.yellowAbove` must be between 0 and 100
- `thresholds.greenAbove` must be >= `thresholds.yellowAbove`

Invalid input or missing required fields will cause confvis to exit with an error.
