# TODOs

## Opportunities (from codebase audit 2026-03-29)

### F3: CLI test suite performance
**Priority:** P3
**Category:** Tests
**Location:** `internal/cli/` (package-level)
**What:** The `internal/cli` test suite takes 69s, 5-10x slower than other packages.
**Why:** Slow test suites erode developer velocity and CI feedback loop time.
**Context:** Most packages complete in 1-4s. Profile with `go test -v ./internal/cli/ -count=1` to identify the slowest individual tests. Consider parallelizing independent test cases with `t.Parallel()`.
**Effort:** S (human) / S (CC)

### F4: Migrate checks client to shared httpclient
**Priority:** P2
**Category:** Architecture
**Location:** `internal/checks/github.go:58-94`
**What:** The `GitHubClient` manages its own `http.Client`, auth headers, and error formatting, duplicating patterns from `internal/sources/httpclient`.
**Why:** Reduces code duplication and ensures the checks client benefits from shared infrastructure improvements (e.g., `parseRetryAfter` and `sleepWithContext` are now in `internal/httputil`, but the retry loop, retryable error type, and status code map are still duplicated).
**Context:** Requires adding POST/PATCH/DELETE support to `httpclient.Client` (currently GET-only). The GET-only retry constraint in the checks client means the httpclient migration should preserve the ability to disable retry per-method. Consider a `httpclient.Do()` method with a `Retryable bool` option.
**Depends on:** None
**Effort:** M (human) / S (CC)
