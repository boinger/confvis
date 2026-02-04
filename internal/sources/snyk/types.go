// Package snyk provides a source for fetching vulnerability metrics from Snyk.
package snyk

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
	LatestIssueCounts *IssueCounts `json:"latest_issue_counts,omitempty"`
}

// IssueCounts contains vulnerability counts by severity.
type IssueCounts struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
}

