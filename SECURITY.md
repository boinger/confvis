# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please report it responsibly.

### How to Report

**Do NOT open a public GitHub issue for security vulnerabilities.**

Instead, please email security concerns to the maintainers. You can find contact information in the repository's profile or reach out via GitHub's private vulnerability reporting feature if available.

### What to Include

When reporting a vulnerability, please include:

1. **Description**: A clear description of the vulnerability
2. **Steps to Reproduce**: Detailed steps to reproduce the issue
3. **Impact Assessment**: Your assessment of the potential impact
4. **Affected Versions**: Which versions are affected (if known)
5. **Suggested Fix**: Any suggestions for remediation (optional)

### Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 7 days
- **Resolution Timeline**:
  - Critical: Target fix within 7 days
  - High: Target fix within 30 days
  - Medium/Low: Target fix within 90 days

### What to Expect

1. **Acknowledgment**: We'll acknowledge receipt of your report
2. **Investigation**: We'll investigate and validate the vulnerability
3. **Resolution**: We'll work on a fix and coordinate disclosure
4. **Credit**: We'll credit you in the release notes (unless you prefer anonymity)

## Security Measures

This project employs several security measures:

### Automated Scanning

- **CodeQL**: Static analysis for security vulnerabilities
- **Trivy**: Vulnerability scanning for dependencies
- **Dependabot**: Automated dependency updates

### Development Practices

- All code changes require pull request review
- CI/CD pipelines run security scans on every PR
- Dependencies are regularly updated
- Strict linting with zero-tolerance for issues

## Scope

This security policy applies to:

- The confvis CLI tool
- All official confvis source integrations
- Documentation and examples

Out of scope:

- Third-party integrations not maintained by this project
- Issues in upstream dependencies (please report these to the relevant projects)

## Hall of Fame

We appreciate the security researchers who help keep confvis secure. Contributors will be acknowledged here (with permission).
