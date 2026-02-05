// Package snyk provides a source for fetching vulnerability metrics from Snyk.
package snyk

import "github.com/boinger/confvis/internal/sources/scoring"

// ProjectResponse represents the response from /rest/orgs/{org_id}/projects/{project_id}.
type ProjectResponse struct {
	Data ProjectData `json:"data"`
}

// ProjectData contains the project information.
type ProjectData struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Attributes ProjectAttributes `json:"attributes"`
	Meta       ProjectMeta       `json:"meta,omitempty"`
}

// ProjectAttributes contains the project's attributes.
type ProjectAttributes struct {
	Name string `json:"name"`
}

// ProjectMeta contains metadata about the project including issue counts.
type ProjectMeta struct {
	LatestIssueCounts *scoring.SeverityCounts `json:"latest_issue_counts,omitempty"`
}

