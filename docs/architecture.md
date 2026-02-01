# Architecture

## Package Structure

```
confvis/
├── cmd/confvis/main.go     # Entry point, invokes CLI
├── internal/
│   ├── cli/                # Command definitions (cobra)
│   │   ├── root.go         # Root command setup
│   │   ├── gauge.go        # gauge subcommand
│   │   └── generate.go     # generate subcommand
│   ├── confidence/         # Core data types and parsing
│   │   ├── types.go        # Report, Factor structs
│   │   └── parser.go       # JSON parsing logic
│   ├── gauge/              # SVG gauge generation
│   │   ├── gauge.go        # SVG rendering with svgo
│   │   └── colors.go       # Color schemes (light/dark)
│   └── dashboard/          # HTML dashboard generation
│       ├── dashboard.go    # Template execution
│       └── templates/      # Embedded HTML templates
└── testdata/               # Test fixtures
```

## Data Flow

```
┌─────────────────┐
│ confidence.json │
└────────┬────────┘
         │ Parse
         ▼
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
