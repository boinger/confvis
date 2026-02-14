# Future Feature Ideas

Captured for later consideration. Not prioritized or committed to.

## Parallel Source Fetching

Sources are fetched sequentially. A bounded goroutine pool would reduce wall-clock time for multi-source configs. The `Source` interface is clean enough to wrap with `sync.WaitGroup` + error group. Retry/backoff in httpclient makes this safer (transient failures don't immediately kill a goroutine).

## Trend Alerts

History tracking exists but is display-only. Detect score regressions across N runs and surface warnings ("score dropped 15% over last 5 runs") without needing explicit thresholds. Useful in CI where nobody looks at the dashboard. Could integrate with the existing check/comment output.

## Source Health/Status Reporting

When a source fails (API down, token expired, misconfigured), errors are opaque. A `confvis status` command that pings each configured source and reports connectivity/auth health would save debugging time. Could also validate tokens have correct scopes.

## Config Validation Command

A `confvis validate` that checks the config file for common mistakes: unknown source names, missing required fields, weights not summing to 100%, invalid threshold ranges. Run before the full pipeline to catch config errors early.
