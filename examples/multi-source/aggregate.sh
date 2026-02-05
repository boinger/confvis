#!/bin/bash
# aggregate.sh - Aggregate metrics from multiple sources into confidence.json
#
# This script demonstrates how to combine metrics from various tools
# (test coverage, linting, security scans) into a single confidence report.
#
# Usage: ./aggregate.sh

set -e

# Default values
COVERAGE=0
LINT_SCORE=100
SECURITY_SCORE=100

# ------------------------------------------------------------------------------
# Test Coverage (Go)
# ------------------------------------------------------------------------------
if command -v go &> /dev/null && [[ -f go.mod ]]; then
    echo "Collecting Go test coverage..."
    if go test -coverprofile=coverage.out ./... 2>/dev/null; then
        COVERAGE=$(go tool cover -func=coverage.out 2>/dev/null | grep total | awk '{print int($3)}')
        rm -f coverage.out
    fi
    echo "  Coverage: ${COVERAGE}%"
fi

# ------------------------------------------------------------------------------
# Linting (golangci-lint)
# ------------------------------------------------------------------------------
if command -v golangci-lint &> /dev/null; then
    echo "Running linter..."
    LINT_OUTPUT=$(golangci-lint run ./... 2>&1 || true)
    LINT_ISSUES=$(echo "$LINT_OUTPUT" | grep -c "^" || echo 0)

    # Deduct 5 points per issue, minimum 0
    LINT_SCORE=$((100 - LINT_ISSUES * 5))
    [ $LINT_SCORE -lt 0 ] && LINT_SCORE=0
    echo "  Lint issues: ${LINT_ISSUES}, Score: ${LINT_SCORE}%"
fi

# ------------------------------------------------------------------------------
# Security Scan (gosec) - optional
# ------------------------------------------------------------------------------
if command -v gosec &> /dev/null; then
    echo "Running security scan..."
    SECURITY_OUTPUT=$(gosec -quiet ./... 2>&1 || true)
    SECURITY_ISSUES=$(echo "$SECURITY_OUTPUT" | grep -c "Severity:" || echo 0)

    # Deduct 10 points per security issue
    SECURITY_SCORE=$((100 - SECURITY_ISSUES * 10))
    [ $SECURITY_SCORE -lt 0 ] && SECURITY_SCORE=0
    echo "  Security issues: ${SECURITY_ISSUES}, Score: ${SECURITY_SCORE}%"
else
    echo "Skipping security scan (gosec not installed)"
fi

# ------------------------------------------------------------------------------
# Calculate Overall Score (weighted average)
# ------------------------------------------------------------------------------
# Weights: Coverage 40%, Linting 35%, Security 25%
OVERALL=$(( (COVERAGE * 40 + LINT_SCORE * 35 + SECURITY_SCORE * 25) / 100 ))

echo ""
echo "Overall score: ${OVERALL}%"

# ------------------------------------------------------------------------------
# Generate confidence.json
# ------------------------------------------------------------------------------
cat > confidence.json << EOF
{
  "title": "Code Quality Report",
  "score": ${OVERALL},
  "threshold": 75,
  "description": "Aggregated from test coverage, linting, and security scans.",
  "factors": [
    {
      "name": "Test Coverage",
      "score": ${COVERAGE},
      "weight": 40,
      "description": "Percentage of code covered by tests"
    },
    {
      "name": "Lint Score",
      "score": ${LINT_SCORE},
      "weight": 35,
      "description": "Code quality from static analysis"
    },
    {
      "name": "Security Scan",
      "score": ${SECURITY_SCORE},
      "weight": 25,
      "description": "Results from security vulnerability scan"
    }
  ]
}
EOF

echo "Generated confidence.json"
