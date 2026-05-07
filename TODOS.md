# TODOs

## Architecture

### F4: Migrate checks client to shared httpclient
**Priority:** P2
**Category:** Architecture
**Location:** `internal/checks/github.go:58-94`
**What:** The `GitHubClient` manages its own `http.Client`, auth headers, and error formatting, duplicating patterns from `internal/sources/httpclient`.
**Why:** Reduces code duplication and ensures the checks client benefits from shared infrastructure improvements (e.g., `parseRetryAfter` and `sleepWithContext` are now in `internal/httputil`, but the retry loop, retryable error type, and status code map are still duplicated).
**Context:** Requires adding POST/PATCH/DELETE support to `httpclient.Client` (currently GET-only). The GET-only retry constraint in the checks client means the httpclient migration should preserve the ability to disable retry per-method. Consider a `httpclient.Do()` method with a `Retryable bool` option.
**Depends on:** None
**Effort:** M (human) / S (CC)

## Tests

### T1: Extract writeMockScript test helper when a 4th consumer emerges
**Priority:** P3
**Category:** Tests
**Location:** `internal/sources/{trufflehog,gitleaks,gosec}/*_test.go` (writeMockScript function)
**What:** Three source packages duplicate a `writeMockScript(t, dir, name, scriptOutput, exitCode)` helper. Bodies are structurally identical; only the heredoc sentinel label differs (TRUFFLEEOF / LEAKSEOF / GOSECEOF).
**Why:** Not worth extracting for three consumers (grype/trivy use the same inline pattern). A 4th mock-script consumer would push us past the duplicate-tolerance threshold.
**Trigger:** Adding mock-bash-script tests to a 4th source package.
**Proposed home:** `internal/sources/internal/testutil/mockscript.go` with a single exported `WriteMockScript` function using a unified `MOCKEOF` heredoc sentinel.
**Depends on:** A 4th consumer existing.
**Effort:** S (human) / XS (CC)

## Infrastructure / CI

### I1: SonarSource scan action runs on Node.js 20 (deprecated)
**Priority:** P2 (deadline-driven)
**Category:** Infrastructure
**Location:** `.github/workflows/ci.yml` — SonarSource/sonarqube-scan-action step
**What:** The pinned `SonarSource/sonarqube-scan-action@a31c9398...` runs on Node.js 20, which GitHub Actions is deprecating.
**Why:** GitHub forces Node 24 as default on 2026-06-02; Node 20 is removed from runners on 2026-09-16. After that, the action fails unless we opt into the temporary unsecure override (`ACTIONS_ALLOW_USE_UNSECURE_NODE_VERSION`).
**Context:** Resolution is upstream — upgrade to a newer `SonarSource/sonarqube-scan-action` tag when one supporting Node 24 is released. Dependabot will likely surface this; this TODO exists so we don't miss it if Dependabot's cadence lags the deprecation date.
**Depends on:** Upstream release from SonarSource.
**Effort:** XS (pin swap).

### I2: Coveralls upload action occasionally exit-127s
**Priority:** P3
**Category:** Infrastructure / CI reliability
**Location:** `.github/workflows/ci.yml` — `coverallsapp/github-action@v2` step (pinned to SHA `648a8eb7...`)
**What:** The upload step has been observed to fail mid-install with `coveralls: command not found` (exit 127) on an otherwise-successful test job. The action's `fail-on-error: false` input does not cover the wrapper script bug.
**Why:** Transient, not caused by our code. A rerun of the failed job succeeds. Worth monitoring — if the pattern repeats, consider bumping to a newer `coverallsapp` tag or switching to installing the coveralls CLI at a known version inside the workflow.
**Depends on:** Observation of repeat failures.
**Effort:** S (upgrade pin) / M (switch strategy).

### I3: Codecov fetch logs 401 Invalid token during Confidence Badge runs
**Priority:** P3
**Category:** Infrastructure / CI reliability (cosmetic)
**Location:** `.github/workflows/dogfood.yml` — Codecov fetch step (`confvis fetch codecov`)
**What:** The codecov fetch step occasionally logs `Error: fetching from codecov: API returned status 401: {"detail":"Invalid token."}` and exits 1. The step is guarded by `continue-on-error: true`, so the workflow continues without coverage data and the badge regenerates from the remaining sources — the 401 is log noise, not a workflow failure. Observed across four consecutive intermediate merge commits on 2026-05-06; recovered without intervention on subsequent commits.
**Why:** Transient, not caused by our code. Either the `CODECOV_TOKEN` secret is rotated/limited and revalidates between runs, or codecov's auth backend rate-limits rapid token reuse. Distinct from I2 (Coveralls) — separate observability sink, separate failure mode.
**Context:** Cosmetic only. The workflow's existing `continue-on-error: true` already prevents the 401 from breaking badge generation; the resulting badge just omits coverage data for that run. If the noise becomes correlated with sustained badge degradation, consider verifying the token's scope/expiry or adding a quick retry on 401.
**Depends on:** Observation of repeat failures or correlation with token-rotation events.
**Effort:** S (token scope check).
