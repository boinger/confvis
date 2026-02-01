# CLI Reference

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--verbose` | `-v` | Enable verbose output |
| `--quiet` | `-q` | Suppress non-error output (overrides `-v`) |
| `--help` | `-h` | Show help |

## Commands

### `confvis generate`

Generate both an SVG badge and HTML dashboard from a confidence report.

```bash
confvis generate -c <config> -o <output-dir> [flags]
```

#### Required Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to confidence report JSON file, or `-` for stdin |
| `--output` | `-o` | Output directory path |

#### Optional Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--dark` | false | Use dark mode colors |
| `--fail-under` | 0 | Exit with code 1 if score is below this value |

#### Output

Creates:
- `<output>/badge.svg` - SVG gauge badge
- `<output>/dashboard/index.html` - HTML dashboard

#### Examples

```bash
# Basic usage
confvis generate -c confidence.json -o ./output

# With dark mode
confvis generate -c confidence.json -o ./output --dark

# Verbose output
confvis generate -c confidence.json -o ./output -v

# Read from stdin
cat confidence.json | confvis generate -c - -o ./output

# Fail if score below threshold (CI/CD)
confvis generate -c confidence.json -o ./output --fail-under 75

# Quiet mode (no output on success)
confvis generate -c confidence.json -o ./output -q
```

---

### `confvis gauge`

Generate an SVG gauge badge from a confidence report.

```bash
confvis gauge -c <config> -o <output-file> [flags]
```

#### Required Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to confidence report JSON file, or `-` for stdin |
| `--output` | `-o` | Output SVG file path, or `-` for stdout |

#### Optional Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--width` | 200 | Gauge width in pixels |
| `--height` | 120 | Gauge height in pixels |
| `--dark` | false | Use dark mode colors |
| `--fail-under` | 0 | Exit with code 1 if score is below this value |

#### Examples

```bash
# Basic usage
confvis gauge -c confidence.json -o badge.svg

# Custom dimensions
confvis gauge -c confidence.json -o badge.svg --width 300 --height 180

# Dark mode
confvis gauge -c confidence.json -o badge.svg --dark

# Read from stdin
cat confidence.json | confvis gauge -c - -o badge.svg

# Write to stdout
confvis gauge -c confidence.json -o -

# Pipe through (stdin to stdout)
cat confidence.json | confvis gauge -c - -o - > badge.svg

# Fail if score below threshold (CI/CD)
confvis gauge -c confidence.json -o badge.svg --fail-under 75

# Quiet mode (no output on success)
confvis gauge -c confidence.json -o badge.svg -q
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (invalid config, file not found, or score below `--fail-under` threshold) |

## Examples

### CI/CD Usage

```bash
# Generate badge, fail build if config is invalid
confvis gauge -c confidence.json -o badge.svg || exit 1

# Fail if score drops below 75
confvis gauge -c confidence.json -o badge.svg --fail-under 75

# Quiet mode for clean CI logs
confvis generate -c confidence.json -o ./output --fail-under 75 -q

# Pipe from another tool
metrics-tool export | confvis gauge -c - -o badge.svg --fail-under 80
```

### Verbose Debugging

```bash
confvis generate -c confidence.json -o ./output -v
# Output:
# Generating output for "Code Quality" (score: 85, threshold: 75)
# Wrote badge to ./output/badge.svg
# Wrote dashboard to ./output/dashboard/index.html
```
