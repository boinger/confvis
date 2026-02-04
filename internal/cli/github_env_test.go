package cli

import (
	"testing"

	"github.com/boinger/confvis/internal/checks"
)

func TestParseRepoFromEnv_AlreadyHasBoth(t *testing.T) {
	env := &checks.GitHubEnv{Repository: "other/repo"}
	owner, repo := ParseRepoFromEnv("myowner", "myrepo", env)

	if owner != "myowner" {
		t.Errorf("expected owner 'myowner', got %q", owner)
	}
	if repo != "myrepo" {
		t.Errorf("expected repo 'myrepo', got %q", repo)
	}
}

func TestParseRepoFromEnv_NilEnv(t *testing.T) {
	owner, repo := ParseRepoFromEnv("", "", nil)

	if owner != "" {
		t.Errorf("expected empty owner, got %q", owner)
	}
	if repo != "" {
		t.Errorf("expected empty repo, got %q", repo)
	}
}

func TestParseRepoFromEnv_EmptyRepository(t *testing.T) {
	env := &checks.GitHubEnv{Repository: ""}
	owner, repo := ParseRepoFromEnv("", "", env)

	if owner != "" {
		t.Errorf("expected empty owner, got %q", owner)
	}
	if repo != "" {
		t.Errorf("expected empty repo, got %q", repo)
	}
}

func TestParseRepoFromEnv_ValidRepository(t *testing.T) {
	env := &checks.GitHubEnv{Repository: "owner/repo"}
	owner, repo := ParseRepoFromEnv("", "", env)

	if owner != "owner" {
		t.Errorf("expected owner 'owner', got %q", owner)
	}
	if repo != "repo" {
		t.Errorf("expected repo 'repo', got %q", repo)
	}
}

func TestParseRepoFromEnv_PartialOwner(t *testing.T) {
	env := &checks.GitHubEnv{Repository: "envowner/envrepo"}
	owner, repo := ParseRepoFromEnv("myowner", "", env)

	if owner != "myowner" {
		t.Errorf("expected owner 'myowner', got %q", owner)
	}
	if repo != "envrepo" {
		t.Errorf("expected repo 'envrepo', got %q", repo)
	}
}

func TestParseRepoFromEnv_PartialRepo(t *testing.T) {
	env := &checks.GitHubEnv{Repository: "envowner/envrepo"}
	owner, repo := ParseRepoFromEnv("", "myrepo", env)

	if owner != "envowner" {
		t.Errorf("expected owner 'envowner', got %q", owner)
	}
	if repo != "myrepo" {
		t.Errorf("expected repo 'myrepo', got %q", repo)
	}
}

func TestParseRepoFromEnv_InvalidFormat(t *testing.T) {
	env := &checks.GitHubEnv{Repository: "invalid-format"}
	owner, repo := ParseRepoFromEnv("", "", env)

	// On parse error, should return original values
	if owner != "" {
		t.Errorf("expected empty owner on parse error, got %q", owner)
	}
	if repo != "" {
		t.Errorf("expected empty repo on parse error, got %q", repo)
	}
}
