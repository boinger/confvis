# Contributing New Sources

This guide explains how to add new metric sources to confvis. Sources are modular integrations that fetch metrics from external systems and convert them into confidence reports.

## Architecture Overview

### Source Interface

All sources implement the `Source` interface defined in `internal/sources/source.go`:

```go
type Source interface {
    // Name returns the unique identifier for this source (e.g., "sonarqube").
    Name() string

    // Fetch retrieves metrics and converts them to a confidence report.
    Fetch(ctx context.Context, opts Options) (*confidence.Report, error)
}
```

### Options Struct

The `Options` struct provides common configuration for all sources:

```go
type Options struct {
    URL       string            // API server URL
    Project   string            // Project key/identifier
    Token     string            // API authentication token
    Branch    string            // Branch to query (optional)
    Title     string            // Report title (optional)
    Threshold int               // Pass/fail threshold (default 75)
    Timeout   int               // HTTP timeout in seconds

    Extra     map[string]string // Source-specific options
}
```

Use the `Extra` map for source-specific options that don't fit the common fields.

### Registry Pattern

Sources register themselves at package initialization using `init()`:

```go
func init() {
    sources.Register(&Source{})
}
```

The registry is a global map that makes sources available by name. Duplicate registrations panic to catch configuration errors early.

### Report Structure

The `confidence.Report` returned by `Fetch()`:

```go
type Report struct {
    Title       string    // Display name
    Score       int       // 0-100, calculated from factors
    Threshold   int       // Minimum acceptable score
    Factors     []Factor  // Score breakdown
    Source      string    // Source identifier
    GeneratedAt string    // RFC3339 timestamp
    // ... additional optional fields
}

type Factor struct {
    Name        string // Factor name (e.g., "Critical Vulnerabilities")
    Score       int    // 0-100
    Weight      int    // Relative importance
    Description string // Optional details
    URL         string // Link to metrics (optional)
}
```

## Step-by-Step Tutorial

### 1. Create the Package Directory

Create a new directory under `internal/sources/`:

```
internal/sources/mysource/
├── mysource.go       # Main implementation
├── client.go         # API client (if needed)
├── types.go          # Type definitions
└── mysource_test.go  # Tests
```

### 2. Define Types (types.go)

Define the types that represent your source's API responses:

```go
package mysource

// APIResponse represents the response from the external API.
type APIResponse struct {
    ProjectID string       `json:"project_id"`
    Metrics   MetricsData  `json:"metrics"`
}

type MetricsData struct {
    Critical int `json:"critical"`
    High     int `json:"high"`
    Medium   int `json:"medium"`
    Low      int `json:"low"`
}
```

### 3. Implement the Source (mysource.go)

```go
package mysource

import (
    "context"
    "fmt"
    "time"

    "github.com/boinger/confvis/internal/confidence"
    "github.com/boinger/confvis/internal/sources"
)

const sourceName = "mysource"

// Environment variable names for configuration.
const (
    EnvToken  = "MYSOURCE_TOKEN"
    EnvAPIURL = "MYSOURCE_URL"
)

// ConfigResolver handles token/URL resolution with proper precedence.
var configResolver = &sources.ConfigResolver{
    SourceName:     sourceName,
    TokenEnvVar:    EnvToken,
    URLEnvVar:      EnvAPIURL,
    TokenRequired:  true,   // Set based on your API requirements
    URLRequired:    false,  // Set to false if you have a default
    DefaultTimeout: 30 * time.Second,
}

// Scoring constants - adjust for your source.
const (
    PenaltyCritical = 25  // Points deducted per critical issue
    PenaltyHigh     = 15
    PenaltyMedium   = 5
    PenaltyLow      = 2

    WeightCritical  = 40  // Relative weight (not percentage)
    WeightHigh      = 30
    WeightMedium    = 20
    WeightLow       = 10
)

// Source implements sources.Source for MySource.
type Source struct{}

func init() {
    sources.Register(&Source{})
}

// Name returns the source identifier.
func (s *Source) Name() string {
    return sourceName
}

// Fetch retrieves metrics and converts them to a confidence report.
func (s *Source) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
    // Resolve configuration with proper precedence
    cfg, err := configResolver.Resolve(opts)
    if err != nil {
        return nil, err
    }

    // Validate required options
    if opts.Project == "" {
        return nil, fmt.Errorf("project required: use --project flag")
    }

    // Create client and fetch data
    client := NewClient(cfg.URL, cfg.Token, cfg.Timeout)
    data, err := client.FetchMetrics(ctx, opts.Project)
    if err != nil {
        return nil, err
    }

    // Build factors with severity-based scoring
    factors := []confidence.Factor{
        {
            Name:        "Critical Issues",
            Score:       SeverityScore(data.Metrics.Critical, PenaltyCritical),
            Weight:      WeightCritical,
            Description: fmt.Sprintf("%d critical", data.Metrics.Critical),
        },
        {
            Name:        "High Issues",
            Score:       SeverityScore(data.Metrics.High, PenaltyHigh),
            Weight:      WeightHigh,
            Description: fmt.Sprintf("%d high", data.Metrics.High),
        },
        {
            Name:        "Medium Issues",
            Score:       SeverityScore(data.Metrics.Medium, PenaltyMedium),
            Weight:      WeightMedium,
            Description: fmt.Sprintf("%d medium", data.Metrics.Medium),
        },
        {
            Name:        "Low Issues",
            Score:       SeverityScore(data.Metrics.Low, PenaltyLow),
            Weight:      WeightLow,
            Description: fmt.Sprintf("%d low", data.Metrics.Low),
        },
    }

    // Determine title
    title := opts.Title
    if title == "" {
        title = opts.Project
    }

    // Build and return report
    report := &confidence.Report{
        Title:       title,
        Threshold:   opts.Threshold,
        Source:      sourceName,
        GeneratedAt: time.Now().UTC().Format(time.RFC3339),
        Factors:     factors,
    }

    // Calculate weighted score from factors
    report.Score = report.CalculateScore()

    return report, nil
}

// SeverityScore calculates a 0-100 score based on issue count and penalty.
// Returns 100 minus (count * penalty), capped at 0.
func SeverityScore(count, penalty int) int {
    score := 100 - (count * penalty)
    if score < 0 {
        return 0
    }
    return score
}
```

### 4. Create the API Client (client.go)

Use the `httpclient` package for consistent HTTP handling:

```go
package mysource

import (
    "context"
    "fmt"
    "net/http"
    "strings"
    "time"

    "github.com/boinger/confvis/internal/sources/httpclient"
)

const defaultBaseURL = "https://api.mysource.io"

// Client is an HTTP client for the MySource API.
type Client struct {
    baseURL string
    http    *httpclient.Client
}

// NewClient creates a new API client.
func NewClient(baseURL, token string, timeout time.Duration) *Client {
    if baseURL == "" {
        baseURL = defaultBaseURL
    }
    baseURL = strings.TrimRight(baseURL, "/")

    return &Client{
        baseURL: baseURL,
        http: httpclient.New(httpclient.Config{
            BaseURL:  baseURL,
            Token:    token,
            AuthType: httpclient.AuthBearer, // or AuthToken, AuthBasic
            Accept:   "application/json",
            Timeout:  timeout,
        }),
    }
}

// NewClientWithHTTP creates a client with a custom http.Client (for testing).
func NewClientWithHTTP(baseURL, token string, httpClient *http.Client) *Client {
    if baseURL == "" {
        baseURL = defaultBaseURL
    }
    baseURL = strings.TrimRight(baseURL, "/")

    return &Client{
        baseURL: baseURL,
        http: httpclient.NewWithHTTPClient(httpclient.Config{
            BaseURL:  baseURL,
            Token:    token,
            AuthType: httpclient.AuthBearer,
            Accept:   "application/json",
        }, httpClient),
    }
}

// FetchMetrics retrieves metrics for a project.
func (c *Client) FetchMetrics(ctx context.Context, projectID string) (*APIResponse, error) {
    endpoint := fmt.Sprintf("%s/projects/%s/metrics", c.baseURL, projectID)

    var result APIResponse
    if err := c.http.Get(ctx, endpoint, &result); err != nil {
        return nil, err
    }

    return &result, nil
}
```

### 5. Register the Import

Add a blank import to `internal/cli/fetch.go`:

```go
import (
    // ... existing imports ...
    _ "github.com/boinger/confvis/internal/sources/mysource"
)
```

### 6. Add Source-Specific Flags (optional)

If your source needs additional flags, add them to `internal/cli/fetch.go`:

```go
var fetchMyOption string

func init() {
    // ... existing flags ...
    fetchCmd.Flags().StringVar(&fetchMyOption, "my-option", "", "mysource: description")
}

func runFetch(cmd *cobra.Command, args []string) error {
    // ... in the Extra map:
    Extra: map[string]string{
        // ... existing entries ...
        "my-option": fetchMyOption,
    },
}
```

## Testing Patterns

### Mock Interface Pattern

Define an interface for your client to enable mock testing:

```go
// Fetcher defines the interface for testing.
type Fetcher interface {
    FetchMetrics(ctx context.Context, projectID string) (*APIResponse, error)
}

// FetchWithClient allows injecting a mock client for testing.
func (s *Source) FetchWithClient(ctx context.Context, fetcher Fetcher, opts sources.Options, projectID string) (*confidence.Report, error) {
    // ... implementation using fetcher instead of creating a client
}
```

### HTTP Test Server Pattern

```go
func TestFetch_Success(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        if r.URL.Path != "/projects/my-project/metrics" {
            t.Errorf("path = %q, want /projects/my-project/metrics", r.URL.Path)
        }

        // Send response
        resp := APIResponse{
            ProjectID: "my-project",
            Metrics: MetricsData{
                Critical: 0,
                High:     2,
                Medium:   5,
                Low:      10,
            },
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    }))
    defer server.Close()

    client := NewClientWithHTTP(server.URL, "test-token", server.Client())

    data, err := client.FetchMetrics(context.Background(), "my-project")
    if err != nil {
        t.Fatalf("FetchMetrics() error = %v", err)
    }

    // Assertions...
}
```

### Mock Fetcher Pattern

```go
type mockFetcher struct {
    response *APIResponse
    err      error
}

func (m *mockFetcher) FetchMetrics(_ context.Context, _ string) (*APIResponse, error) {
    return m.response, m.err
}

func TestFetchWithClient_Success(t *testing.T) {
    mock := &mockFetcher{
        response: &APIResponse{
            Metrics: MetricsData{Critical: 1, High: 2, Medium: 3, Low: 4},
        },
    }

    s := &Source{}
    report, err := s.FetchWithClient(context.Background(), mock, sources.Options{
        Project: "my-project",
    }, "my-project")

    // Assertions...
}
```

### Test Coverage Requirements

- Test the scoring algorithm with various inputs
- Test configuration validation (missing token, missing project, etc.)
- Test API error handling
- Test title fallback logic
- Test environment variable fallbacks
- Achieve 80%+ code coverage

## Code Templates

### Minimal Source Skeleton

For simple sources that don't need a separate client:

```go
package minimal

import (
    "context"
    "time"

    "github.com/boinger/confvis/internal/confidence"
    "github.com/boinger/confvis/internal/sources"
)

const sourceName = "minimal"

type Source struct{}

func init() {
    sources.Register(&Source{})
}

func (s *Source) Name() string {
    return sourceName
}

func (s *Source) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
    // Implement fetching and scoring logic
    return &confidence.Report{
        Title:       opts.Title,
        Score:       100,
        Threshold:   opts.Threshold,
        Source:      sourceName,
        GeneratedAt: time.Now().UTC().Format(time.RFC3339),
    }, nil
}
```

### Command-Based Source (like Trivy)

For sources that run local commands:

```go
package scanner

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
)

// Client executes commands and parses output.
type Client struct {
    command string
}

// NewClient creates a command executor.
func NewClient(command string) *Client {
    if command == "" {
        command = "default-scanner"
    }
    return &Client{command: command}
}

// Scan runs the scanner command and parses JSON output.
func (c *Client) Scan(ctx context.Context, target string) (*ScanReport, error) {
    args := []string{"--format", "json", target}

    cmd := exec.CommandContext(ctx, c.command, args...)
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("running scanner: %w (stderr: %s)", err, stderr.String())
    }

    var report ScanReport
    if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
        return nil, fmt.Errorf("parsing output: %w", err)
    }

    return &report, nil
}
```

## Checklist

Before submitting a PR for a new source:

- [ ] Implement the `Source` interface (`Name()`, `Fetch()`)
- [ ] Use `ConfigResolver` for token/URL handling with environment variable fallbacks
- [ ] Use `httpclient` package for HTTP calls (for API-based sources)
- [ ] Register in `init()` with `sources.Register()`
- [ ] Add blank import to `internal/cli/fetch.go`
- [ ] Add source-specific flags if needed (in `fetch.go`)
- [ ] Write tests with mocks achieving 80%+ coverage
- [ ] Test configuration validation (missing required fields)
- [ ] Test API/command error handling
- [ ] Add usage example to `fetch.go` help text
- [ ] Update README source list
- [ ] All linting passes with zero issues (`make lint`)
- [ ] All tests pass (`make test`)

## Scoring Guidelines

### Severity-Based Scoring

For vulnerability/issue scanners, use severity-based scoring:

```go
// Score = 100 - (count * penalty), minimum 0
func SeverityScore(count, penalty int) int {
    score := 100 - (count * penalty)
    if score < 0 {
        return 0
    }
    return score
}
```

Suggested penalty values:
| Severity | Penalty | Rationale |
|----------|---------|-----------|
| Critical | 25-33 | 3-4 issues = zero score |
| High | 15-20 | 5-7 issues = zero score |
| Medium | 5-10 | 10-20 issues = zero score |
| Low | 2-5 | 20-50 issues = zero score |

### Weight Distribution

Weights are relative, not percentages. Common patterns:

| Pattern | Critical | High | Medium | Low |
|---------|----------|------|--------|-----|
| Security-focused | 40 | 30 | 20 | 10 |
| Balanced | 25 | 25 | 25 | 25 |
| Low-tolerance | 50 | 30 | 15 | 5 |

### Quality-Based Scoring

For quality metrics (ratings, percentages):

```go
// Convert A-E rating to 0-100 score
func RatingToScore(rating string) int {
    switch rating {
    case "A": return 100
    case "B": return 75
    case "C": return 50
    case "D": return 25
    case "E": return 0
    default:  return 0
    }
}

// Invert percentage (e.g., for duplication)
func InvertPercentage(pct float64) int {
    return int(100 - pct)
}
```

## Questions?

Open an issue on GitHub for questions about implementing new sources.
