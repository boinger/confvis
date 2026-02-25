# Plan: Four CLI UX Enhancements

## Feature 1: Smart `--format` default for `gauge`
- [x] 1a. Change `--format` flag default from `"svg"` to `""` in gauge.go init()
- [x] 1b. Add format resolution logic in `gaugeImpl()` — empty→text for stdout, svg for file
- [x] 1c. Update help text for `--format` flag
- [x] 1d. Update `validateGaugeInputs` to accept empty format (pre-resolution)
- [x] 1e. Update test helper `defaultGaugeDeps` and fix affected tests
- [x] 1f. Add new tests: stdout→text default, file→svg default, explicit format always honored

## Feature 2: `gate` emits `$GITHUB_OUTPUT`
- [x] 2a. Add `GitHubOutputFile string` to `GateDeps`
- [x] 2b. Populate from `os.Getenv("GITHUB_OUTPUT")` in `runGate()`
- [x] 2c. Write `gate_result` and `gate_score` after `CheckThresholds()` in `gateImpl()`
- [x] 2d. Add tests: writes correct values to temp file; no-op when unset

## Feature 3: `comment github` gate awareness
- [x] 3a. Add `GateFailUnder` and `GateFailOnRegression` to `CommentGitHubDeps`
- [x] 3b. Add cobra flags `--gate-fail-under` and `--gate-fail-on-regression`
- [x] 3c. Thread values through to `generateCommentBody()`
- [x] 3d. Add `writeGateWarning()` in `gauge_format.go`
- [x] 3e. Call `writeGateWarning()` from `writeGitHubComment()`
- [x] 3f. Add tests: warning present when gate would fail, absent when passing

## Feature 4: `--version` / `version` command
- [x] 4a. Add `SetVersion()` func in `root.go`
- [x] 4b. Wire `var version = "dev"` and `cli.SetVersion(version)` in `main.go`
- [x] 4c. Update comment footer to use CLI version via `rootCmd.Version`

## Verification
- [x] V1. `go build ./...`
- [x] V2. `golangci-lint run ./...` — zero issues
- [x] V3. `go test ./... -count=1` — all pass
