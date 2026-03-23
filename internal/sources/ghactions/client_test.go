package ghactions

import (
	"net/http"
	"testing"
	"time"

	"github.com/boinger/confvis/internal/sources/httpclient"
)

func TestNewClient_CustomURL(t *testing.T) {
	c := NewClient("https://api.github.example.com", "token", 5*time.Second)
	if c.baseURL != "https://api.github.example.com" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://api.github.example.com")
	}
}

func TestNewClientWithHTTP_DefaultBaseURL(t *testing.T) {
	c := NewClientWithHTTP("", "token", http.DefaultClient)
	if c.baseURL != httpclient.GitHubDefaultURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, httpclient.GitHubDefaultURL)
	}
}

func TestNewClientWithHTTP_CustomURL(t *testing.T) {
	c := NewClientWithHTTP("https://api.github.example.com", "token", http.DefaultClient)
	if c.baseURL != "https://api.github.example.com" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://api.github.example.com")
	}
}

func TestNewClientWithHTTP_TrimsTrailingSlash(t *testing.T) {
	c := NewClientWithHTTP("https://api.github.example.com/", "token", http.DefaultClient)
	if c.baseURL != "https://api.github.example.com" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://api.github.example.com")
	}
}

func TestClient_ActionsURL_EmptyOwnerRepo(t *testing.T) {
	c := NewClientWithHTTP("", "token", http.DefaultClient)
	got := c.ActionsURL("")
	if got != "" {
		t.Errorf("ActionsURL(%q) = %q, want %q", "", got, "")
	}
}
