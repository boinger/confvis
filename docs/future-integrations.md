# Future Integration Candidates

Potential code quality and CI tool integrations for confvis, organized by category and priority.

---

## Medium Priority

### GitLab CI

**Category:** CI/CD
**API:** GitLab REST API
**Badge:** `https://img.shields.io/gitlab/pipeline-status/{owner}/{repo}`
**Why:** Major platform, many teams use GitLab instead of GitHub.

**Implementation notes:**
- Endpoint: `GET /projects/{id}/pipelines` and `/projects/{id}/jobs`
- Auth: Private token or OAuth
- Similar concepts to GitHub Actions (pipelines = workflows, jobs = runs)

**Scoring approach:** Success rate of recent pipelines, like ghactions

---

### OWASP Dependency-Check

**Category:** Vulnerability Scanning
**API:** CLI tool (JSON output)
**Badge:** No official shields.io badge (results are local)
**Why:** Free/OSS alternative to commercial scanners. Widely used in enterprise.

**Implementation notes:**
- CLI: `dependency-check --scan <path> --format JSON --out <output>`
- Output includes: dependencies with vulnerabilities, CVE details, severity (CVSS)
- CVSS scores map to severity levels

**Scoring approach:** Severity-weighted like grype/trivy

---

### npm audit / yarn audit

**Category:** Vulnerability Scanning (JavaScript)
**API:** CLI tools (JSON output)
**Badge:** Via Snyk — `https://img.shields.io/snyk/vulnerabilities/npm/{package}` (per-package)
**Why:** Direct package manager integration. Many projects are JS/TS.

**Implementation notes:**
- npm: `npm audit --json`
- yarn: `yarn audit --json`
- Output includes: advisories with severity, vulnerable versions, patched versions

**Scoring approach:** Severity-weighted (critical/high/moderate/low)

---

## Lower Priority

### Checkmarx

**Category:** SAST (Enterprise)
**API:** REST API
**Badge:** No public shields.io badge (enterprise/private)
**Why:** Common in enterprise environments. Usually requires license.

**Implementation notes:**
- REST API with OAuth authentication
- Scan results include: query name, severity, state, file, line
- May require polling for scan completion

**Scoring approach:** Severity-weighted

---

### Veracode

**Category:** SAST (Enterprise)
**API:** REST API
**Badge:** No public shields.io badge (enterprise/private)
**Why:** Another major enterprise SAST vendor.

**Implementation notes:**
- REST API with HMAC authentication
- Multiple scan types: static, dynamic, SCA
- Results include: flaw ID, severity, CWE, remediation

**Scoring approach:** Severity-weighted, possibly per scan type

---

### CircleCI

**Category:** CI/CD
**API:** REST API v2
**Badge:** `https://img.shields.io/circleci/build/github/{owner}/{repo}`
**Why:** Popular CI platform, especially for open source.

**Implementation notes:**
- Endpoint: `GET /project/{vcs}/{org}/{repo}/pipeline` and `/workflow/{id}/job`
- Auth: API token
- Similar concepts to GitHub Actions

**Scoring approach:** Pipeline/workflow success rate

---

### Jenkins

**Category:** CI/CD
**API:** REST API
**Badge:** `https://img.shields.io/jenkins/build?jobUrl={jenkins-url}/job/{job-name}`
**Why:** Still widely used, especially in enterprise/on-prem.

**Implementation notes:**
- Endpoint: `GET /job/{name}/api/json` with build history
- Auth: API token or username/password
- Build results: SUCCESS, FAILURE, UNSTABLE, ABORTED

**Scoring approach:** Success rate of recent builds

---

### FOSSA

**Category:** License Compliance
**API:** REST API
**Badge:** `https://app.fossa.com/api/projects/git%2Bgithub.com%2F{owner}%2F{repo}.svg?type=shield`
**Why:** License compliance is increasingly important for legal/procurement.

**Implementation notes:**
- REST API with API key
- Returns: license types, compliance issues, policy violations

**Scoring approach:** Binary (compliant/non-compliant) or count of issues

---

### Allure

**Category:** Test Reporting
**API:** File-based (JSON/XML) or Allure Server API
**Badge:** Via Allure TestOps (self-hosted badge URL)
**Why:** Rich test reporting, shows trends and history.

**Implementation notes:**
- Can read Allure report JSON directly
- Or query Allure TestOps server API
- Includes: test counts, pass/fail/skip, duration, history

**Scoring approach:** Test pass rate, possibly weighted by test importance

---

## Badge Summary

| Tool | Badge Available | shields.io Pattern |
|------|-----------------|-------------------|
| GitLab CI | Yes | `/gitlab/pipeline-status/{owner}/{repo}` |
| CircleCI | Yes | `/circleci/build/github/{owner}/{repo}` |
| Jenkins | Yes | `/jenkins/build?jobUrl={url}` |
| FOSSA | Yes | Via FOSSA API (custom URL) |
| npm/yarn | Yes | `/snyk/vulnerabilities/npm/{package}` (per-package) |
| Allure | Partial | Self-hosted only |
| OWASP Dep-Check | No | — |
| Checkmarx | No | Enterprise/private |
| Veracode | No | Enterprise/private |

---

## Implementation Patterns

All new sources should follow existing patterns:

1. **Package structure:** `internal/sources/<name>/`
   - `client.go` — API/CLI client
   - `types.go` — Data structures
   - `<name>.go` — Source implementation
   - `<name>_test.go` — Tests

2. **For API-based sources:** Use `httpclient` package, follow codecov/dependabot patterns

3. **For CLI-based sources:** Use `cmdrun` package, follow semgrep/grype/trivy patterns

4. **Scoring:** Use `scoring` package for severity-weighted calculations

5. **Testing:** Mock servers for APIs, mock scripts for CLIs
