package cli

import (
	"testing"

	"github.com/boinger/confvis/internal/checks"
)

func TestParseRepoFromEnv(t *testing.T) {
	tests := []struct {
		name         string
		currentOwner string
		currentRepo  string
		env          *checks.GitHubEnv
		wantOwner    string
		wantRepo     string
	}{
		{
			name:         "already has both",
			currentOwner: "myowner",
			currentRepo:  "myrepo",
			env:          &checks.GitHubEnv{Repository: "other/repo"},
			wantOwner:    "myowner",
			wantRepo:     "myrepo",
		},
		{
			name:      "nil env",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "empty repository",
			env:       &checks.GitHubEnv{Repository: ""},
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "valid repository",
			env:       &checks.GitHubEnv{Repository: "owner/repo"},
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:         "partial owner",
			currentOwner: "myowner",
			env:          &checks.GitHubEnv{Repository: "envowner/envrepo"},
			wantOwner:    "myowner",
			wantRepo:     "envrepo",
		},
		{
			name:        "partial repo",
			currentRepo: "myrepo",
			env:         &checks.GitHubEnv{Repository: "envowner/envrepo"},
			wantOwner:   "envowner",
			wantRepo:    "myrepo",
		},
		{
			name:      "invalid format",
			env:       &checks.GitHubEnv{Repository: "invalid-format"},
			wantOwner: "",
			wantRepo:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo := ParseRepoFromEnv(tt.currentOwner, tt.currentRepo, tt.env)
			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}
