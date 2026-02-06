package dependabot

import (
	"encoding/json"

	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/githubalerts"
	"github.com/boinger/confvis/internal/sources/scoring"
)

func init() {
	sources.Register(githubalerts.NewSource(githubalerts.SourceConfig{
		Name:         "dependabot",
		TokenEnvVar:  "DEPENDABOT_TOKEN", // #nosec G101 -- not a credential, just env var name
		EndpointPath: "dependabot/alerts",
		WebURLPath:   "security/dependabot",
	}, countAlerts, nil))
}

// countAlerts extracts severity counts from Dependabot alerts JSON.
func countAlerts(data []byte) (scoring.SeverityCounts, error) {
	var alerts AlertsResponse
	if err := json.Unmarshal(data, &alerts); err != nil {
		return scoring.SeverityCounts{}, err
	}

	var counts scoring.SeverityCounts
	for _, alert := range alerts {
		scoring.CountSeverity(&counts, alert.SecurityAdvisory.Severity, "dependabot", true)
	}
	return counts, nil
}
