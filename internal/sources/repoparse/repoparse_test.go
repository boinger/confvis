package repoparse

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		project   string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "valid owner/repo",
			project:   "boinger/confvis",
			wantOwner: "boinger",
			wantRepo:  "confvis",
			wantErr:   false,
		},
		{
			name:      "valid with hyphen",
			project:   "my-org/my-repo",
			wantOwner: "my-org",
			wantRepo:  "my-repo",
			wantErr:   false,
		},
		{
			name:      "valid with nested slash",
			project:   "owner/repo/with/slashes",
			wantOwner: "owner",
			wantRepo:  "repo/with/slashes",
			wantErr:   false,
		},
		{
			name:    "empty string",
			project: "",
			wantErr: true,
		},
		{
			name:    "no slash",
			project: "justarepo",
			wantErr: true,
		},
		{
			name:    "empty owner",
			project: "/repo",
			wantErr: true,
		},
		{
			name:    "empty repo",
			project: "owner/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := Parse(tt.project)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if owner != tt.wantOwner {
					t.Errorf("Parse() owner = %v, want %v", owner, tt.wantOwner)
				}
				if repo != tt.wantRepo {
					t.Errorf("Parse() repo = %v, want %v", repo, tt.wantRepo)
				}
			}
		})
	}
}

func TestMustParse(t *testing.T) {
	tests := []struct {
		name      string
		project   string
		wantOwner string
		wantRepo  string
	}{
		{
			name:      "valid owner/repo",
			project:   "boinger/confvis",
			wantOwner: "boinger",
			wantRepo:  "confvis",
		},
		{
			name:      "empty string returns empty",
			project:   "",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "invalid format returns empty",
			project:   "noslash",
			wantOwner: "",
			wantRepo:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo := MustParse(tt.project)
			if owner != tt.wantOwner {
				t.Errorf("MustParse() owner = %v, want %v", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("MustParse() repo = %v, want %v", repo, tt.wantRepo)
			}
		})
	}
}
