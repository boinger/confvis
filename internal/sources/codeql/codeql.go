package codeql

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/githubalerts"
	"github.com/boinger/confvis/internal/sources/scoring"
)

func init() {
	sources.Register(githubalerts.NewSource(githubalerts.SourceConfig{
		Name:         "codeql",
		TokenEnvVar:  "CODEQL_TOKEN", // #nosec G101 -- not a credential, just env var name
		EndpointPath: "code-scanning/alerts",
		WebURLPath:   "security/code-scanning",
	}, countAlerts, extraParams))
}

// extraParams adds the optional tool_name filter parameter.
func extraParams(opts sources.Options) url.Values {
	toolName := sources.GetExtra(opts, "tool", "")
	if toolName == "" {
		return nil
	}
	return url.Values{"tool_name": {toolName}}
}

// countAlerts extracts severity counts from CodeQL alerts JSON.
// CodeQL alerts have a security_severity_level field (critical, high, medium, low).
// If that's not set, we fall back to the rule severity (error->high, warning->medium, note->low).
func countAlerts(data []byte) (scoring.SeverityCounts, error) {
	var alerts AlertsResponse
	if err := json.Unmarshal(data, &alerts); err != nil {
		return scoring.SeverityCounts{}, err
	}

	var counts scoring.SeverityCounts
	for _, alert := range alerts {
		// Prefer security_severity_level if available
		severity := strings.ToLower(alert.Rule.SecuritySeverityLevel)
		if severity == "" {
			// Fall back to rule severity
			switch strings.ToLower(alert.Rule.Severity) {
			case "error":
				severity = "high"
			case "warning":
				severity = "medium"
			default:
				severity = "low"
			}
		}
		scoring.CountSeverity(&counts, severity, "codeql", true)
	}
	return counts, nil
}
