# Installation

## GitHub Action (Recommended for CI/CD)

For GitHub Actions workflows, use the native action - no installation required:

```yaml
- uses: boinger/confvis@v1
  with:
    config: confidence.json
    output: badge.svg
```

See [GitHub Action Documentation](github-action.md) for all options and examples.

## Using `go install`

For local development or non-GitHub CI environments:

```bash
go install github.com/boinger/confvis/cmd/confvis@latest
```

This installs the binary to your `$GOPATH/bin` (or `$GOBIN`). Ensure this directory is in your `$PATH`.

## Building from Source

Clone and build:

```bash
git clone https://github.com/boinger/confvis.git
cd confvis
go build -o confvis ./cmd/confvis
```

Install to `$GOPATH/bin`:

```bash
go install ./cmd/confvis
```

## Verifying Installation

```bash
confvis --help
```

Expected output:

```
confvis generates visual representations of confidence scores.

It reads JSON confidence reports and produces:
- SVG gauge badges showing the overall score
- HTML dashboards with detailed factor breakdowns

Usage:
  confvis [command]

Available Commands:
  aggregate   Aggregate multiple reports into a single dashboard
  baseline    Manage confidence baselines
  comment     Post confidence report as a comment
  completion  Generate the autocompletion script for the specified shell
  fetch       Fetch metrics from an external source
  gate        CI gate: check thresholds and exit non-zero on failure
  gauge       Generate an SVG gauge badge
  generate    Generate badge and dashboard
  help        Help about any command
  version     Print the version number

Flags:
  -h, --help      help for confvis
  -q, --quiet     suppress non-error output
  -v, --verbose   verbose output
      --version   version for confvis

Use "confvis [command] --help" for more information about a command.
```

## Requirements

- Go 1.23 or later (for building)
- No runtime dependencies - the binary is self-contained
