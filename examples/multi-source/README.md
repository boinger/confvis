# Multi-Source Score Aggregation

This example shows how to aggregate confidence scores from multiple sources into a single report.

## Usage

```bash
# Make executable
chmod +x aggregate.sh

# Run aggregation
./aggregate.sh

# Generate badge
confvis generate -c confidence.json -o ./badges
```

## How It Works

The `aggregate.sh` script:

1. **Collects test coverage** using `go test -cover`
2. **Runs linting** with `golangci-lint`
3. **Runs security scans** with `gosec` (if installed)
4. **Calculates weighted average** with configurable weights
5. **Outputs `confidence.json`** ready for confvis

## Customization

Edit the script to:

- Add/remove metric sources
- Adjust weights for each factor
- Change scoring formulas
- Add other language tools (Jest, pytest, ESLint, etc.)

## Weights

Default weights:
- Test Coverage: 40%
- Lint Score: 35%
- Security Scan: 25%

Modify the `OVERALL` calculation line to change weights.

## CI Integration

Add to your CI pipeline after running tests:

```yaml
- name: Aggregate metrics
  run: ./examples/multi-source/aggregate.sh

- name: Generate badge
  run: confvis generate -c confidence.json -o ./badges
```
