# Architecture

## Package Structure

```
confvis/
├── cmd/confvis/main.go     # Entry point, invokes CLI
├── internal/
│   ├── cli/                # Command definitions (cobra)
│   │   ├── root.go         # Root command setup
│   │   ├── gauge.go        # gauge subcommand
│   │   ├── generate.go     # generate subcommand
│   │   ├── aggregate.go    # aggregate subcommand
│   │   ├── fetch.go        # fetch subcommand (external sources)
│   │   ├── baseline.go     # baseline save/show subcommands
│   │   └── check.go        # check github subcommand
│   ├── confidence/         # Core data types and parsing
│   │   ├── types.go        # Report, Factor structs
│   │   └── parser.go       # JSON parsing logic
│   ├── gauge/              # SVG gauge generation
│   │   ├── gauge.go        # SVG rendering with svgo
│   │   └── colors.go       # Color schemes (light/dark)
│   ├── dashboard/          # HTML dashboard generation
│   │   ├── dashboard.go    # Template execution
│   │   └── templates/      # Embedded HTML templates
│   ├── history/            # Score history tracking
│   │   └── history.go      # JSON lines format handling
│   ├── baseline/           # Baseline storage for regression detection
│   │   └── baseline.go     # Git ref and file storage
│   ├── checks/             # CI platform check integrations
│   │   └── github.go       # GitHub Checks API client
│   └── sources/            # External source modules
│       ├── source.go       # Source interface and registry
│       ├── config.go       # Configuration resolver helper
│       ├── httpclient/     # Common HTTP client
│       │   ├── client.go   # Configurable HTTP client (auth, headers)
│       │   ├── url.go      # URL normalization utilities
│       │   └── github.go   # GitHub API config helpers
│       ├── repoparse/      # Repository identifier parsing
│       │   └── repoparse.go # Parse "owner/repo" format
│       ├── scoring/        # Shared scoring utilities
│       │   ├── scoring.go  # SeverityScore calculation
│       │   ├── factors.go  # Factor building helpers
│       │   └── report.go   # Report building helper
│       ├── cmdrun/         # CLI command execution
│       │   └── cmdrun.go   # Run() and FormatError()
│       ├── sonarqube/      # SonarQube implementation
│       ├── codecov/        # Codecov implementation
│       ├── ghactions/      # GitHub Actions implementation
│       ├── snyk/           # Snyk implementation
│       ├── dependabot/     # GitHub Dependabot implementation
│       ├── trivy/          # Trivy implementation
│       ├── grype/          # Grype implementation
│       └── semgrep/        # Semgrep implementation
└── testdata/               # Test fixtures
```

## Data Flow

```
┌─────────────────┐     ┌─────────────────┐
│ confidence.json │     │ External Source │
└────────┬────────┘     │ (SonarQube etc) │
         │              └────────┬────────┘
         │ Parse                 │ Fetch
         │              ┌────────┘
         ▼              ▼
      ┌─────────────────┐
      │ confidence.     │
      │   Report        │
      └────────┬────────┘
               │
     ┌─────────┼─────────┐
     │         │         │
     ▼         ▼         ▼
 ┌───────┐ ┌───────────┐ ┌────────┐
 │ gauge │ │ dashboard │ │ checks │
 └───┬───┘ └─────┬─────┘ └───┬────┘
     │           │           │
     ▼           ▼           ▼
 ┌───────┐ ┌───────────┐ ┌────────────┐
 │  SVG  │ │   HTML    │ │ GitHub API │
 └───────┘ └───────────┘ └────────────┘
```

## Package Responsibilities

### `cmd/confvis`

Entry point only. Delegates to `internal/cli.Execute()`.

### `internal/cli`

Defines CLI structure using [Cobra](https://github.com/spf13/cobra):
- Parses command-line flags
- Orchestrates file I/O
- Calls appropriate generators

### `internal/confidence`

Core types and JSON parsing:
- `Report` - Overall report with title, score, threshold
- `Factor` - Individual contributing factor
- `ParseFile()` - Read and validate JSON

### `internal/gauge`

SVG gauge generation:
- Uses [svgo](https://github.com/ajstarks/svgo) for SVG rendering
- Renders semi-circle arc gauge
- Supports light/dark color schemes
- Shows pass/fail status

### `internal/dashboard`

HTML dashboard generation:
- Uses Go's `html/template` with embedded templates
- Renders full report with factor breakdown
- Embeds the gauge SVG inline

### `internal/baseline`

Baseline storage for regression detection in CI/CD:
- `Baseline` - Extends Report with save metadata (timestamp, commit, branch)
- Git ref storage - Stores baselines in git refs without commits
- File storage - Alternative for non-git environments
- Used by `confvis baseline save/show` and `--compare-baseline`

### `internal/checks`

CI platform check integrations:
- `GitHubClient` - Creates GitHub Check Runs via the Checks API
- Supports pass/fail status based on threshold
- Generates markdown summary with factor breakdown
- Auto-detects GitHub Actions environment variables

### `internal/sources`

Modular framework for fetching metrics from external systems:
- `Source` interface defines the contract for all sources
- Registry allows auto-registration via `init()`
- Each source is a sub-package (e.g., `sources/sonarqube`)
- `ConfigResolver` helper consolidates token/URL resolution from options and environment variables

### `internal/sources/httpclient`

Common HTTP client used by API-based sources:
- Configurable authentication: Bearer, Token, Basic, or None
- Custom Accept headers and extra headers (e.g., API version)
- Centralized error handling and JSON decoding
- `NormalizeBaseURL()` helper for consistent URL handling
- `GitHubConfig()` and `GitHubConfigWithVersion()` for GitHub API clients
- Reduces duplication across source clients

### `internal/sources/repoparse`

Repository identifier parsing utilities:
- `Parse()` - Split "owner/repo" into separate parts with validation
- `MustParse()` - Like Parse but returns empty strings on error (for URL builders)
- Used by GitHub-based sources (ghactions, dependabot) and codecov

### `internal/sources/scoring`

Shared scoring utilities used by vulnerability/issue sources:
- `SeverityScore()` - Calculate score from issue count and penalty
- `SeverityCounts` - Standard struct for critical/high/medium/low counts
- `VulnSeverityConfigs()` - Build severity config arrays for standard vulnerability sources
- `BuildSeverityFactors()` - Generate confidence factors from severity configs
- `BuildReport()` - Create complete report with calculated score

### `internal/sources/cmdrun`

CLI command execution utilities for command-based sources:
- `Run()` - Execute commands with context, handling compound commands
- `FormatError()` - Format errors with stderr output for debugging
- Used by trivy, grype, and semgrep sources

### `internal/sources/sonarqube`

SonarQube integration:
- Uses common `httpclient` with Basic auth
- Maps SonarQube metrics to confidence factors
- Converts A-E ratings to 0-100 scores

### `internal/sources/codecov`

Codecov integration:
- Uses common `httpclient` with Bearer auth
- Fetches coverage percentage from Codecov API
- Supports GitHub, GitLab, and Bitbucket providers

### `internal/sources/ghactions`

GitHub Actions integration:
- Uses common `httpclient` with Bearer auth and GitHub API headers
- Calculates workflow success rate from recent runs
- Supports GitHub Enterprise via custom API URL

### `internal/sources/snyk`

Snyk integration:
- Uses common `httpclient` with Token auth
- Converts vulnerability counts to severity-weighted scores
- Requires organization ID and project ID

### `internal/sources/trivy`

Trivy integration:
- Runs local Trivy scanner (not HTTP-based)
- Parses JSON output to extract vulnerability counts
- Supports custom Trivy command (e.g., Docker)

## Extension Points

### Adding New Output Formats

1. Create new package under `internal/` (e.g., `internal/markdown`)
2. Implement `Generate(w io.Writer, report *confidence.Report, opts Options) error`
3. Add CLI command in `internal/cli/`

### Adding New Color Schemes

1. Add scheme function in `internal/gauge/colors.go`
2. Update `Options` to support scheme selection
3. Add CLI flag if desired

### Custom Gauge Styles

The gauge rendering is self-contained in `gauge.go`. Modify:
- Arc geometry calculations
- Text positioning
- Stroke widths and styles

### Adding New Sources

1. Create new package under `internal/sources/` (e.g., `internal/sources/newsource`)
2. Implement the `Source` interface:
   ```go
   type Source interface {
       Name() string
       Fetch(ctx context.Context, opts Options) (*confidence.Report, error)
   }
   ```
3. Register in `init()`:
   ```go
   func init() {
       sources.Register(&Source{})
   }
   ```
4. Import the package in `internal/cli/fetch.go` with blank import:
   ```go
   _ "github.com/boinger/confvis/internal/sources/newsource"
   ```

The source will automatically be available via `confvis fetch newsource`.

#### Using Common Helpers

For API-based sources, use the shared infrastructure:

**HTTP Client** (`internal/sources/httpclient`):
```go
import "github.com/boinger/confvis/internal/sources/httpclient"

client := httpclient.New(httpclient.Config{
    BaseURL:  "https://api.example.com",
    Token:    token,
    AuthType: httpclient.AuthBearer, // or AuthToken, AuthBasic, AuthNone
    Accept:   "application/json",
    Timeout:  30 * time.Second,
})

var result ResponseType
err := client.Get(ctx, endpoint, &result)
```

**Configuration Resolver** (`internal/sources/config.go`):
```go
var configResolver = &sources.ConfigResolver{
    SourceName:     "newsource",
    TokenEnvVar:    "NEWSOURCE_TOKEN",
    URLEnvVar:      "NEWSOURCE_URL",
    TokenRequired:  true,
    URLRequired:    false,
    DefaultTimeout: 30 * time.Second,
}

func (s *Source) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
    cfg, err := configResolver.Resolve(opts)
    if err != nil {
        return nil, err
    }
    // cfg.Token, cfg.URL, cfg.Timeout are now resolved
    client := NewClient(cfg.URL, cfg.Token, cfg.Timeout)
    // ...
}
```

#### Testability Pattern

For testable source implementations, use the `Fetcher` interface pattern:

1. Define a `Fetcher` interface that your HTTP client satisfies:
   ```go
   type Fetcher interface {
       FetchData(ctx context.Context, params string) (*Response, error)
       DataURL(params string) string
   }
   ```

2. Add a `FetchWithClient` method that accepts the interface:
   ```go
   func (s *Source) FetchWithClient(ctx context.Context, fetcher Fetcher, opts sources.Options) (*confidence.Report, error) {
       // Core logic here - fully testable with mock fetcher
   }
   ```

3. Have `Fetch` resolve configuration and delegate:
   ```go
   func (s *Source) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
       // Resolve token, URL, etc. from opts and environment
       client := NewClient(apiURL, token, timeout)
       return s.FetchWithClient(ctx, client, opts)
   }
   ```

This pattern (similar to `*WithFS` in `internal/cli/`) allows injecting mock clients in tests while keeping the public API unchanged. The existing `*Client` types automatically satisfy their `Fetcher` interfaces via Go's structural typing.
