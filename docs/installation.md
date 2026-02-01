# Installation

## Using `go install`

The simplest way to install confvis:

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
  gauge       Generate an SVG gauge badge
  generate    Generate badge and dashboard
  help        Help about any command

Flags:
  -h, --help      help for confvis
  -v, --verbose   verbose output
```

## Requirements

- Go 1.21 or later (for building)
- No runtime dependencies - the binary is self-contained
