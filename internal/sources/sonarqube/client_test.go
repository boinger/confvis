package sonarqube

import (
	"net/http"
	"testing"
	"time"
)

func TestNewClient_EmptyBaseURL(t *testing.T) {
	// SonarQube has no default URL; empty stays empty.
	c := NewClient("", "token", 5*time.Second)
	if c.baseURL != "" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "")
	}
}

func TestNewClient_CustomBaseURL(t *testing.T) {
	c := NewClient("https://sonar.example.com", "token", 5*time.Second)
	if c.baseURL != "https://sonar.example.com" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://sonar.example.com")
	}
}

func TestNewClientWithHTTP_EmptyBaseURL(t *testing.T) {
	c := NewClientWithHTTP("", "token", http.DefaultClient)
	if c.baseURL != "" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "")
	}
}

func TestNewClientWithHTTP_CustomBaseURL(t *testing.T) {
	c := NewClientWithHTTP("https://sonar.example.com", "token", http.DefaultClient)
	if c.baseURL != "https://sonar.example.com" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://sonar.example.com")
	}
}

func TestNewClientWithHTTP_TrimsTrailingSlash(t *testing.T) {
	c := NewClientWithHTTP("https://sonar.example.com/", "token", http.DefaultClient)
	if c.baseURL != "https://sonar.example.com" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "https://sonar.example.com")
	}
}

func TestClient_MeasureURL_NoBranch(t *testing.T) {
	c := NewClient("https://sonar.example.com", "", 5*time.Second)

	got := c.MeasureURL("myproject", "coverage", "")
	want := "https://sonar.example.com/component_measures?id=myproject&metric=coverage"
	if got != want {
		t.Errorf("MeasureURL() = %q, want %q", got, want)
	}
}

func TestClient_MeasureURL_SpecialCharacters(t *testing.T) {
	c := NewClient("https://sonar.example.com", "", 5*time.Second)

	got := c.MeasureURL("org:project", "coverage", "feature/branch")
	want := "https://sonar.example.com/component_measures?id=org%3Aproject&metric=coverage&branch=feature%2Fbranch"
	if got != want {
		t.Errorf("MeasureURL() = %q, want %q", got, want)
	}
}
