// Package coverage provides shared infrastructure for coverage source providers.
package coverage

import (
	"math"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/scoring"
)

// BuildReport creates a coverage confidence report with a single coverage factor.
// This consolidates the common pattern used by coverage providers (coveralls, codecov).
func BuildReport(title, sourceName string, threshold int, coveragePercent float64, reportURL string) (*confidence.Report, error) {
	factors := []confidence.Factor{
		{
			Name:   "Code Coverage",
			Score:  int(math.Round(coveragePercent)),
			Weight: 100,
			URL:    reportURL,
		},
	}
	return scoring.BuildReport(title, sourceName, threshold, factors)
}

// ResolveService extracts the service from options, defaulting to "github".
func ResolveService(opts sources.Options) string {
	return sources.GetExtra(opts, "service", "github")
}
