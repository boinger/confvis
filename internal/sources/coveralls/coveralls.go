package coveralls

import (
	"encoding/json"
	"fmt"

	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/coverage"
)

func init() {
	sources.Register(coverage.NewSource(coverage.SourceConfig{
		Name:          "coveralls",
		TokenEnvVar:   "COVERALLS_TOKEN",
		TokenRequired: false, // Public repos don't require a token
		BaseURL:       "https://coveralls.io",
		BuildAPIPath:  func(service, owner, repo string) string { return fmt.Sprintf("/%s/%s/%s.json", service, owner, repo) },
		BuildWebURL:   func(service, owner, repo string) string { return fmt.Sprintf("https://coveralls.io/%s/%s/%s", service, owner, repo) },
	}, extractCoverage))
}

// extractCoverage extracts the coverage percentage from Coveralls API response.
func extractCoverage(data []byte) (float64, error) {
	var resp ReportResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return 0, err
	}
	return resp.CoveredPercent, nil
}
