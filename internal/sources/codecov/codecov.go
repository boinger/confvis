package codecov

import (
	"encoding/json"
	"fmt"

	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/coverage"
)

func init() {
	sources.Register(coverage.NewSource(coverage.SourceConfig{
		Name:          "codecov",
		TokenEnvVar:   "CODECOV_TOKEN", // #nosec G101 -- not a credential, just env var name
		TokenRequired: true,
		BaseURL:       "https://api.codecov.io",
		BuildAPIPath:  func(service, owner, repo string) string { return fmt.Sprintf("/api/v2/%s/%s/repos/%s/report/", service, owner, repo) },
		BuildWebURL:   func(service, owner, repo string) string { return fmt.Sprintf("https://app.codecov.io/%s/%s/%s", service, owner, repo) },
	}, extractCoverage))
}

// extractCoverage extracts the coverage percentage from Codecov API response.
func extractCoverage(data []byte) (float64, error) {
	var resp ReportResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return 0, err
	}
	return resp.Totals.Coverage, nil
}
