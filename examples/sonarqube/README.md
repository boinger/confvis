# SonarQube Integration

This example shows how to fetch metrics from SonarQube and generate confidence badges.

## Prerequisites

- A SonarQube server with your project analyzed
- A SonarQube API token (User > My Account > Security > Tokens)

## Local Usage

```bash
# Set environment variables
export SONARQUBE_URL=https://sonarqube.example.com
export SONARQUBE_TOKEN=squ_xxxxxxxxxxxxxxxxxxxx

# Fetch metrics and generate badge
confvis fetch sonarqube -p your-project-key -o confidence.json
confvis gauge -c confidence.json -o badge.svg

# Or in one pipeline
confvis fetch sonarqube -p your-project-key -o - | confvis gauge -c - -o badge.svg
```

## GitHub Actions

See `workflow.yml` for a complete GitHub Actions workflow.

Required secrets:
- `SONARQUBE_URL`: Your SonarQube server URL
- `SONARQUBE_TOKEN`: API token for authentication

## Metric Mapping

| SonarQube | confvis Factor | Weight |
|-----------|----------------|--------|
| Coverage | Test Coverage | 25% |
| Reliability Rating | Reliability | 25% |
| Security Rating | Security | 25% |
| Maintainability Rating | Maintainability | 25% |

Ratings (A-E) are converted to scores (100-0).

## Customization

Override the default threshold:

```bash
confvis fetch sonarqube -p myproject --threshold 80 -o confidence.json
```

Use a custom title:

```bash
confvis fetch sonarqube -p myproject --title "Backend API" -o confidence.json
```

Query a specific branch:

```bash
confvis fetch sonarqube -p myproject --branch develop -o confidence.json
```
