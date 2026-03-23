package snyk

import (
	"net/http"
	"testing"
	"time"
)

func TestNewClient_CustomBaseURL(t *testing.T) {
	c := NewClient("https://custom.snyk.io", "token", 5*time.Second)
	if c.baseURL != "https://custom.snyk.io" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.snyk.io")
	}
}

func TestNewClientWithHTTP_DefaultBaseURL(t *testing.T) {
	c := NewClientWithHTTP("", "token", http.DefaultClient)
	if c.baseURL != defaultBaseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, defaultBaseURL)
	}
}

func TestNewClientWithHTTP_CustomBaseURL(t *testing.T) {
	c := NewClientWithHTTP("https://custom.snyk.io", "token", http.DefaultClient)
	if c.baseURL != "https://custom.snyk.io" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.snyk.io")
	}
}

func TestNewClientWithHTTP_TrimsTrailingSlash(t *testing.T) {
	c := NewClientWithHTTP("https://custom.snyk.io/", "token", http.DefaultClient)
	if c.baseURL != "https://custom.snyk.io" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://custom.snyk.io")
	}
}

func TestClient_ProjectURL_SpecialCharacters(t *testing.T) {
	c := NewClient("", "token", 5*time.Second)

	// ProjectURL does not URL-encode; it builds a simple path.
	got := c.ProjectURL("org-with-dash", "project-with-dash")
	want := "https://app.snyk.io/org/org-with-dash/project/project-with-dash"
	if got != want {
		t.Errorf("ProjectURL() = %q, want %q", got, want)
	}
}
