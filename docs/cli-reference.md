# CLI Reference

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--verbose` | `-v` | Enable verbose output |
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
| `--config` | `-c` | Path to confidence report JSON file |
| `--output` | `-o` | Output directory path |

#### Optional Flags

| Flag | Description |
|------|-------------|
| `--dark` | Use dark mode colors |

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
| `--config` | `-c` | Path to confidence report JSON file |
| `--output` | `-o` | Output SVG file path |

#### Optional Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--width` | 200 | Gauge width in pixels |
| `--height` | 120 | Gauge height in pixels |
| `--dark` | false | Use dark mode colors |

#### Examples

```bash
# Basic usage
confvis gauge -c confidence.json -o badge.svg

# Custom dimensions
confvis gauge -c confidence.json -o badge.svg --width 300 --height 180

# Dark mode
confvis gauge -c confidence.json -o badge.svg --dark
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (invalid config, file not found, etc.) |

## Examples

### CI/CD Usage

```bash
# Generate badge, fail build if config is invalid
confvis gauge -c confidence.json -o badge.svg || exit 1
```

### Verbose Debugging

```bash
confvis generate -c confidence.json -o ./output -v
# Output:
# Generating output for "Code Quality" (score: 85, threshold: 75)
# Wrote badge to ./output/badge.svg
# Wrote dashboard to ./output/dashboard/index.html
```
