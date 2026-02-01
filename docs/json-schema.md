# JSON Schema

confvis reads confidence reports in JSON format. This document describes the schema.

## Schema Overview

```json
{
  "title": "string (required)",
  "score": "integer (required, 0-100)",
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
      "description": "string (optional)"
    }
  ]
}
```

## Field Reference

### Report Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | Yes | Report title, displayed in dashboard header |
| `score` | integer | Yes | Overall confidence score (0-100) |
| `threshold` | integer | Yes | Minimum passing score. Score >= threshold = PASS |
| `description` | string | No | Detailed report description |
| `thresholds` | object | No | Custom color thresholds (see below) |
| `factors` | array | No | Contributing factors breakdown |

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
| `description` | string | No | Explanation of this factor |

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
- `thresholds.greenAbove` must be between 0 and 100
- `thresholds.yellowAbove` must be between 0 and 100
- `thresholds.greenAbove` must be >= `thresholds.yellowAbove`

Invalid JSON or missing required fields will cause confvis to exit with an error.
