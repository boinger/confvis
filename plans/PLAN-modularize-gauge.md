# Plan: Modularize `gauge.go` (817 → ~340 lines)

## Steps

- [x] Step 1: Create `gauge_threshold.go` — move `parseFactorThresholds`, `parseFactorThresholdSpec`, `checkFactorThresholds`
- [x] Step 2: Create `gauge_history.go` — move `loadHistoryScores`, `appendToHistory`, `resolveHistoryStorage`, `loadBaselineForComparison`, `resolveBaseline`
- [x] Step 3: Create `gauge_badge.go` — move `generateSVGBadge`, `generateFlatBadge`, `generateSparklineBadge`, `generateGaugeBadge`
- [x] Step 4: Create `gauge_format.go` — move `writeFormatOutput`, `writeJSON`, `writeText`, `writeMarkdown`, `writeGitHubComment` + 4 sub-functions
- [x] Step 5: Final cleanup + verify all imports, run lint + full tests, update docs

## Verification (after each step)

```
go build ./...
golangci-lint run ./...
go test ./internal/cli/... -count=1
```
