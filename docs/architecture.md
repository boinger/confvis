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
│   │   └── fetch.go        # fetch subcommand (external sources)
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
│   └── sources/            # External source modules
│       ├── source.go       # Source interface and registry
│       └── sonarqube/      # SonarQube implementation
│           ├── sonarqube.go
│           ├── client.go
│           └── types.go
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
          ┌────┴────┐
          │         │
          ▼         ▼
      ┌───────┐ ┌───────────┐
      │ gauge │ │ dashboard │
      └───┬───┘ └─────┬─────┘
          │           │
          ▼           ▼
      ┌───────┐ ┌───────────┐
      │  SVG  │ │   HTML    │
      └───────┘ └───────────┘
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

### `internal/sources`

Modular framework for fetching metrics from external systems:
- `Source` interface defines the contract for all sources
- Registry allows auto-registration via `init()`
- Each source is a sub-package (e.g., `sources/sonarqube`)

### `internal/sources/sonarqube`

SonarQube integration:
- HTTP client with Basic auth support
- Maps SonarQube metrics to confidence factors
- Converts A-E ratings to 0-100 scores

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

1. Create new package under `internal/sources/` (e.g., `internal/sources/codecov`)
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
   _ "github.com/boinger/confvis/internal/sources/codecov"
   ```

The source will automatically be available via `confvis fetch codecov`.
