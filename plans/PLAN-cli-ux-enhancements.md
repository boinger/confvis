# Plan: Four CLI UX Enhancements

## Feature 1: Smart `--format` default for `gauge`
- [ ] 1a. Change `--format` flag default from `"svg"` to `""` in gauge.go init()
- [ ] 1b. Add format resolution logic in `gaugeImpl()` — empty→text for stdout, svg for file
- [ ] 1c. Update help text for `--format` flag
- [ ] 1d. Update `validateGaugeInputs` to accept empty format (pre-resolution)
- [ ] 1e. Update test helper `defaultGaugeDeps` and fix affected tests
- [ ] 1f. Add new tests: stdout→text default, file→svg default, explicit format always honored

## Feature 2: `gate` emits `$GITHUB_OUTPUT`
- [ ] 2a. Add `GitHubOutputFile string` to `GateDeps`
- [ ] 2b. Populate from `os.Getenv("GITHUB_OUTPUT")` in `runGate()`
- [ ] 2c. Write `gate_result` and `gate_score` after `CheckThresholds()` in `gateImpl()`
- [ ] 2d. Add tests: writes correct values to temp file; no-op when unset

## Feature 3: `comment github` gate awareness
- [ ] 3a. Add `GateFailUnder` and `GateFailOnRegression` to `CommentGitHubDeps`
- [ ] 3b. Add cobra flags `--gate-fail-under` and `--gate-fail-on-regression`
- [ ] 3c. Thread values through to `generateCommentBody()`
- [ ] 3d. Add `writeGateWarning()` in `gauge_format.go`
- [ ] 3e. Call `writeGateWarning()` from `writeGitHubComment()`
- [ ] 3f. Add tests: warning present when gate would fail, absent when passing

## Feature 4: `--version` / `version` command
- [ ] 4a. Add `SetVersion()` func in `root.go`
- [ ] 4b. Wire `var version = "dev"` and `cli.SetVersion(version)` in `main.go`
- [ ] 4c. Update comment footer to use CLI version via `rootCmd.Version`

## Verification
- [ ] V1. `go build ./...`
- [ ] V2. `golangci-lint run ./...` — zero issues
- [ ] V3. `go test ./... -count=1` — all pass
