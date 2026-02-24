# Plan: `confvis gate` Command

## Steps

- [x] 1. Explore existing gauge code and understand patterns
- [x] 2. Create `internal/cli/threshold.go` — extract ThresholdConfig, CheckThresholds(), move checkFactorThresholds()
- [x] 3. Create `internal/cli/baseline_loader.go` — extract BaselineConfig, LoadBaseline(), move resolveBaseline()
- [x] 4. Refactor `gauge.go` to use shared threshold/baseline code (no behavior change)
- [x] 5. Clean up `gauge_history.go` and `gauge_threshold.go` after extraction
- [x] 6. Create `internal/cli/gate.go` — GateDeps, gateImpl(), cobra command
- [x] 7. Write `internal/cli/gate_test.go` and `internal/cli/threshold_test.go`
- [x] 8. Update docs: `docs/cli-reference.md` and `README.md`
- [x] 9. Verify: build, lint, full test suite, manual testing

## Key Design Decisions

- `CheckThresholds()` returns structured `ThresholdResult` so each command can format output differently
- `LoadBaseline()` takes a `BaselineConfig` struct decoupled from `*GaugeDeps`
- `checkFactorThresholds()` moves from `gauge_threshold.go` to `threshold.go` (shared)
- `parseFactorThresholds()` stays in `gauge_threshold.go` (both commands use it)
- Gate output formatting is gate-specific (Score: N/100, Threshold: N ✓/✗)
- Gauge output formatting preserved exactly (stderr messages, ExitFunc calls)

## Modified Files

- `internal/cli/threshold.go` (new)
- `internal/cli/baseline_loader.go` (new)
- `internal/cli/gate.go` (new)
- `internal/cli/gate_test.go` (new)
- `internal/cli/threshold_test.go` (new)
- `internal/cli/gauge.go` (modified)
- `internal/cli/gauge_history.go` (modified)
- `internal/cli/gauge_threshold.go` (modified)
- `internal/cli/config.go` (modified — add gate flag bindings)
- `docs/cli-reference.md` (modified)
- `README.md` (modified)
